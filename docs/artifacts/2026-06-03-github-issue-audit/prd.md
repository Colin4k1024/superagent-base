# PRD: GitHub Issue #2 项目审查报告处置计划

**类型**: 项目质量改进
**状态**: intake / draft
**日期**: 2026-06-03
**Owner**: tech-lead
**来源 Issue**: Colin4k1024/superagent-base#2 《项目审查报告》

---

## 背景

GitHub 仓库 `Colin4k1024/superagent-base` 的 Issue #2《项目审查报告》由外部审查人员提交，包含 37 个问题，涵盖 Critical / Security / Architecture / DevOps / Testing / Configuration / Documentation / Code Quality 八大类别。本文档对这 37 个问题进行结构化分析，给出优先级评估、解决方案和角色分派，为后续 `/team-plan` 和 `/team-execute` 提供可执行依据。

---

## 目标与成功标准

**业务目标**: 消除项目中的安全漏洞和关键缺陷，提升代码库的生产就绪度，降低外部贡献者上手门槛。

**成功标准**:
- P0 (Critical + Security) 问题全部关闭
- P1 (Architecture + DevOps) 核心问题解决
- CI 流水线建立，测试覆盖率基线可度量
- 文档与代码保持一致

---

## 用户故事

| 角色 | 故事 | 验收标准 |
|------|------|----------|
| 开发者 | 我希望 Go module 路径正确，能 `go get` 成功 | `go build ./...` 无路径错误 |
| 安全工程师 | 我希望 API Key 不在前端 localStorage 明文存储 | 敏感凭据存入 httpOnly Cookie 或通过 BFF 代理 |
| DevOps | 我希望 Docker 镜像不以 root 运行 | Dockerfile 有 USER 非 root 指令，HEALTHCHECK 存在 |
| QA | 我希望有 CI workflow 自动跑测试 | GitHub Actions 在 PR 时自动触发 `go test ./...` |
| 贡献者 | 我希望文档端口、版本号一致 | README 中所有端口和 API 版本描述与代码一致 |

---

## 问题清单与解决方案

### P0 — Critical（必须立即修复）

#### C-1: Go module 路径与仓库名不匹配
- **现象**: `go.mod` 中路径为 `github.com/superagent-ai/superagent-base/backend`，而 fork 后 GitHub 路径为 `Colin4k1024/superagent-base`
- **影响**: 外部贡献者无法 `go get`，CI 可能失败
- **方案**: 在 fork 维护策略中说明 module path 保持上游不变；若需改名则全局替换并更新所有 import
- **Owner**: tech-lead
- **估时**: 0.5d

#### C-2: 核心模块缺乏文档
- **现象**: `pkg/skill/`, `pkg/tool/`, `pkg/a2ui/`, `pkg/mcp/` 无 README 或注释文档
- **影响**: 新人无法理解核心抽象
- **方案**: 为每个 pkg 子目录补充最小 README（职责、接口、使用示例）
- **Owner**: backend-engineer + tech-lead review
- **估时**: 2d

#### C-3: API 版本不一致（v1 vs v2）
- **现象**: 代码路由注册为 `/api/v1/...`，部分文档写 `/v2/`，CLAUDE.md 与 README 描述不同
- **影响**: 集成方无法确定正确端点
- **方案**: 统一以代码为准（v1），全局搜索并修正文档中的 v2 引用
- **Owner**: backend-engineer
- **估时**: 0.5d

#### C-4: code_execute 工具缺乏沙箱隔离
- **现象**: 代码执行工具直接在宿主进程运行，无 cgroup/namespace/seccomp 隔离
- **影响**: 任意代码执行风险，生产环境不可用
- **方案**: 短期禁用该工具（配置项默认 disabled）；长期评估 gVisor/Firecracker/nsjail 方案，进 ADR
- **Owner**: architect + backend-engineer
- **估时**: 0.5d（禁用）/ 2周（沙箱方案）

---

### P0 — Security（必须修复，部分影响生产）

