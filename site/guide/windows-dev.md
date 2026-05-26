# Windows 本地开发指南

## 前置条件

| 工具 | 最低版本 | 说明 |
|------|---------|------|
| Go | 1.24+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 18+ | [nodejs.org](https://nodejs.org/)，运行前端开发服务器 |
| Docker Desktop | 4.x | 包含 Docker Compose，用于启动 MySQL + Redis |
| Git | 2.x | |
| PowerShell | 5.1+ | Windows 内置，建议用 7.x |

> **WSL2 用户**：可直接使用 `make dev`，无需本文档。

---

## 快速启动

```powershell
# 1. 克隆仓库
git clone https://github.com/superagent-ai/superagent-base.git
cd superagent-base

# 2. 复制 env 文件（MySQL + Redis 地址）
copy docker\.env.dev backend\.env.dev
# 根据实际地址编辑 backend\.env.dev 中的 MYSQL_DSN 和 REDIS_ADDR

# 3. 一键启动中间件 + 后端 + 前端
.\scripts\dev-windows.ps1
```

启动后访问地址：
- 后端 API：`http://localhost:8888`
- 前端页面：`http://localhost:3000`（自动代理 `/api` → `:8888`）

---

## 分步启动

```powershell
# 只启动 MySQL + Redis（Docker）
.\scripts\dev-windows.ps1 -Action middleware

# 只启动后端（中间件已运行时）
.\scripts\dev-windows.ps1 -Action server

# 只启动前端（后端已运行时）
.\scripts\dev-windows.ps1 -Action web

# 停止中间件
.\scripts\dev-windows.ps1 -Action down

# 停止并清理数据
.\scripts\dev-windows.ps1 -Action clean
```

### 手动启动（不用脚本）

```powershell
# 启动中间件
docker compose -f docker/docker-compose-dev.yml up -d --wait

# 启动后端（新 PowerShell 窗口）
cd backend
$env:APP_ENV = "dev"
go run .

# 启动前端（另一个 PowerShell 窗口）
cd web
npm install   # 首次运行
npm run dev
```

> **注意**：PowerShell 不支持 `APP_ENV=dev go run .` 语法（Linux/macOS 用法），
> 需要先 `$env:APP_ENV = "dev"` 再 `go run .`。

---

## env 配置说明

`backend/.env.dev` 示例：

```env
MYSQL_DSN=coze:coze123@tcp(127.0.0.1:3306)/opencoze?charset=utf8mb4&parseTime=True
REDIS_ADDR=127.0.0.1:6379
VECTOR_STORE_TYPE=        # 留空：禁用向量存储（无需 Milvus）
ES_ADDR=                  # 留空：禁用 Elasticsearch
COZE_MQ_TYPE=nsq
CODE_RUNNER_TYPE=local
OTEL_ENABLED=false
```

如果 MySQL 或 Redis 运行在非默认端口，修改对应地址即可。

如果使用本地 LLM（如 LM Studio），追加：
```env
MODEL_BASE_URL_0=http://127.0.0.1:8000/v1
MODEL_API_KEY_0=your-key
MODEL_ID_0=your-model-id
BUILTIN_CM_OPENAI_BASE_URL=http://127.0.0.1:8000/v1
```

---

## 常见问题

### `docker compose` 命令找不到
确认 Docker Desktop 已安装并在运行，或使用 `docker-compose`（带连字符的旧版命令）替代。

### 端口被占用
```powershell
netstat -ano | findstr :3306   # MySQL
netstat -ano | findstr :6379   # Redis
netstat -ano | findstr :8888   # 后端
netstat -ano | findstr :3000   # 前端
```
修改 `docker/docker-compose-dev.yml` 中对应端口映射，并同步修改 `backend/.env.dev`。
前端端口在 `web/vite.config.ts` 的 `server.port` 中修改。

### go build 提示缺少 CGO 依赖
本项目不需要 CGO，确认 Go 环境正常：
```powershell
go version
go env GOOS GOARCH
```

### 防火墙阻断 Docker 网络
以管理员身份运行 PowerShell，或在 Windows Defender 防火墙中允许 Docker Desktop。

### npm install 或 npm run dev 失败
确认 Node.js 18+ 已安装：
```powershell
node --version
npm --version
```
