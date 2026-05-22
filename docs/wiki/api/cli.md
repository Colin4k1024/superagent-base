# CLI 工具 (sactl)

## 安装

```bash
cd backend
go build -o ../bin/sactl ./cmd/sactl/
```

## 命令

### Skill 管理

```bash
# 搜索 Skill
sactl skill search "数据分析"

# 安装 Skill
sactl skill install datetime@1.0.0

# 列出已安装
sactl skill list

# 卸载
sactl skill uninstall datetime
```

### Agent 管理

```bash
# 从 YAML 加载 Agent（占位，后续完善）
sactl agent apply -f configs/agents/my-agent.yaml
```

## 配置

通过环境变量配置 SkillsHub 端点：

```bash
export SKILLS_HUB_URL=http://your-skillshub:8080
export SKILLS_HUB_TOKEN=your-token
```
