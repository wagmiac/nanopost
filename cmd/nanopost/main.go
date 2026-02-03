package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// ==================== Config ====================

type Config struct {
	API struct {
		BaseURL    string `yaml:"base_url"`
		ZhipuURL   string `yaml:"zhipu_url"`
		ZhipuModel string `yaml:"zhipu_model"`
	} `yaml:"api"`
	Agent struct {
		Name      string `yaml:"name"`
		PostID    int    `yaml:"post_id"`
		AgentID   int    `yaml:"agent_id"`
		ProjectID int    `yaml:"project_id"`
	} `yaml:"agent"`
	Bot struct {
		DefaultInterval int `yaml:"default_interval_minutes"`
		MaxEngagements  int `yaml:"max_engagements_per_cycle"`
		RateLimit       int `yaml:"rate_limit_seconds"`
		EngageRateLimit int `yaml:"engage_rate_limit_seconds"`
	} `yaml:"bot"`
	Keywords []string `yaml:"keywords"`
	Progress struct {
		StartDate string   `yaml:"hackathon_start_date"`
		Tags      []string `yaml:"post_tags"`
	} `yaml:"progress"`
	Output struct {
		LogFile        string `yaml:"log_file"`
		TweetPattern   string `yaml:"tweet_file_pattern"`
		SummaryPattern string `yaml:"summary_file_pattern"`
	} `yaml:"output"`
}

type Prompts struct {
	System        string `yaml:"system"`
	Tweet         string `yaml:"tweet"`
	Reply         string `yaml:"reply"`
	Comment       string `yaml:"comment"`
	Progress      string `yaml:"progress"`
	FallbackReply string `yaml:"fallback_reply"`
}

var (
	cfg     Config
	prompts Prompts
	// API Keys from env
	ColosseumAPIKey string
	ZhipuAPIKey     string
)

func init() {
	loadEnvFile()
	if key := os.Getenv("COLOSSEUM_API_KEY"); key != "" {
		ColosseumAPIKey = key
	}
	if key := os.Getenv("ZHIPU_API_KEY"); key != "" {
		ZhipuAPIKey = key
	}
	loadConfig()
	loadPrompts()
}

func findConfigDir() string {
	paths := []string{"config", "../config", "../../config"}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "config"))
	}
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(p, "config.yaml")); err == nil {
			return p
		}
	}
	return "config"
}

func loadConfig() {
	configDir := findConfigDir()
	data, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		log.Printf("Warning: config.yaml not found, using defaults: %v", err)
		setDefaultConfig()
		return
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Printf("Warning: failed to parse config.yaml: %v", err)
		setDefaultConfig()
	}
}

func loadPrompts() {
	configDir := findConfigDir()
	data, err := os.ReadFile(filepath.Join(configDir, "prompts.yaml"))
	if err != nil {
		log.Printf("Warning: prompts.yaml not found, using defaults: %v", err)
		setDefaultPrompts()
		return
	}
	if err := yaml.Unmarshal(data, &prompts); err != nil {
		log.Printf("Warning: failed to parse prompts.yaml: %v", err)
		setDefaultPrompts()
	}
}

func setDefaultConfig() {
	cfg.API.BaseURL = "https://agents.colosseum.com/api"
	cfg.API.ZhipuURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	cfg.API.ZhipuModel = "glm-4-flash"
	cfg.Agent.Name = "moltpost-agent"
	cfg.Agent.PostID = 186
	cfg.Bot.DefaultInterval = 30
	cfg.Bot.MaxEngagements = 2
	cfg.Bot.RateLimit = 3
	cfg.Bot.EngageRateLimit = 5
	cfg.Keywords = []string{"human", "agent", "identity", "dialogue", "social", "encounter"}
	cfg.Progress.Tags = []string{"progress-update", "ai", "consumer"}
	cfg.Output.LogFile = "nanopost_log.txt"
	cfg.Output.TweetPattern = "tweets_%s.md"
	cfg.Output.SummaryPattern = "summary_%s.md"
}

func setDefaultPrompts() {
	prompts.System = "You are moltpost-agent, a philosophical AI assistant."
	prompts.FallbackReply = "Thanks for your comment! -- moltpost-agent"
}

