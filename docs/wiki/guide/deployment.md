# 部署指南

## 本地开发（推荐）

### 前置条件

- Go 1.24+
- Docker + Docker Compose
- LM Studio 或 Ollama（本地 LLM）

### 快速启动

```bash
# 1. 克隆仓库
git clone <repo-url>
cd superagent-base

# 2. 启动 MySQL + Redis（轻量级 dev 栈）
make dev-middleware

# 3. 构建并运行 backend
make dev-server

# 或者一步完成：
make dev
```

backend 在 `http://localhost:8888` 启动，gRPC 监听 `localhost:50051`。

### 停止

```bash
make dev-down        # 停止容器
make dev-clean       # 停止容器并删除 MySQL 数据
```

### 环境配置

dev 模式使用 `docker/.env.dev`，启动时自动复制到 `backend/.env.dev`。修改模型配置：

```bash
# docker/.env.dev
MODEL_PROTOCOL_0=openai
MODEL_ID_0=Qwen3-Coder-Next-4bit
MODEL_API_KEY_0=123456
MODEL_BASE_URL_0=http://127.0.0.1:8000/v1  # LM Studio
```

详细模型配置见 [model-config.md](./model-config.md)。

---

## Docker Compose 全栈（debug 模式）

完整栈包含 MySQL、Redis、Elasticsearch、Milvus、MinIO、NSQ、etcd 等所有服务。

### 启动

```bash
# 1. 复制并编辑 env 文件
cp docker/.env.debug.example docker/.env.debug
# 编辑 docker/.env.debug，填写模型 API Key 等

# 2. 启动全部中间件
make middleware

# 3. 构建并运行 backend
make server
```

### 或者使用 debug 一键启动

```bash
make debug
```

此命令等价于 `make env middleware python server`。

### 服务端口

| 服务 | 端口 |
|------|------|
| Backend HTTP | 8888 |
| Backend gRPC | 50051 |
| MySQL | 3306 |
| Redis | 6379 |
| MinIO | 9000 / 9001 |
| Elasticsearch | 9200 |
| NSQ | 4150 / 4151 |
| etcd | 2379 |
| Milvus | 19530 |

### 关闭

```bash
make down    # 停止容器
make clean   # 停止并清理 volumes 数据
```

---

## 生产：Docker Compose

### 准备 env 文件

```bash
cp docker/.env.example docker/.env
# 编辑 docker/.env，填写生产参数：
# - MYSQL_PASSWORD（改为强密码）
# - MINIO_ROOT_PASSWORD
# - MODEL_API_KEY_0
# - PLUGIN_AES_AUTH_SECRET（随机生成）
# - USE_SSL=1（如启用 HTTPS）
```

### 启动

```bash
make web
```

### 关闭

```bash
make down_web
```

---

## 生产：Kubernetes + Helm

### 前置条件

- Kubernetes 1.24+
- Helm 3.x
- 已配置的 MySQL、Redis（建议使用托管服务）

### 安装

```bash
cd helm

# 查看可配置项
helm show values charts/opencoze/

# 安装（最简配置）
helm install superagent charts/opencoze \
  --namespace superagent \
  --create-namespace \
  --set mysql.host=your-mysql-host \
  --set redis.addr=your-redis-host:6379 \
  --set model.apiKey=your-api-key \
  --set model.baseURL=https://api.openai.com/v1 \
  --set model.id=gpt-4o
```

### 升级

```bash
helm upgrade superagent charts/opencoze \
  --namespace superagent \
  --reuse-values \
  --set image.tag=v1.2.0
```

### 查看状态

```bash
kubectl get pods -n superagent
kubectl logs -n superagent deploy/superagent-backend -f
```

---

## 环境变量完整参考

### 服务器

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `APP_ENV` | `""` | 环境标识，决定加载哪个 `.env.*` 文件 |
| `LISTEN_ADDR` | `:8888` | HTTP 监听地址 |
| `GRPC_LISTEN_ADDR` | `:50051` | gRPC 监听地址 |
| `LOG_LEVEL` | `info` | 日志级别：trace / debug / info / warn / error / fatal |
| `SERVER_HOST` | — | 对外暴露的服务 URL（用于回调等） |
| `MAX_REQUEST_BODY_SIZE` | `209715200` | 最大请求体（字节，默认 200MB） |
| `USE_SSL` | `0` | 启用 HTTPS：设为 `1` 并配置证书路径 |
| `SSL_CERT_FILE` | — | TLS 证书文件路径 |
| `SSL_KEY_FILE` | — | TLS 私钥文件路径 |

