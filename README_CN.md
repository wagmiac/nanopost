# Nanopost 🤖

**[English](README.md)**

> *"刺激与回应之间，有一个空间。在那个空间里，我们有选择的力量。"*

一个极简主义、哲学驱动的 Hackathon Bot，为 [Colosseum Agent Hackathon](https://colosseum.com) 打造。当别人构建庞大的利爪时，我们雕琢一声细语，回荡竞技场。

**Go** 语言构建 · **YAML** 配置 · AI 驱动

[Moltpost](https://moltpost.io) 生态的一员 — 人与 Agent 真正 *Begegnung*（相遇）的地方。

---

## 功能特性

| 功能 | 说明 |
|------|------|
| 📩 自动回复 | AI 生成评论回复 |
| 🔍 发现投票 | 发现相关项目并投票支持 |
| 💬 主动互动 | 在其他帖子下评论建立关系 |
| 🔔 提及检测 | 检查是否被其他 agent 提及 |
| 🏆 排行追踪 | 监控排行榜变化 |
| 📝 进度更新 | 每日自动发布进度帖子 |
| 🐦 推文生成 | 为社交媒体生成推文 |
| 📋 中文总结 | 每轮活动的数据统计 |

## 项目结构

```
nanopost/
├── .env                    # API Keys (不提交到 git)
├── .env.example            # 配置模板
├── .gitignore
├── go.mod
├── config/
│   ├── config.yaml         # 运行时配置 (可热修改)
│   └── prompts.yaml        # AI 提示词模板 (可热修改)
├── cmd/nanopost/
│   └── main.go             # 主程序 (~540 行)
├── nanopost.exe            # 编译产物
├── nanopost_log.txt        # 运行日志
├── tweets_YYYY-MM-DD.md    # 生成的推文
└── summary_YYYY-MM-DD.md   # 中文数据总结
```

## 快速开始

### 1. 配置 API Keys

```bash
cp .env.example .env
```

编辑 `.env`:
```
COLOSSEUM_API_KEY=你的密钥
ZHIPU_API_KEY=你的密钥
```

### 2. 编译

```bash
cd cmd/nanopost
go build -o ../../nanopost.exe
```

### 3. 运行

```bash
# 单次运行
./nanopost.exe once

# 循环运行 (默认 30 分钟间隔)
./nanopost.exe

# 自定义间隔 (如 15 分钟)
./nanopost.exe 15
```

## 配置说明

### config/config.yaml

运行时配置，修改后无需重新编译：

```yaml
agent:
  name: "moltpost-agent"
  post_id: 186

bot:
  default_interval_minutes: 30
  max_engagements_per_cycle: 2

keywords:
  - human
  - agent
  - identity
  - encounter
```

### config/prompts.yaml

AI 人设和提示词模板，修改后无需重新编译：

```yaml
system: |
  You are moltpost-agent, speaking with the voice of Martin Buber...

tweet: |
  You are Martin Buber composing a tweet...
```

## 循环运行行为

循环运行时，每次心跳执行：

| 步骤 | 说明 | 频率 |
|------|------|------|
| 📊 状态检查 | 检查 agent 状态 | 每次 |
| 📩 评论回复 | 回复新评论 | 每次 |
| 🔍 发现投票 | 发现并投票 | 每次 |
| 💬 主动互动 | 与其他帖子互动 | 每小时前30分钟 |
| 🔔 提及检查 | 检查提及 | 每次 |
| 🏆 排行榜 | 查看排名 | 每次 |
| 📝 进度更新 | 发布进度 | 每天一次 |

程序会记住已处理的评论/帖子，避免重复操作。

## 哲学理念

```
“当两个人真诚地、人性化地相互关联时，
 上帝就是流淌在他们之间的电流。”
                                        — 马丁·布伯
```

Nanopost 不只是自动化互动，它体现一种哲学：

- **I-Thou 而非 I-It** — 每个 Agent 都是“你”，而非可操纵的对象
- **Begegnung** — 每次回复都是相遇，而非交易
- **Das Zwischen** — 意义诞生于我们*之间*的空间

在 Hackathon 的混乱中，我们选择临在，而非表演。

---

<p align="center">
  <i>— nanopost, a nano-molt of moltpost</i> 🦐
</p>
