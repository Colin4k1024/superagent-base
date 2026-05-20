# Kubernetes 部署方案设计

## 背景

项目已有基础 Helm chart（`helm/charts/superagent/`），包含 Deployment、ConfigMap、HPA、Ingress、Secret 模板。后端是**准无状态**的——Agent 定义从 ConfigMap 加载到内存，会话数据在 Redis/MySQL。

## 现状问题

| 问题 | 位置 | 修正 |
|------|------|------|
| readinessProbe 打 `/metrics` | deployment.yaml:44 | 改为 `/ready` |
| 无 gRPC Ingress | templates/ | 补充 |
| 无 ServiceMonitor | templates/ | 补充 |
| HPA 仅 CPU | hpa.yaml | 增加自定义指标 |
| 无 terminationGracePeriod | deployment.yaml | 增加 60s |

## 核心架构决策

### Deployment vs StatefulSet

**选择：Deployment**

理由：
- Agent 注册表在启动时从 ConfigMap 加载到内存，无持久化需求
- Memory 后端走 Redis，无本地磁盘状态
- 不需要稳定网络标识或有序滚动

### 多副本 Agent 定义一致性

| 方案 | 适用场景 | 推荐 |
|------|----------|------|
| **A: ConfigMap 挂载** | Agent <50 个，变更不频繁 | 短期 |
| **B: 中心化 Registry** | 企业规模，数百 Agent | 长期可选 |
| **C: Git-sync sidecar** | 大量 YAML，需版本控制 | 中期 |

短期用 A（已实现），中期演进到 C。fsnotify watcher 天然兼容 ConfigMap symlink 替换和 git-sync 文件变更。

## 设计方案

### HPA 策略增强

```yaml
metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Pods
    pods:
      metric:
        name: superagent_active_sessions
      target:
        type: AverageValue
        averageValue: "50"
```

### 滚动更新配置

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      terminationGracePeriodSeconds: 60
      containers:
        - lifecycle:
            preStop:
              exec:
                command: ["sleep", "5"]
```

### gRPC Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ .Release.Name }}-grpc
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
spec:
  rules:
    - host: {{ .Values.grpc.host }}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: {{ include "superagent.fullname" . }}
                port:
                  name: grpc
```

### ServiceMonitor

```yaml
{{- if .Values.observability.metrics.serviceMonitor }}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ include "superagent.fullname" . }}
spec:
  selector:
    matchLabels:
      {{- include "superagent.selectorLabels" . | nindent 6 }}
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
{{- end }}
```

### Git-sync Sidecar（中期方案）

```yaml
containers:
  - name: git-sync
    image: registry.k8s.io/git-sync/git-sync:v4.2.0
    args:
      - --repo=https://github.com/org/agent-configs.git
      - --root=/git
      - --period=30s
    volumeMounts:
      - name: agent-configs
        mountPath: /git
  - name: superagent
    volumeMounts:
      - name: agent-configs
        mountPath: /app/configs/agents
        subPath: agents
```

### 依赖中间件策略

| 中间件 | 推荐方式 | 理由 |
|--------|----------|------|
| MySQL | 外部托管 (RDS) | 生产级 HA |
| Redis | 外部托管或 Bitnami chart | 需持久化 |
| ES | 外部托管或 ECK | 运维复杂 |
| MinIO | 云对象存储 (S3/OSS) | 直接替换 |

### Secret 管理

```yaml
# values.yaml
mysql:
  existingSecret: ""  # 引用外部 Secret（Sealed Secrets / Vault）
  auth:
    password: ""      # 仅在无 existingSecret 时使用
```

## Grafana Dashboard 面板

| 面板 | 指标 |
|------|------|
| Agent QPS | `rate(superagent_chat_requests_total[5m])` |
| 模型延迟 P95 | `histogram_quantile(0.95, superagent_model_duration_seconds_bucket)` |
| Tool 成功率 | `rate(superagent_tool_calls_total{status="success"}[5m])` |
| Reload 失败 | `superagent_agent_reload_failures_total` |
| 活跃连接 | `superagent_active_sse_connections` |

## 热重载在 K8s 下的行为

1. kubelet 每 60s 同步 ConfigMap 更新到 Pod 内文件
2. fsnotify watcher 捕获 symlink 替换事件
3. debounce 2s 后触发 `reloader.ReloadDir()`
4. Two-pass build：leaf → orchestration
5. 构建失败的 agent 保留旧版本

**多 Pod 不一致窗口**：最多 1-2 分钟（kubelet sync + debounce），可接受。急需立即生效可调用 Admin API `POST /api/v1/admin/reload`。

## 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 部署类型 | Deployment | 准无状态 |
| Agent 同步 | ConfigMap → Git-sync | 渐进式 |
| HPA 指标 | CPU + active_sessions | 贴合业务 |
| 探针 | `/ready` | 反映 runtime 就绪 |
| 优雅终止 | 60s + preStop sleep 5 | SSE 长连接 |
| 中间件 | 外部托管 | 生产级 |

## 风险

- ConfigMap 1MB 限制（>50 Agent 需切 git-sync）
- 多 Pod 短暂不一致（大多数场景可接受）
- 自定义 HPA 指标需 Prometheus Adapter 额外部署

## 关键代码位置

- `backend/main.go:274-290` — 健康检查
- `backend/pkg/agentdef/runtime.go:48-57` — AgentRuntime 内存状态
- `backend/pkg/agentdef/watcher.go:60-105` — fsnotify 热重载
- `helm/charts/superagent/` — 现有 Helm chart
