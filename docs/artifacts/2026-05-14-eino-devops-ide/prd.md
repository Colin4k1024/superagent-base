# PRD: Eino Dev IDE 插件集成（图形化编排）

- **状态**: completed
- **角色**: tech-lead
- **阶段**: executed
- **日期**: 2026-05-14
- **Slug**: eino-devops-ide

---

## 背景

项目已深度集成 Eino 框架（`v0.4.8`）进行 AI Agent 和 Workflow 编排。目前所有 Graph 构建均以纯代码方式完成（`compose.Graph` / `compose.Chain`）。

Eino 官方提供 `eino-ext/devops` 包与配套 IDE 插件（GoLand / VS Code），可在开发阶段实现：

1. **图形化编排**：在 IDE 中可视化设计 Graph 拓扑，自动生成对应 Go 代码
2. **可视化调试**：运行时通过 IDE 插件观察 Graph 节点执行状态和数据流

需求来源：用户希望通过图形化编排策略快速构建 Agent/Workflow，尽量不手写代码，最大化兼容 Eino 官方能力。

---

## 目标与成功标准

### 业务目标
- 开发者能在 GoLand 2023.2+ 或 VS Code 1.97.x+ 中通过 Eino Dev 插件可视化设计 Graph
- 编排结果可直接对应当前项目中的 `compose.Graph` 代码

### 成功标准
- [ ] `eino-ext/devops` 依赖成功加入 `backend/go.mod`
- [ ] `devops.Init(ctx)` 在 `main.go` 中成功启动，监听 `127.0.0.1:52538`
- [ ] IDE 插件在 GoLand/VS Code 中可识别并连接到本地 devops server
- [ ] 可视化图形编排界面正常展示当前项目已有的 Graph 结构

---

## 范围

### In Scope
- 在 `backend/go.mod` 添加 `github.com/cloudwego/eino-ext/devops` 依赖
- 在 `backend/main.go` 的 dev/debug 模式启动路径中调用 `devops.Init(ctx)`
- 验证 eino 版本兼容性（当前 v0.4.8，devops 要求 v0.6.0）
- IDE 插件安装指引文档

### Out of Scope
- 不修改任何业务逻辑代码
- 不引入自定义 devops 扩展
- 不涉及生产环境部署（devops server 仅用于本地开发）
- 不修改前端代码

---

## 用户故事

**Story 1**: 作为开发者，我希望在 GoLand/VS Code 中安装 Eino Dev 插件后，能通过图形界面设计 Agent Graph，减少手写 compose.Graph 代码的工作量。

**验收标准**:
- IDE 插件已安装（GoLand 2023.2+ 或 VS Code 1.97.x+）
- 项目启动后 devops server 在 `127.0.0.1:52538` 可访问
- IDE 插件侧边栏显示 Graph 可视化界面

**Story 2**: 作为开发者，我希望可视化调试时能看到 Graph 节点的执行状态和数据流，而无需添加大量 logging 代码。

**验收标准**:
- 运行测试用例时，IDE 插件调试面板展示节点执行顺序
- 节点输入/输出可在插件中查看

---

## 技术约束与风险

### 关键风险：eino 版本升级

| 项目 | 当前版本 | devops 要求 | 风险 |
|------|---------|------------|------|
| `github.com/cloudwego/eino` | v0.4.8 | v0.6.0 (devops go.mod) | **HIGH** - Go MVS 将自动升级至 v0.6.0 |

**影响分析**:
- eino v0.4.8 → v0.6.0 跨越 2 个 minor 版本
- 需要在 `/team-plan` 阶段验证升级是否有 breaking changes
- 现有依赖 (`eino-ext/components/model/*`, `eino-ext/components/embedding/*`) 可能需要同步升级

### 其他约束
- `devops.Init()` 默认绑定 `127.0.0.1:52538`，仅限本地开发使用（安全）
- IDE 插件要求特定版本：GoLand 2023.2+ / VS Code 1.97.x+
- 集成代码量极小（仅 1 行 `devops.Init(ctx)`）

---

## 参与角色

| 角色 | 职责 |
|------|------|
| tech-lead | intake、方案收口、eino 版本升级风险决策 |
| backend-engineer | go.mod 更新、main.go 集成、版本兼容性验证 |
| qa-engineer | 本地验证 devops server 可达性、IDE 插件连接测试 |

---

## 待确认项

1. **[CRITICAL]** eino v0.4.8 → v0.6.0 升级是否有 breaking API changes？哪些文件受影响？
2. **[HIGH]** 现有 `eino-ext/components/*` 依赖是否需要同步升级版本？
3. **[MEDIUM]** `devops.Init()` 是否应该仅在 dev/debug 环境中启用（通过环境变量控制）？
4. **[LOW]** 团队使用 GoLand 还是 VS Code？是否都需要验证？

---

## 需求挑战会候选分组

- **eino 版本升级影响评估**：backend-engineer + tech-lead（评估 v0.4.8→v0.6.0 的 breaking changes 范围）
- **集成最小化方案**：确认 `devops.Init()` 的调用时机和环境隔离策略

---

## 非目标（Karpathy Guidelines）

- 不引入任何新的编排模式或自定义扩展
- 不修改生产代码路径
- 不替换现有 Eino 使用方式，仅增加 dev 可观察性