#### S-1: API Key 明文存储在 localStorage
- **现象**: 前端将 API Key 写入 `localStorage`，可被 XSS 读取
- **影响**: API Key 泄漏，影响所有用户
- **方案**: 改用 httpOnly Cookie 或通过 BFF 代理，前端不持有明文 Key
- **Owner**: frontend-engineer + architect
- **估时**: 1d

#### S-2: MySQL DSN 包含明文密码在配置文件
- **现象**: `.env.example` 或默认配置包含弱密码或明文 DSN
- **影响**: 开发者误将默认配置带入生产
- **方案**: `.env.example` 用占位符替代真实密码，启动时强制校验 `DB_PASSWORD` 非空且非默认
- **Owner**: devops-engineer
- **估时**: 0.5d

#### S-3: Redis 默认无密码
- **现象**: 开发配置 Redis 无 auth
- **方案**: `docker-compose-dev.yml` 加 requirepass，代码侧 `REDIS_PASSWORD` 环境变量强制读取
- **Owner**: devops-engineer
- **估时**: 0.5d

#### S-4: `/metrics` 端点未鉴权
- **现象**: Prometheus `/metrics` 端点对所有人开放，暴露内部指标和配置信息
- **方案**: 添加 Bearer Token 或 IP 白名单中间件，或仅 localhost 监听
- **Owner**: backend-engineer
- **估时**: 0.5d

#### S-5: checkpoint 默认使用内存存储
- **现象**: 生产无持久化 checkpoint，agent 重启后中断状态丢失
- **方案**: 补充文档说明生产需配置 Redis checkpoint；代码侧在启用 interrupt 时检查存储配置并告警
- **Owner**: backend-engineer
- **估时**: 0.5d

#### S-6: evolution 模块写操作缺乏访问控制
- **现象**: evolution 基因库写入接口无鉴权
- **方案**: 在路由层加 API Key 或 RBAC 检查
- **Owner**: backend-engineer
- **估时**: 0.5d

---

### P1 — Architecture（计划修复）

#### A-1: Helm chart 命名与项目不一致
- **方案**: 统一 chart name 与 app label，更新 values.yaml 注释
- **Owner**: devops-engineer
- **估时**: 1d

#### A-2: agent 构建顺序依赖隐式假设
- **现象**: 两阶段构建（leaf first）缺乏文档说明，YAML 顺序敏感
- **方案**: 在 builder.go 添加注释；将依赖解析错误提升为启动时 fatal
- **Owner**: backend-engineer
- **估时**: 0.5d

#### A-3: Evolution 模块与 MySQL 共用存储
- **现象**: 基因库和主业务数据库混用，影响隔离和性能
- **方案**: ADR 评估是否拆库；短期加 table prefix 隔离
- **Owner**: architect
- **估时**: 进 ADR 讨论

#### A-4: eino_graph 标注"coming soon"但有代码引用
- **方案**: 清理死引用或移除 coming soon 标注
- **Owner**: backend-engineer
- **估时**: 0.5d

#### A-5: deep_agent / plan_execute 状态不明
- **现象**: 代码存在但未知是否生产可用
- **方案**: 文档标注实验性状态；添加 `spec.status: experimental` 字段
- **Owner**: backend-engineer + tech-lead
- **估时**: 0.5d

---

### P1 — DevOps（计划修复）

#### D-1: Dockerfile 以 root 运行
- **方案**: 添加 `RUN adduser -D appuser && USER appuser`
- **Owner**: devops-engineer
- **估时**: 0.5d

#### D-2: 无 HEALTHCHECK 指令
- **方案**: `HEALTHCHECK CMD wget -qO- http://localhost:8888/health || exit 1`
- **Owner**: devops-engineer
- **估时**: 0.5d

#### D-3: Docker 镜像不含前端资源
- **现象**: 容器启动后前端页面 404，需单独构建前端并复制 static/
- **方案**: 多阶段构建：先 `npm run build`，再 `make fe`，再 Go 编译
- **Owner**: devops-engineer
- **估时**: 1d