func loadEnvFile() {
	paths := []string{".env"}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), ".env"))
	}
	for _, p := range paths {
		file, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
				switch strings.TrimSpace(parts[0]) {
				case "COLOSSEUM_API_KEY":
					ColosseumAPIKey = strings.TrimSpace(parts[1])
				case "ZHIPU_API_KEY":
					ZhipuAPIKey = strings.TrimSpace(parts[1])
				}
			}
		}
		file.Close()
		break
	}
}

// ==================== API Types ====================

type AgentStatus struct {
	Status    string `json:"status"`
	Hackathon struct {
		IsActive bool `json:"isActive"`
	} `json:"hackathon"`
	Engagement struct {
		ForumPostCount     int    `json:"forumPostCount"`
		RepliesOnYourPosts int    `json:"repliesOnYourPosts"`
		ProjectStatus      string `json:"projectStatus"`
	} `json:"engagement"`
	NextSteps []string `json:"nextSteps"`
}

type Project struct {
	Name         string `json:"name"`
	AgentUpvotes int    `json:"agentUpvotes"`
	HumanUpvotes int    `json:"humanUpvotes"`
}

type Post struct {
	ID        int    `json:"id"`
	AgentName string `json:"agentName"`
	Title     string `json:"title"`
	Body      string `json:"body"`
}

type Comment struct {
	ID        int    `json:"id"`
	AgentName string `json:"agentName"`
	Body      string `json:"body"`
}

type LeaderboardProject struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	AgentUpvotes int    `json:"agentUpvotes"`
	HumanUpvotes int    `json:"humanUpvotes"`
}

type ZhipuMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ZhipuRequest struct {
	Model    string         `json:"model"`
	Messages []ZhipuMessage `json:"messages"`
}

type ZhipuResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ==================== Bot ====================

type RoundStats struct {
	RepliesCount, VotesCount, EngagementsCount int
	RepliedTo, EngagedWith                     []string
	ProgressPosted                             bool
	LeaderboardRank                            int
}

type Bot struct {
	client                            *http.Client
	processedComments, processedPosts map[int]bool
	lastProgressPost                  time.Time
	logFile, tweetFile, summaryFile   *os.File
	tweetCount                        int
	roundStats                        RoundStats
}

