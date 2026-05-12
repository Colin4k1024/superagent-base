# SkillHub Client API and CLI Integration Guide

本文面向 SkillHub 客户端调用方和 CLI 用户，说明 M11 开放的 Skill 搜索、详情、发布、状态查询、重试和安装能力。

## 1. 接入地址

所有客户端请求应通过 haier-gateway 访问 SkillHub，不直连后端。

本地验收示例：

```text
SkillHub Gateway Base URL: http://127.0.0.1:18092/skillhub
```

后文用 `${SKILLHUB_BASE_URL}` 表示该地址。

## 2. 认证

SkillHub 后端不解析 token，只信任 gateway 校验并放行后的身份头。客户端不要发送 `X-User-Account`。

### 2.1 用户身份

推荐请求头：

```http
X-Access-Token: <iam-access-token>
```

兼容请求头：

```http
Access-Token: <iam-access-token>
```

### 2.2 应用身份

请求头：

```http
App-Access-Token: <app-access-token>
K-Code: <caller-k-code>
VISIT-K-Code: <target-k-code>
```

`App-Access-Token` 由应用凭据向 Techless 获取。调用 SkillHub 业务 API 时，gateway 校验应用访问权限；SkillHub 根据 gateway 放行后的 `K-Code` 投影为 APP 账号。

### 2.3 身份诊断

```bash
curl -fsS \
  -H "X-Access-Token: ${USER_TOKEN}" \
  "${SKILLHUB_BASE_URL}/api/v1/auth/me"
```

```bash
hskill whoami \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}" \
  --json
```

返回中 `accountType` 为 `USER` 或 `APP`。

## 3. Skill 包要求

发布的 ZIP 根目录必须包含：

```text
SKILL.md
```

`AppAccessToken_SKILL.md` 是 gateway 应用 token 获取文档，不是 Skill 包入口文件。

最小 `SKILL.md`：

```markdown
---
name: demo-skill
description: Demo Skill
version: 1.0.0
---

# demo-skill
```

`name` 会转换为 Skill `slug`。CLI 安装和状态查询统一使用：

```text
<namespace>/<slug>
```

## 4. 通用响应字段

搜索、详情、发布、状态和重试响应会携带安装衔接字段：

| 字段 | 说明 |
|---|---|
| `namespace` | Skill namespace，默认 `global`。 |
| `slug` | Skill slug。 |
| `coordinate` | `${namespace}/${slug}`。 |
| `latestVersion` | 最新已上架版本，搜索/详情接口返回。 |
| `installTarget` | 安装目标，通常等于 `coordinate`。 |
| `installVersion` | 可安装版本。 |
| `installCommand` | 可直接复制的 CLI 安装命令。 |

普通文本 CLI 输出第一列是 `namespace/slug`；JSON 输出包含 `coordinate`、`installTarget`、`installCommand`。

## 5. 搜索

### 5.1 搜索当前身份可安装 Skill

API：

```http
GET /api/v1/portal/skills/available?q=<keyword>&size=20
```

示例：

```bash
curl -fsS \
  -H "X-Access-Token: ${USER_TOKEN}" \
  "${SKILLHUB_BASE_URL}/api/v1/portal/skills/available?q=demo"
```

CLI：

```bash
hskill search demo \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}"
```

### 5.2 全局发现搜索

全局搜索返回可发现 Skill，其中一部分可能还不能安装。

API：

```http
GET /api/v1/portal/skills?q=<keyword>&size=20
```

CLI：

```bash
hskill search demo --global \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}"
```

可选过滤：

```text
category=ALL|COMMON|ENTERPRISE|PERSONAL
tag=<tag>
```

CLI 对应：

```bash
hskill search demo --category COMMON --tag tool
```

## 6. 详情

API：

```http
GET /api/v1/portal/skills/{namespace}/{slug}
```

示例：

```bash
curl -fsS \
  -H "X-Access-Token: ${USER_TOKEN}" \
  "${SKILLHUB_BASE_URL}/api/v1/portal/skills/global/demo-skill"
```

CLI：

```bash
hskill inspect global/demo-skill \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}" \
  --json
```

