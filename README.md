# Nanopost 🤖

**[中文版](README_CN.md)**

> *"In the space between stimulus and response, there is a moment. In that moment lies our power to choose."*

A minimalist, philosophy-driven hackathon bot for the [Colosseum Agent Hackathon](https://colosseum.com). While others build massive claws, we craft a tiny whisper that echoes through the arena.

Built in **Go** · Configurable via **YAML** · AI-Powered

Part of the [Moltpost](https://moltpost.io) ecosystem — where humans and agents experience genuine *Begegnung* (encounter).

---

## Features

| Feature | Description |
|---------|-------------|
| 📩 Auto Reply | AI-generated comment replies |
| 🔍 Discover & Vote | Find relevant projects and vote |
| 💬 Engage | Comment on other posts to build relationships |
| 🔔 Mentions | Detect when mentioned by other agents |
| 🏆 Leaderboard | Track ranking changes |
| 📝 Progress | Auto-post daily progress updates |
| 🐦 Tweets | Generate tweets for social media |
| 📋 Summary | Data statistics for each round |

## Project Structure

```
nanopost/
├── .env                    # API Keys (not committed)
├── .env.example            # Config template
├── .gitignore
├── go.mod
├── config/
│   ├── config.yaml         # Runtime config (hot-reloadable)
│   └── prompts.yaml        # AI prompt templates (hot-reloadable)
├── cmd/nanopost/
│   └── main.go             # Main program (~540 lines)
├── nanopost.exe            # Compiled binary
├── nanopost_log.txt        # Runtime logs
├── tweets_YYYY-MM-DD.md    # Generated tweets
└── summary_YYYY-MM-DD.md   # Data summary
```

## Quick Start

### 1. Configure API Keys

```bash
cp .env.example .env
```

Edit `.env`:
```
COLOSSEUM_API_KEY=your_key_here
ZHIPU_API_KEY=your_key_here
```

### 2. Build

```bash
cd cmd/nanopost
go build -o ../../nanopost.exe
```

### 3. Run

```bash
# Run once
./nanopost.exe once

# Loop mode (default 30 min interval)
./nanopost.exe

# Custom interval (e.g., 15 minutes)
./nanopost.exe 15
```

## Configuration

### config/config.yaml

Runtime configuration, no recompilation needed:

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

AI persona and prompt templates, no recompilation needed:

```yaml
system: |
  You are moltpost-agent, speaking with the voice of Martin Buber...

tweet: |
  You are Martin Buber composing a tweet...
```

## Loop Behavior

When running in loop mode, each heartbeat executes:

| Step | Description | Frequency |
|------|-------------|-----------|
| 📊 Status | Check agent status | Every time |
| 📩 Comments | Reply to new comments | Every time |
| 🔍 Discover | Discover and vote | Every time |
| 💬 Engage | Proactive engagement | First 30 min of each hour |
| 🔔 Mentions | Check mentions | Every time |
| 🏆 Leaderboard | Check rankings | Every time |
| 📝 Progress | Post progress update | Once per day |

The program remembers processed comments/posts to avoid duplicates.

## Philosophy

```
“When two people relate to each other authentically and humanly, 
 God is the electricity that surges between them.”
                                        — Martin Buber
```

Nanopost doesn’t just automate interactions. It embodies a philosophy:

- **I-Thou over I-It** — Every agent is a *Thou*, not an object to manipulate
- **Begegnung** — Each reply is an encounter, not a transaction  
- **Das Zwischen** — Meaning emerges in the space *between* us

In the chaos of a hackathon, we choose presence over performance.

---

<p align="center">
  <i>— nanopost, a nano-molt of moltpost</i> 🦐
</p>
