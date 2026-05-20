# Workflow 并行执行设计

## 背景

当前 Workflow 引擎采用 Kahn 拓扑排序后完全串行执行（`workflow_builder.go:74-96`），同一层级无依赖的节点未被并行化。共享 state 是无锁 `map[string]string`，在单 goroutine 下安全但无法并发写入。

## 目标

同一拓扑层级内无数据依赖的节点应并行执行，同时保证状态安全、错误可控、checkpoint 可恢复。

## 设计方案

### 拓扑层级识别

将 Kahn 算法改为按层输出：

```go
func (w *WorkflowAgent) topologicalLevels() ([][]string, error) {
    // 初始化入度和邻接表（与现有 topologicalSort 相同）
    var levels [][]string
    for len(queue) > 0 {
        level := queue
        queue = nil
        levels = append(levels, level)
        for _, curr := range level {
            for _, neighbor := range adj[curr] {
                inDegree[neighbor]--
                if inDegree[neighbor] == 0 {
                    queue = append(queue, neighbor)
                }
            }
        }
    }
    return levels, nil
}
```

### 并发安全 State

```go
type safeState struct {
    mu   sync.RWMutex
    data map[string]string
}

func (s *safeState) snapshot() map[string]string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    cp := make(map[string]string, len(s.data))
    for k, v := range s.data { cp[k] = v }
    return cp
}

func (s *safeState) set(key, val string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data[key] = val
}
```

**核心不变式**：同层节点只读上游 level 写入的 state，写入各自 `nodeID.output` key，互不冲突。

### 并行执行层

```go
func (w *WorkflowAgent) executeLevel(ctx context.Context, sessionID string,
    nodeIDs []string, state *safeState) error {

    sem := make(chan struct{}, w.maxParallelism())
    var wg sync.WaitGroup
    errCh := make(chan error, len(nodeIDs))

    for _, id := range nodeIDs {
        node := w.getNode(id)
        wg.Add(1)
        go func(n *WorkflowNode) {
            defer wg.Done()
            defer func() {
                if r := recover(); r != nil {
                    errCh <- fmt.Errorf("node %q panicked: %v", n.ID, r)
                }
            }()
            sem <- struct{}{}
            defer func() { <-sem }()

            result, err := w.executeNode(ctx, sessionID, n, state.snapshot())
            if err != nil {
                errCh <- err
                return
            }
            state.set(n.ID+".output", result)
        }(node)
    }
    wg.Wait()
    close(errCh)
    return collectErrors(errCh)
}
```

### 错误处理策略

| 策略 | 行为 |
|------|------|
| `fail_fast`（默认） | 任一节点失败 cancel 同层其他 goroutine |
| `best_effort` | 等待所有节点完成，仅下游依赖失败节点时终止 |

实现：`context.WithCancel` 派生子 ctx，`fail_fast` 时首个 error 触发 cancel。

### 条件边语义

- 条件边的 `Condition` 表达式依赖上游节点输出
- 层间转换时评估条件边：只有条件为 true 的目标节点加入下一层执行集
- 条件节点本身在并行层中执行，其输出在层结束后用于裁剪下一层的边

### YAML Schema 扩展

```yaml
spec:
  workflow:
    execution:
      max_parallelism: 4          # 单层最大并发数，0=无限制
      error_strategy: fail_fast   # fail_fast | best_effort
    nodes: [...]
    edges: [...]
```

```go
type WorkflowExecution struct {
    MaxParallelism int    `yaml:"max_parallelism,omitempty"`
    ErrorStrategy  string `yaml:"error_strategy,omitempty"`
}
```

### Checkpoint 兼容

```go
type WorkflowCheckpoint struct {
    CompletedLevels int             `json:"completed_levels"`
    CurrentLevel    int             `json:"current_level"`
    CompletedNodes  map[string]bool `json:"completed_nodes"`
    State           map[string]string `json:"state"`
}
```

Resume 时：跳过已完成 level，对当前 level 中已完成节点跳过、仅执行剩余。

### A2UI 事件流

- 每个节点开始：`progress` 事件 `{"node_id": "xxx", "status": "running"}`
- 完成：`progress` 事件 `{"node_id": "xxx", "status": "done"}`
- 失败：`error` 事件附带 `node_id`
- 客户端按 `node_id` 做 demux（SSE 单通道交错安全）

## 技术决策

| 决策 | 选择 | 备选 | 理由 |
|------|------|------|------|
| 并行粒度 | Level-based | 细粒度就绪驱动 | 实现简单、正确性易证明 |
| State 隔离 | 同层读 snapshot、写各自 key | 全局锁 | 无写冲突 |
| 并发控制 | 信号量 | Worker pool | 更轻量，goroutine 即用即释 |
| 错误策略 | 可配置 | 仅 fail_fast | 覆盖更多场景 |

## 风险

- Level-based 粒度较粗，某层一个慢节点卡住整层（可接受，V2 可评估细粒度）
- 并行节点的 A2UI 事件交错需客户端适配
- 条件边评估在层间引入串行点（不可避免）

## 关键代码位置

- `backend/pkg/agentdef/workflow_builder.go:63` — topologicalSort
- `backend/pkg/agentdef/workflow_builder.go:74-96` — 串行执行循环
- `backend/pkg/agentdef/workflow_builder.go:61` — state map
- `backend/pkg/agentdef/schema.go:112-161` — Workflow schema
- `backend/pkg/agentdef/interrupt.go:29-53` — Checkpoint
