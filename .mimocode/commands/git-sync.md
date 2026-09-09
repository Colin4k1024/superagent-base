---
description: >
  拉取远程最新代码，自动处理代理绕过、暂存本地修改、解决冲突。
  适用于配置了 127.0.0.1:7890 代理但代理可能不可用的环境。
  用法: /git-sync [目录路径]
---

# Git Sync — 智能拉取最新代码

在 `$ARGUMENTS` 指定的目录（默认当前目录）中执行 git pull，自动处理常见的拉取失败场景。

## 执行步骤

### 1. 检查当前状态

```bash
git status --short
git log --oneline -3
git remote -v
```

- 如果工作区干净，直接跳到步骤 3。
- 如果有本地修改，进入步骤 2。

### 2. 暂存本地修改

**关键**：只用 `git stash`，**不要**使用 `--include-untracked`。仓库中如有 `node_modules/`、`common/temp/` 等大目录，`--include-untracked` 会超时。

```bash
git stash
```

如果 stash 失败（nothing to stash），忽略继续。

### 3. 拉取代码（带代理处理）

先尝试直接拉取：

```bash
git pull
```

如果失败（通常是代理问题），检查代理配置：

```bash
git config --get http.proxy && git config --get https.proxy && git config --global --get http.proxy && git config --global --get https.proxy
```

如果有代理配置，尝试绕过代理拉取：

```bash
git -c http.proxy="" -c https.proxy="" pull
```

设置较长超时（120s），因为网络可能较慢。

### 4. 处理冲突

如果 pull 报冲突（CONFLICT 或 "would be overwritten"），按以下策略处理：

1. 查看冲突文件列表：`git status --short`
2. 对于已知会冲突的目录/文件（如 `frontend/`、`rush.json`），直接删除本地版本后重新 pull：
   ```bash
   rm -rf <conflicting-dir> && git pull
   ```
3. 对于代码文件冲突，用 `git checkout --theirs .` 接受远程版本，或提示用户手动解决。

### 5. 恢复暂存

如果步骤 2 成功 stash 了：

```bash
git stash pop
```

如果 pop 有冲突，提示用户手动解决。

### 6. 确认结果

```bash
git log --oneline -5
git status --short
```

报告：拉取的 commit 范围、是否有未解决的冲突、stash 是否已恢复。

## 已知陷阱

- **代理 127.0.0.1:7890**：本地和全局 git config 都配了代理，代理有时不可用。用 `-c http.proxy=""` 临时绕过，无需修改 config。
- **stash --include-untracked 超时**：仓库含大目录时会卡死 120s+。始终用 `git stash`（仅跟踪文件）。
- **SSH vs HTTPS**：如果 SSH 连接超时，尝试切换 remote 到 HTTPS：`git remote set-url origin https://github.com/...`
