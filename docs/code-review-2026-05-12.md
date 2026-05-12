# Code Review — 2026-05-12

审查范围：`backend/pkg/agentdef/`、`backend/pkg/skill/`、`backend/pkg/memory/`、`backend/pkg/a2ui/`、`backend/cmd/examples/`、`configs/agents/`

**总结：BLOCK — 存在 CRITICAL 级别问题，不建议直接上生产。**

---

## CRITICAL（必须修复）

### C1. 代码执行节点无沙箱 — `workflow_builder.go:200-271`

YAML 中的 `node.Code` 不经过沙箱，直接通过 `exec.Command` 运行 Python/Node/Bash。输入转义不完整（Python 三引号注入、bash 变量逃逸）。任何 YAML 文件都可触发宿主机任意命令执行。

**修复**：至少加 `env -i` 隔离环境变量 + 资源限制；长期方案用容器/WASM 沙箱。

### C2. `executeCodeNode` 丢弃调用方 context — `workflow_builder.go:248`

```go
// 现状（错误）
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

// 修复
ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
```

HTTP 断连或上游超时后子进程继续运行，goroutine 泄漏长达 10 秒。

### C3. `cacheKey` 哈希缺分隔符，存在碰撞 — `builder.go:638-643`

`("a","bc","d")` 和 `("ab","c","d")` 产生相同哈希，不同 session 可命中对方的缓存响应。

**修复**：字段间写入分隔符，如 `h.Write([]byte{0})`。

### C4. `EventStream.Send` 持锁阻塞在 channel send — `pkg/a2ui/stream.go:36-42`

channel 满时 `Send` 持 `mu.Lock()` 死等，此时 `Close` 尝试获锁直接死锁。

**修复**：将 channel send 移到锁外，或改用 atomic boolean 保护 `closed` 状态。

---

## HIGH（应在合并前修复）

| # | 问题 | 位置 |
|---|------|------|
| H1 | 所有 goroutine 向 channel 发送时无 `ctx.Done()` guard，HTTP 断连后 goroutine 泄漏 | `builder.go`, `orchestration.go`, `interrupt.go` 多处 |
| H2 | `InterruptableAgent` / `cacheAgent` 内存 map 无上限、无定期清理，高并发下无限增长 | `interrupt.go:92`, `builder.go:586` |
| H3 | `retryAgent` backoff `time.Sleep` 不响应 context 取消，最坏情况 700ms 无法中断 | `builder.go:499` |
| H4 | `builtin.DeleteState` 非原子读改写（HGetAll → Del → HSet），并发下数据丢失 | `pkg/memory/builtin/builtin.go:261` |
| H5 | Redis key 拼接未转义 `sessionID`/`userID`，存在 key 注入风险 | `pkg/memory/builtin/builtin.go:94,132` |
| H6 | Letta client URL path 中 `agentID` 未做 `url.PathEscape`，路径注入 | `pkg/memory/letta/client.go:155` 及多处 |
| H7 | UUID 生成用时间种子伪随机，文档注释声称使用 `crypto/rand`（不符） | `pkg/skill/builtin/builtin.go:156` |
| H8 | `two-pass buildAll` 替换整个 map，热重载失败会丢失已有 agent | `runtime.go:186` |
| H9 | 工作流 `spec.model.primary` 被运行时 `modelCfg.ModelID` 覆盖，YAML 声明静默失效 | `workflow_builder.go:134` |

---

## MEDIUM（影响正确性或可维护性）

| # | 问题 | 位置 |
|---|------|------|
| M1 | `builtin.Delete` 使用错误 Redis key（`Del` 而非 `HDel`），**Delete 是 no-op** | `pkg/memory/builtin/builtin.go:196` |
| M2 | `diffAndNotify` 每次热重载对所有 agent 触发 `ChangeUpdated`，无内容比较 | `reload.go:89` |
| M3 | `shallowCopyDefWithSystemPrompt` 共享 slice/map 字段，并发热重载存在数据竞争 | `builder.go:1097` |
| M4 | `CompositeInvoker` 本地调用失败时静默 fallback 到 HTTP，丢失原始错误 | `pkg/skill/invoker.go:146` |
| M5 | `json.Marshal` 错误在 SSE encoder 中被 `_` 丢弃，客户端收到空帧 | `pkg/a2ui/encoder.go:27` |
| M6 | `builtin.Search` 全量加载后客户端过滤，无分页上限，大数据集下 OOM 风险 | `pkg/memory/builtin/builtin.go:165` |
| M7 | Supervisor 动态 prompt 注入到 user 消息槽而非 system 槽，模型指令遵循度下降 | `orchestration.go:54` |
| M8 | `parseISO8601Millis` 在 mem0/zep/letta 三处重复，且实现有细微差异 | 三个 memory 包 |
| M9 | 示例代码硬编码 MiniMax vendor URL + 模型 ID，与 YAML 声明的 `gpt-4o` 不符 | `cmd/examples/document_pipeline/main.go:29` |
| M10 | 测试中 `apiKey = "123456"` 硬编码，触发 secret scanning 误报 | `runtime_test.go:116` |

---

## LOW

| # | 问题 | 位置 |
|---|------|------|
| L1 | `ParallelAgent` 子任务输出顺序非确定性，未按 index 排序 | `orchestration.go:186` |
| L2 | `PlanExecuteAgent` 始终只用 `executors[0]`，其他 executor 从未被调用 | `plan_execute.go:89` |
| L3 | `rateLimitAgent.times` 底层数组在滑窗清理后不释放 | `builder.go:550` |
| L4 | memory `AddMessage` 错误被 `_` 丢弃，无日志 | `builder.go:928,953` |
| L5 | `eventAgentWrapper` goroutine 缺 `defer stream.Close()`，panic 时 stream 泄漏 | `event_agent.go:58` |
| L6 | `HubClient.Install` 接口方法定义但从未被 `Manager` 调用，死代码 | `pkg/skill/client.go:116` |

---

## 优先修复路径

```
第一优先（CRITICAL，立即修复）
  C1 workflow code 节点沙箱 + context 传递
  C2 executeCodeNode 使用父 ctx 派生超时
  C3 cacheKey 碰撞修复（加分隔符）
  C4 EventStream 锁与 channel send 分离

第二优先（HIGH，合并前修复）
  H1 全链路 goroutine 统一加 ctx.Done() guard
  H4 memory DeleteState 改用 HDel 原子操作
  H5/H6 Redis key 和 URL path 做转义/校验
  H7 UUID 改 crypto/rand

其余 MEDIUM/LOW 可在后续迭代修复。
```

---

*审查工具：go vet、go test（196 passed）、human review*
*审查日期：2026-05-12*