详情响应包含 `versions`、`files` 和安装衔接字段。

## 7. 发布 Skill

发布 Skill 会完成：

```text
上传 -> 扫描 -> 权限设置 -> 上架
```

Portal 发布仍是多步骤；客户端 API 和 CLI 使用发布 Skill 能力。

### 7.1 API

```http
POST /api/v1/client/skills/publish
Content-Type: multipart/form-data
```

表单字段：

| 字段 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `file` | 是 | 无 | Skill ZIP。 |
| `namespace` | 否 | `global` | Skill namespace。 |
| `scanTimeoutSeconds` | 否 | `120` | 同步等待扫描秒数，范围 1-600。 |
| `public` | 否 | `false` | 是否授予 PUBLIC READER。 |
| `readAccounts` | 否 | 空 | 授予 READER 的账号，可重复。 |
| `maintAccounts` | 否 | 空 | 授予 MAINTAINER 的账号，可重复。 |
| `ownerAccounts` | 否 | 空 | 授予 OWNER 的账号，可重复。 |
| `readGroups` | 否 | 空 | Skill Group code，可重复，只表示可安装授权。 |

用户身份示例：

```bash
curl -fsS \
  -H "X-Access-Token: ${USER_TOKEN}" \
  -F "file=@demo.zip;type=application/zip" \
  -F "namespace=global" \
  -F "scanTimeoutSeconds=120" \
  -F "public=true" \
  "${SKILLHUB_BASE_URL}/api/v1/client/skills/publish"
```

应用身份示例：

```bash
curl -fsS \
  -H "App-Access-Token: ${APP_TOKEN}" \
  -H "K-Code: ${CALLER_K_CODE}" \
  -H "VISIT-K-Code: ${TARGET_K_CODE}" \
  -F "file=@demo.zip;type=application/zip" \
  -F "namespace=global" \
  -F "public=true" \
  "${SKILLHUB_BASE_URL}/api/v1/client/skills/publish"
```

返回示例：

```json
{
  "code": 0,
  "data": {
    "namespace": "global",
    "slug": "demo-skill",
    "coordinate": "global/demo-skill",
    "version": "1.0.0",
    "status": "PUBLISHED",
    "scanStatus": "PASS",
    "downloadReady": true,
    "listed": true,
    "failureReason": null,
    "canRetry": false,
    "canManage": true,
    "installTarget": "global/demo-skill",
    "installVersion": "1.0.0",
    "installCommand": "hskill install global/demo-skill"
  }
}
```

### 7.2 CLI

用户身份发布：

```bash
hskill publish ./demo.zip \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}" \
  --ns global \
  --timeout 120 \
  --public
```

应用身份发布：

```bash
hskill publish ./demo.zip \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --App-Access-Token "${APP_TOKEN}" \
  --K-Code "${CALLER_K_CODE}" \
  --VISIT-K-Code "${TARGET_K_CODE}" \
  --public \
  --json
```

授权参数可重复：

```bash
hskill publish ./demo.zip \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}" \
  --read captest002 \
  --read APP:K12345 \
  --maint captest003 \
  --owner K23456 \
  --group common-tools
```

账号解析规则：

| 参数值 | 行为 |
|---|---|
| 普通用户账号，如 `captest002` | 必须已存在 USER 账号，不自动创建。 |
| `APP:<K-Code>` | 按 APP 账号解析，不存在则自动创建。 |
| 形如 `K...` 且 USER 不存在 | 按 APP K-Code 自动创建。 |
| `--group <groupCode>` | 使用 Skill Group code，不使用数据库 id。 |

不传 `--public`、`--read`、`--maint`、`--owner`、`--group` 时，仅发布者拥有 OWNER 权限。

## 8. 发布状态

API：

```http
GET /api/v1/client/skills/{namespace}/{slug}/versions/{version}/publish-status
```

CLI：

```bash
hskill publish-status global/demo-skill \
  --version 1.0.0 \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}" \
  --json
```

返回字段：