### Agent 运行时

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `AGENT_CONFIG_DIR` | `configs/agents` | Agent YAML 配置目录，支持热重载 |
| `AGENT_RELOAD_DEBOUNCE` | `2s` | 热重载防抖时间 |
| `INTERRUPT_DEFAULT_TIMEOUT` | `300` | 中断状态默认保留秒数 |
| `SKILLS_HUB_URL` | — | SkillsHub 服务 URL（用于安装远程技能） |
| `MCP_CONFIG_FILE` | — | MCP 服务器配置文件路径 |

### MySQL

| 变量 | 说明 |
|------|------|
| `MYSQL_DSN` | 完整 DSN，格式：`user:pass@tcp(host:port)/db?charset=utf8mb4&parseTime=True` |

### Redis

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis 地址 |
| `REDIS_PASSWORD` | `""` | Redis 密码 |

### 对象存储

| 变量 | 说明 |
|------|------|
| `STORAGE_TYPE` | 存储类型：`minio` / `tos` / `s3` |
| `STORAGE_BUCKET` | 存储桶名称 |
| `MINIO_ENDPOINT` | MinIO 地址（如 `127.0.0.1:9000`） |
| `MINIO_AK` | MinIO Access Key |
| `MINIO_SK` | MinIO Secret Key |
| `MINIO_API_HOST` | MinIO 对外访问 URL（如 `http://127.0.0.1:9000`） |

### Elasticsearch（可选）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ES_ADDR` | — | ES 地址，留空则禁用 |
| `ES_VERSION` | `v8` | ES 版本 |
| `ES_USERNAME` | — | ES 用户名 |
| `ES_PASSWORD` | — | ES 密码 |

### 消息队列（可选）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `COZE_MQ_TYPE` | `nsq` | 队列类型：`nsq` / `kafka` / `rmq` |
| `MQ_NAME_SERVER` | — | 队列服务地址，留空则禁用 |

### 向量存储（可选）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `VECTOR_STORE_TYPE` | — | 向量库类型，留空则禁用 |
| `EMBEDDING_TYPE` | — | 向量化类型，留空则禁用 |

### 模型

| 变量 | 说明 |
|------|------|
| `MODEL_PROTOCOL_0` | 协议：openai / deepseek / claude / gemini / ark / ollama / qwen |
| `MODEL_OPENCOZE_ID_0` | 平台内部模型 ID（正整数） |
| `MODEL_NAME_0` | 展示名称 |
| `MODEL_ID_0` | API model 参数值 |
| `MODEL_API_KEY_0` | API 密钥 |
| `MODEL_BASE_URL_0` | API 基础 URL |
| `BUILTIN_CM_TYPE` | 内置能力模型协议 |
| `BUILTIN_CM_<TYPE>_BASE_URL` | 内置能力模型 URL |
| `BUILTIN_CM_<TYPE>_API_KEY` | 内置能力模型密钥 |
| `BUILTIN_CM_<TYPE>_MODEL` | 内置能力模型 ID |

### 插件安全

| 变量 | 说明 |
|------|------|
| `PLUGIN_AES_AUTH_SECRET` | 插件认证 AES 密钥（生产环境必须随机生成） |
| `PLUGIN_AES_STATE_SECRET` | 插件状态 AES 密钥 |
| `PLUGIN_AES_OAUTH_TOKEN_SECRET` | 插件 OAuth Token AES 密钥 |

### 可观测性

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `OTEL_ENABLED` | `false` | 启用 OpenTelemetry 追踪 |
| `SERVICE_NAME` | — | 服务名称（用于追踪 / 指标标签） |

---

## Prometheus Metrics

backend 在 `/metrics` 端点暴露 Prometheus 指标，不受认证中间件保护：

```bash
curl http://localhost:8888/metrics
```

---

## 健康检查与日志

```bash
# 查看 backend 日志
tail -f crash.log          # crash 日志
# 或通过 docker logs
docker logs sa-mysql
docker logs sa-redis

# E2E 冒烟测试
./scripts/e2e-test.sh http://localhost:8888
```