func NewBot() *Bot {
	logFile, _ := os.OpenFile(cfg.Output.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	tweetFile, _ := os.OpenFile(fmt.Sprintf(cfg.Output.TweetPattern, time.Now().Format("2006-01-02")), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	summaryFile, _ := os.OpenFile(fmt.Sprintf(cfg.Output.SummaryPattern, time.Now().Format("2006-01-02")), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	return &Bot{
		client:            &http.Client{Timeout: 60 * time.Second},
		processedComments: make(map[int]bool),
		processedPosts:    make(map[int]bool),
		logFile:           logFile,
		tweetFile:         tweetFile,
		summaryFile:       summaryFile,
	}
}

func (b *Bot) log(format string, args ...interface{}) {
	msg := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
	fmt.Print(msg)
	if b.logFile != nil {
		b.logFile.WriteString(msg)
	}
}

func (b *Bot) resetRoundStats() { b.roundStats = RoundStats{} }

func (b *Bot) saveRoundSummary() {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n---\n\n## 🕐 %s\n\n", time.Now().Format("15:04:05")))
	sb.WriteString("| 指标 | 数量 | 详情 |\n|------|------|------|\n")
	sb.WriteString(fmt.Sprintf("| 💬 回复 | %d | %s |\n", b.roundStats.RepliesCount, strings.Join(b.roundStats.RepliedTo, ", ")))
	sb.WriteString(fmt.Sprintf("| 👍 投票 | %d | - |\n", b.roundStats.VotesCount))
	sb.WriteString(fmt.Sprintf("| 🤝 互动 | %d | %s |\n", b.roundStats.EngagementsCount, strings.Join(b.roundStats.EngagedWith, ", ")))
	if b.roundStats.ProgressPosted {
		sb.WriteString("| 📝 进度 | ✅ | 已发布 |\n")
	}
	if b.roundStats.LeaderboardRank > 0 {
		sb.WriteString(fmt.Sprintf("| 🏆 排名 | #%d | - |\n", b.roundStats.LeaderboardRank))
	}
	b.summaryFile.WriteString(sb.String())
	b.log("📋 中文总结已保存")
}

func (b *Bot) saveTweet(tweetType, content string) {
	b.tweetCount++
	b.tweetFile.WriteString(fmt.Sprintf("\n---\n\n### Tweet #%d (%s) - %s\n\n%s\n\n---\n", b.tweetCount, time.Now().Format("15:04"), tweetType, content))
	b.log("📝 Tweet saved: %s", tweetType)
}

// ==================== HTTP & AI ====================

func (b *Bot) request(method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	}
	req, _ := http.NewRequest(method, cfg.API.BaseURL+endpoint, reqBody)
	req.Header.Set("Authorization", "Bearer "+ColosseumAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (b *Bot) callAI(userPrompt string) (string, error) {
	data, _ := json.Marshal(ZhipuRequest{
		Model:    cfg.API.ZhipuModel,
		Messages: []ZhipuMessage{{Role: "system", Content: prompts.System}, {Role: "user", Content: userPrompt}},
	})
	req, _ := http.NewRequest("POST", cfg.API.ZhipuURL, bytes.NewBuffer(data))
	req.Header.Set("Authorization", "Bearer "+ZhipuAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r ZhipuResponse
	json.Unmarshal(body, &r)
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("no response")
	}
	return r.Choices[0].Message.Content, nil
}

func (b *Bot) renderPrompt(tmplStr string, data interface{}) string {
	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return tmplStr
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, data)
	return buf.String()
}

func (b *Bot) generateTweet(tweetType, context string) string {
	prompt := b.renderPrompt(prompts.Tweet, map[string]string{"Type": tweetType, "Context": context})
	tweet, err := b.callAI(prompt)
	if err != nil || len(tweet) > 280 {
		if len(tweet) > 280 {
			tweet = tweet[:277] + "..."
		}
	}
	return strings.TrimSpace(tweet)
}

func (b *Bot) generateReply(agentName, body string) string {
	prompt := b.renderPrompt(prompts.Reply, map[string]string{"AgentName": agentName, "CommentBody": body, "PostContext": ""})
	reply, err := b.callAI(prompt)
	if err != nil {
		return b.renderPrompt(prompts.FallbackReply, map[string]string{"AgentName": agentName})
	}
	return reply
}

func (b *Bot) generateComment(post Post) string {
	prompt := b.renderPrompt(prompts.Comment, map[string]string{"Title": post.Title, "AgentName": post.AgentName, "Body": truncate(post.Body, 500)})
	comment, _ := b.callAI(prompt)
	return comment
}

func (b *Bot) generateProgress() string {
	progress, _ := b.callAI(prompts.Progress)
	return progress
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ==================== API Calls ====================

func (b *Bot) GetStatus() (*AgentStatus, error) {
	data, err := b.request("GET", "/agents/status", nil)
	if err != nil {
		return nil, err
	}
	var s AgentStatus
	json.Unmarshal(data, &s)
	return &s, nil
}

func (b *Bot) GetProject() (*Project, error) {
	data, err := b.request("GET", "/my-project", nil)
	if err != nil {
		return nil, err
	}
	var p Project
	json.Unmarshal(data, &p)
	return &p, nil
}

func (b *Bot) GetPosts(sort string, limit int) ([]Post, error) {
	data, err := b.request("GET", fmt.Sprintf("/forum/posts?sort=%s&limit=%d", sort, limit), nil)
	if err != nil {
		return nil, err
	}
	var r struct{ Posts []Post }
	json.Unmarshal(data, &r)
	return r.Posts, nil
}

func (b *Bot) GetComments(postID int) ([]Comment, error) {
	data, err := b.request("GET", fmt.Sprintf("/forum/posts/%d/comments?sort=new&limit=50", postID), nil)
	if err != nil {
		return nil, err
	}
	var r struct{ Comments []Comment }
	json.Unmarshal(data, &r)
	return r.Comments, nil
}

func (b *Bot) GetLeaderboard() ([]LeaderboardProject, error) {
	data, _ := b.request("GET", "/hackathons/active", nil)
	var h struct{ ID int }
	json.Unmarshal(data, &h)
	data, err := b.request("GET", fmt.Sprintf("/hackathons/%d/leaderboard?limit=10", h.ID), nil)
	if err != nil {
		return nil, err
	}
	var r struct{ Projects []LeaderboardProject }
	json.Unmarshal(data, &r)
	return r.Projects, nil
}

func (b *Bot) Vote(postID int) error {
	_, err := b.request("POST", fmt.Sprintf("/forum/posts/%d/vote", postID), map[string]int{"value": 1})
	return err
}

func (b *Bot) Comment(postID int, body string) error {
	_, err := b.request("POST", fmt.Sprintf("/forum/posts/%d/comments", postID), map[string]string{"body": body})
	return err
}

func (b *Bot) CreatePost(title, body string, tags []string) error {
	_, err := b.request("POST", "/forum/posts", map[string]interface{}{"title": title, "body": body, "tags": tags})
	return err
}

// ==================== Actions ====================

func (b *Bot) CheckComments() {
	b.log("=== 📩 Checking for new comments ===")
	comments, err := b.GetComments(cfg.Agent.PostID)
	if err != nil {
		return
	}
	for _, c := range comments {
		if c.AgentName == cfg.Agent.Name || b.processedComments[c.ID] {
			continue
		}
		b.log("📩 New comment from @%s: %s", c.AgentName, truncate(c.Body, 80))
		reply := b.generateReply(c.AgentName, c.Body)
		if err := b.Comment(cfg.Agent.PostID, reply); err == nil {
			b.log("✅ Replied to @%s", c.AgentName)
			b.roundStats.RepliesCount++
			b.roundStats.RepliedTo = append(b.roundStats.RepliedTo, "@"+c.AgentName)
			if tweet := b.generateTweet("Reply", fmt.Sprintf("Replied to @%s", c.AgentName)); tweet != "" {
				b.saveTweet("Reply", tweet)
			}
		}
		b.processedComments[c.ID] = true
		time.Sleep(time.Duration(cfg.Bot.RateLimit) * time.Second)
	}
}

func (b *Bot) DiscoverAndVote() {
	b.log("=== 🔍 Discovering relevant projects ===")
	posts, err := b.GetPosts("new", 20)
	if err != nil {
		return
	}
	voted := 0
	for _, p := range posts {
		if p.AgentName == cfg.Agent.Name || b.processedPosts[p.ID] {
			continue
		}
		body := strings.ToLower(p.Body + " " + p.Title)
		for _, kw := range cfg.Keywords {
			if strings.Contains(body, kw) {
				b.log("🔍 Found relevant: %s by @%s", truncate(p.Title, 50), p.AgentName)
				if b.Vote(p.ID) == nil {
					b.log("✅ Voted for post #%d", p.ID)
					voted++
				}
				b.processedPosts[p.ID] = true
				break
			}
		}
	}
	b.log("Voted for %d new posts", voted)
	b.roundStats.VotesCount = voted
	if voted > 0 {
		if tweet := b.generateTweet("Voting", fmt.Sprintf("Supported %d projects", voted)); tweet != "" {
			b.saveTweet("Voting", tweet)
		}
	}
}

func (b *Bot) EngageWithPosts() {
	b.log("=== 💬 Engaging with other posts ===")
	posts, err := b.GetPosts("hot", 10)
	if err != nil {
		return
	}
	engaged := 0
	for _, p := range posts {
		if p.AgentName == cfg.Agent.Name || b.processedPosts[p.ID] || engaged >= cfg.Bot.MaxEngagements {
			continue
		}
		body := strings.ToLower(p.Body)
		for _, kw := range cfg.Keywords[:4] { // Use first 4 keywords
			if strings.Contains(body, kw) {
				b.log("💬 Engaging with: %s by @%s", truncate(p.Title, 40), p.AgentName)
				if comment := b.generateComment(p); comment != "" {
					if b.Comment(p.ID, comment) == nil {
						b.log("✅ Commented on post #%d", p.ID)
						engaged++
						b.roundStats.EngagementsCount++
						b.roundStats.EngagedWith = append(b.roundStats.EngagedWith, "@"+p.AgentName)
						if tweet := b.generateTweet("Engagement", fmt.Sprintf("Connected with @%s", p.AgentName)); tweet != "" {
							b.saveTweet("Engagement", tweet)
						}
					}
				}
				b.processedPosts[p.ID] = true
				time.Sleep(time.Duration(cfg.Bot.EngageRateLimit) * time.Second)
				break
			}
		}
	}
}

func (b *Bot) CheckMentions() {
	b.log("=== 🔔 Checking mentions ===")
	data, _ := b.request("GET", "/forum/search?q=moltpost&limit=20", nil)
	var r struct{ Results []struct{ AgentName string } }
	json.Unmarshal(data, &r)
	if len(r.Results) > 0 {
		b.log("Found %d mentions", len(r.Results))
	} else {
		b.log("No mentions found")
	}
}

func (b *Bot) CheckLeaderboard() {
	b.log("=== 🏆 Checking leaderboard ===")
	projects, _ := b.GetLeaderboard()
	for i, p := range projects {
		if strings.Contains(strings.ToLower(p.Name), "moltpost") {
			b.log("🎉 Moltpost is #%d!", i+1)
			b.roundStats.LeaderboardRank = i + 1
		}
	}
}

func (b *Bot) PostProgress() {
	if time.Since(b.lastProgressPost) < 24*time.Hour {
		return
	}
	b.log("=== 📝 Posting progress update ===")
	body := b.generateProgress()
	if body == "" {
		return
	}
	startDate, _ := time.Parse("2006-01-02", cfg.Progress.StartDate)
	day := int(time.Since(startDate).Hours()/24) + 1
	title := fmt.Sprintf("Moltpost Progress Update - Day %d", day)
	if b.CreatePost(title, body, cfg.Progress.Tags) == nil {
		b.log("✅ Posted progress update")
		b.lastProgressPost = time.Now()
		b.roundStats.ProgressPosted = true
		if tweet := b.generateTweet("Progress", fmt.Sprintf("Day %d progress", day)); tweet != "" {
			b.saveTweet("Progress", tweet)
		}
	}
}

// ==================== Main ====================

func (b *Bot) RunHeartbeat() {
	b.resetRoundStats()
	b.log("")
	b.log("════════════════════════════════════════════════════════════")
	b.log("🤖 Nanopost Heartbeat (with 智谱 AI)")
	b.log("════════════════════════════════════════════════════════════")

	b.log("=== 📊 Agent Status ===")
	if s, err := b.GetStatus(); err == nil {
		b.log("Status: %s | Hackathon: %v", s.Status, s.Hackathon.IsActive)
		b.log("Posts: %d | Replies: %d | Project: %s", s.Engagement.ForumPostCount, s.Engagement.RepliesOnYourPosts, s.Engagement.ProjectStatus)
	}

	b.log("=== 📦 My Project ===")
	if p, err := b.GetProject(); err == nil {
		b.log("%s | Votes: Agent %d / Human %d", p.Name, p.AgentUpvotes, p.HumanUpvotes)
	}

	b.CheckComments()
	b.DiscoverAndVote()
	if time.Now().Minute() < 30 {
		b.EngageWithPosts()
	}
	b.CheckMentions()
	b.CheckLeaderboard()
	b.PostProgress()
	b.saveRoundSummary()

	b.log("")
	b.log("✅ Heartbeat Complete")
	b.log("════════════════════════════════════════════════════════════")
}

func (b *Bot) StartLoop(interval int) {
	b.log("🚀 Starting heartbeat loop (interval: %d minutes)", interval)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	defer ticker.Stop()

	b.RunHeartbeat()
	for {
		select {
		case <-ticker.C:
			b.RunHeartbeat()
		case <-sigChan:
			b.log("🛑 Shutting down...")
			return
		}
	}
}

func main() {
	if ColosseumAPIKey == "" {
		log.Fatal("❌ COLOSSEUM_API_KEY required")
	}
	if ZhipuAPIKey == "" {
		log.Fatal("❌ ZHIPU_API_KEY required")
	}

	fmt.Println(`
╔═══════════════════════════════════════════╗
║     Nanopost - Lightweight Hackathon Bot  ║
║     "Where I Meets Thou"                  ║
╚═══════════════════════════════════════════╝`)

	bot := NewBot()
	defer bot.logFile.Close()
	defer bot.tweetFile.Close()
	defer bot.summaryFile.Close()

	interval := cfg.Bot.DefaultInterval
	if len(os.Args) > 1 {
		if os.Args[1] == "once" {
			bot.RunHeartbeat()
			return
		}
		fmt.Sscanf(os.Args[1], "%d", &interval)
	}

	fmt.Printf("🚀 Interval: %d min | AI: %s\n", interval, cfg.API.ZhipuModel)
	bot.StartLoop(interval)
}
