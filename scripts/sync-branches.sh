#!/usr/bin/env bash
# sync-branches.sh — 双端分支同步工具
#
# 分支模型:
#   main     → GitHub (origin)     公开代码
#   internal → GitLab (haier-origin) 叠加内部工具/文档/CI
#
# 用法:
#   ./scripts/sync-branches.sh merge    # 将 main 的新 commit 合并到 internal
#   ./scripts/sync-branches.sh push     # 推送 main→GitHub, internal→GitLab
#   ./scripts/sync-branches.sh status   # 查看两分支差异

set -euo pipefail

GITHUB_REMOTE="origin"
GITLAB_REMOTE="haier-origin"

case "${1:-status}" in
  merge)
    echo ">>> 将 main 的更新合并到 internal..."
    git checkout internal
    git merge main --no-edit
    git checkout main
    echo "✓ 合并完成。internal 已包含 main 的最新通用改动。"
    ;;

  push)
    echo ">>> 推送 main → GitHub ($GITHUB_REMOTE)..."
    git push "$GITHUB_REMOTE" main

    echo ">>> 推送 internal → GitLab ($GITLAB_REMOTE) as main..."
    git push "$GITLAB_REMOTE" internal:main

    echo "✓ 双端推送完成。"
    ;;

  status)
    echo "=== main (GitHub) ==="
    git log --oneline -5 main
    echo ""
    echo "=== internal (GitLab) ==="
    git log --oneline -5 internal
    echo ""
    echo "--- internal 相对 main 多出的 commit ---"
    git log --oneline main..internal
    echo ""
    echo "--- main 相对 internal 多出的 commit (需要 merge) ---"
    git log --oneline internal..main
    ;;

  *)
    echo "用法: $0 {merge|push|status}"
    exit 1
    ;;
esac