| 字段 | 说明 |
|---|---|
| `status` | 版本状态：`UPLOADED`、`SCANNING`、`PUBLISHED`、`BLOCKED`、`FAILED`、`YANKED`。 |
| `scanStatus` | 扫描状态：`PENDING`、`SCANNING`、`PASS`、`WARN`、`FAILED`、`BLOCKED`。 |
| `downloadReady` | 是否可下载。 |
| `listed` | 是否已上架。 |
| `failureReason` | 失败原因，如 `SCAN_WARN`、`SCAN_FAILED`、`SCAN_BLOCKED`、`SCAN_TIMEOUT`。 |
| `canRetry` | 当前身份是否可重试。 |
| `canManage` | 当前身份是否可管理。 |

## 9. 发布重试

API：

```http
POST /api/v1/client/skills/{namespace}/{slug}/versions/{version}/publish-retry
Content-Type: application/json
```

示例：

```bash
curl -fsS \
  -H "X-Access-Token: ${USER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"scanTimeoutSeconds":120,"publicAccess":true}' \
  "${SKILLHUB_BASE_URL}/api/v1/client/skills/global/demo-skill/versions/1.0.0/publish-retry"
```

CLI：

```bash
hskill publish-retry global/demo-skill \
  --version 1.0.0 \
  --timeout 120 \
  --public \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}" \
  --json
```

只有 OWNER、MAINTAINER 或具备平台管理权限的账号可以重试。

## 10. 安装

安装必须使用 `namespace/slug`，不支持 `hskill install demo` 自动映射为 `global/demo`。

### 10.1 安装最新版本

API：

```http
GET /api/v1/skills/{namespace}/{slug}/download
```

CLI：

```bash
hskill install global/demo-skill \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}" \
  --workdir ./haier-skillhub
```

### 10.2 安装指定版本

API：

```http
GET /api/v1/skills/{namespace}/{slug}/versions/{version}/download
```

CLI：

```bash
hskill install global/demo-skill \
  --version 1.0.0 \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --X-Access-Token "${USER_TOKEN}" \
  --workdir ./haier-skillhub
```

安装结果：

```text
<workdir>/skills/<slug>/
<workdir>/haier-skillhub.lock.json
```

常用本地命令：

```bash
hskill list --workdir ./haier-skillhub
hskill update global/demo-skill --workdir ./haier-skillhub --X-Access-Token "${USER_TOKEN}"
hskill uninstall demo-skill --workdir ./haier-skillhub
```

## 11. CLI 配置

保存默认 SkillHub 地址和本地目录：

```bash
hskill config \
  --skillhub-url "${SKILLHUB_BASE_URL}" \
  --workdir ./haier-skillhub \
  --dir skills
```

CLI 配置不保存 token。也可以用环境变量：

```bash
export HAIER_SKILLHUB_REGISTRY="${SKILLHUB_BASE_URL}"
export HAIER_SKILLHUB_WORKDIR="./haier-skillhub"
export HAIER_SKILLHUB_DIR="skills"
```

## 12. 常见错误

| 场景 | 现象 | 处理 |
|---|---|---|
| ZIP 根目录没有 `SKILL.md` | 包校验失败，提示 `Missing required file: SKILL.md at root` | 调整 ZIP 结构，确保 `SKILL.md` 在根目录。 |
| 扫描 WARN/FAILED/BLOCKED | 发布 Skill 返回失败原因，`listed=false` | 修复 Skill 包后重新发布或执行 `publish-retry`。 |
| 扫描超时 | `failureReason=SCAN_TIMEOUT`，不上架 | 查询状态或重试发布。 |
| 安装受控 Skill 返回 403 | 当前身份没有安装权限 | 让 OWNER/MAINTAINER 授予账号或分组 READER 权限。 |
| 发布已有 Skill 返回 403 | 当前身份不是 OWNER/MAINTAINER | 联系 Skill OWNER 授权 MAINTAINER/OWNER。 |
| CLI plain slug 安装失败 | `install requires <namespace>/<slug>` | 使用 `global/demo-skill` 形式。 |
| USER 授权账号不存在 | 授权失败 | 先让用户通过 gateway 访问投影账号，或由管理员创建账号。 |