#### D-4: Makefile 含 macOS 专属命令
- **方案**: 替换为跨平台写法或添加 OS 检测
- **Owner**: devops-engineer
- **估时**: 0.5d

#### D-5: `.env.example` 路径与说明不匹配
- **方案**: 统一路径说明，在 README 和 CLAUDE.md 中保持一致
- **Owner**: devops-engineer
- **估时**: 0.25d

#### D-6: alpine 基础镜像版本未固定
- **方案**: `FROM alpine:3.19` 替代 `FROM alpine:latest`
- **Owner**: devops-engineer
- **估时**: 0.25d

---

### P2 — Testing（计划建立基线）

#### T-1: 单元测试覆盖率低
- **方案**: 建立覆盖率基线（当前度量），设定 60% 目标，优先覆盖 pkg/agentdef、pkg/modelrouter
- **Owner**: qa-engineer + backend-engineer
- **估时**: 3d

#### T-2: 缺少 CI workflow
- **方案**: 添加 `.github/workflows/ci.yml`，触发 `go test ./...`、`golangci-lint`、`npm run build`
- **Owner**: devops-engineer
- **估时**: 1d

#### T-3: 无编排层集成测试
- **方案**: 为 supervisor/sequential/parallel agent 添加 E2E 场景测试
- **Owner**: qa-engineer
- **估时**: 2d

---

### P2 — Configuration（低风险，计划修复）

#### CF-1: DSN 中 db 名为 `opencoze`
- **方案**: 改为 `superagent` 或通过环境变量控制，更新 `.env.example`
- **Owner**: devops-engineer
- **估时**: 0.25d

#### CF-2: 重复的 model 配置
- **现象**: 同一模型在多处配置，容易不一致
- **方案**: 提取到统一 model catalog，其他处引用
- **Owner**: backend-engineer
- **估时**: 1d

#### CF-3: Evolution 模块无生产部署指南
- **方案**: 补充 `docs/wiki/evolution-production.md`
- **Owner**: backend-engineer
- **估时**: 0.5d

---

### P2 — Documentation（可并行修复）

#### DOC-1: 端口说明不一致（文档 3000 vs 代码 3500）
- **方案**: 统一为实际端口，全局搜索替换
- **Owner**: backend-engineer
- **估时**: 0.25d

#### DOC-2: CLAUDE.md v1 API 与 README v2 不一致
- **方案**: 以代码路由为准统一版本号
- **Owner**: backend-engineer
- **估时**: 0.25d

#### DOC-3: Makefile help 信息过时
- **方案**: 更新 `make help` 输出，与现有 target 对齐
- **Owner**: devops-engineer
- **估时**: 0.25d

#### DOC-4: CHANGELOG 日期全部相同
- **方案**: 按实际发版时间修正；建立 changelog 维护规范
- **Owner**: tech-lead
- **估时**: 0.25d

#### DOC-5: AI 工具目录（.claude/ 等）被提交
- **方案**: 将 `.claude/`、`.cursor/`、`.aider/` 加入 `.gitignore`；清理已追踪文件
- **Owner**: tech-lead
- **估时**: 0.25d

---

### P3 — Code Quality（技术债，逐步改善）

#### Q-1: Makefile shell 脆弱（未处理错误）
- **方案**: 添加 `set -euo pipefail`，关键命令加 `|| exit 1`
- **Owner**: devops-engineer
- **估时**: 0.5d

#### Q-2: YAML 校验规则未文档化
- **方案**: 补充 schema 说明，或添加 JSON Schema/CUE 校验
- **Owner**: backend-engineer
- **估时**: 1d

#### Q-3: TurnLoop 无超时配置项
- **方案**: 添加 `turn_timeout` 环境变量，默认 120s
- **Owner**: backend-engineer
- **估时**: 0.5d

#### Q-4: Prometheus metric 命名不一致
- **现象**: 混用下划线和驼峰
- **方案**: 统一为 snake_case，更新 Grafana 面板
- **Owner**: backend-engineer
- **估时**: 0.5d

#### Q-5: fsnotify debounce 存在竞态
- **方案**: 添加 debounce 定时器，避免 YAML 写入时多次触发重载
- **Owner**: backend-engineer
- **估时**: 0.5d

---

## 范围

**In Scope**:
- 上述 37 个已识别问题的修复计划
- CI 流水线建立
- 安全配置加固

**Out of Scope**:
- 新功能开发（evolution 新特性、新 agent 类型）
- 重大架构重写（存储层分离等进 ADR 后单独立项）
- 性能优化专项

---

## 风险与依赖

| 风险 | 影响 | 缓解 |
|------|------|------|
| code_execute 沙箱方案选型复杂 | C-4 长期方案可能超出本期范围 | 短期先禁用，沙箱方案独立 ADR |
| localStorage 改造涉及前后端协调 | S-1 需前后端同步修改 | 前后端 engineer 同期介入 |
| Helm chart 改名影响已部署环境 | A-1 有迁移风险 | 保持向后兼容 label，单独迁移文档 |
| 覆盖率目标难以短期达到 | T-1 需持续投入 | 先建立度量基线，分批迭代 |

**待确认项**:
- [ ] C-4 code_execute：是否有生产用户，短期禁用影响范围？
- [ ] S-1 API Key 存储：当前认证架构是否支持 BFF 代理模式？
- [ ] A-3 Evolution 拆库：业务量级是否已到拆库临界点？
- [ ] DOC-5 `.claude/` 目录：是否有意提交作为协作配置共享？

---

## 企业治理待确认项

- 应用等级：当前为 POC / 开源项目，无正式等级评定，按 T4 最低基线执行
- 数据合规：无 PII / 跨境数据处理，暂无合规风险
- 技术架构等级：开源自建，不受集团统一架构约束

---

## 参与角色清单

| 角色 | 参与范围 |
|------|----------|
| tech-lead | 全局审查、C-1/C-4/A-3/DOC-5，最终放行 |
| architect | C-4 沙箱方案、A-3 ADR、S-1 BFF 架构 |
| backend-engineer | C-2/C-3/A-2/A-4/A-5/S-4/S-5/S-6/CF-2/Q-2/Q-3/Q-4/Q-5 |
| frontend-engineer | S-1 API Key 存储改造 |
| devops-engineer | D-1~D-6/S-2/S-3/T-2/CF-1/CF-3/Q-1/DOC-3/DOC-5 |
| qa-engineer | T-1/T-3 测试覆盖建立 |

---

## 需求挑战会候选分组

**建议进入挑战会的问题组**（存在方案分歧或跨角色依赖）:

1. **安全组**: S-1（API Key 存储）+ S-4（/metrics 鉴权）— 需确认认证架构边界
2. **沙箱组**: C-4（code_execute）— 技术方案评估，gVisor vs nsjail vs 禁用
3. **存储架构组**: A-3（Evolution 拆库）— 是否本期立项，还是 backlog
4. **CI/CD 组**: T-2 + D-1~D-3 — 流水线设计与 Docker 多阶段构建联动

---

## 领域技能包启用建议

- 启用 `golang/coding-style` 和 `golang/testing` 规则覆盖 Q 类问题
- 启用 `common/security` 覆盖 S 类问题审查
- 启用 `common/enterprise-architecture-governance` 的 T4 基线用于 DevOps 合规检查
- 启用 `typescript/frontend` 覆盖 S-1 前端改造

---

## UI 范围

S-1 涉及前端 API Key 存储改造，目标端为 Web 浏览器，关键页面为登录/设置页。设计约束：不改变现有 UI 视觉，仅修改凭据存储位置和传输方式。性能/A11y 门禁：不涉及新页面，无新增可访问性要求。

---

**已创建**: `docs/artifacts/2026-06-03-github-issue-audit/prd.md`
