package story_persist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dili/manager/backend/models"

	"gorm.io/gorm"
)

// GenerateRequest 管理端 AI 生成故事正文。
type GenerateRequest struct {
	Theme         string `json:"theme"`
	Title         string `json:"title"`
	PoolKind      string `json:"pool_kind"`
	CanonicalKey  string `json:"canonical_key"`
	NarrationMode string `json:"narration_mode"`
	Mode          string `json:"mode"`
	AgeBand       string `json:"age_band"`
	LLMConfigID   string `json:"llm_config_id"`
	ExtraPrompt   string `json:"extra_prompt"`
}

// GenerateResult AI 生成结果（不落库，供前端填表）。
type GenerateResult struct {
	Title        string   `json:"title"`
	ThemeKey     string   `json:"theme_key"`
	CanonicalKey string   `json:"canonical_key"`
	FullText     string   `json:"full_text"`
	Segments     []string `json:"segments"`
	Model        string   `json:"model,omitempty"`
}

func GenerateStoryText(ctx context.Context, db *gorm.DB, req GenerateRequest) (*GenerateResult, error) {
	theme := strings.TrimSpace(req.Theme)
	if theme == "" {
		theme = strings.TrimSpace(req.CanonicalKey)
	}
	if theme == "" {
		return nil, errors.New("theme 不能为空")
	}
	_, cfgMap, err := loadLLMConfig(db, req.LLMConfigID)
	if err != nil {
		return nil, err
	}
	baseURL := firstNonEmpty(cfgMap, "base_url", "api_url")
	apiKey := firstNonEmpty(cfgMap, "api_key", "token")
	model := firstNonEmpty(cfgMap, "model", "model_name")
	if baseURL == "" {
		return nil, errors.New("LLM 未配置 base_url/api_url")
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	endpoint := joinChatCompletionsURL(baseURL)

	systemPrompt := buildAdminStorySystemPrompt(req)
	userPrompt := buildAdminStoryUserPrompt(req, theme)

	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.8,
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用 LLM 失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LLM HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 400))
	}
	content, err := extractChatContent(respBody)
	if err != nil {
		return nil, err
	}
	title, fullText := stripLeadingMetaTitle(content)
	if title == "" {
		title = strings.TrimSpace(req.Title)
	}
	if title == "" {
		title = theme
		if !strings.Contains(title, "故事") {
			title = theme + "的故事"
		}
	}
	canonical := strings.TrimSpace(req.CanonicalKey)
	if canonical == "" && (req.PoolKind == PoolNamed || req.NarrationMode == "canonical") {
		canonical = theme
	}
	fullText = strings.TrimSpace(fullText)
	if fullText == "" {
		return nil, errors.New("LLM 未返回有效正文")
	}
	return &GenerateResult{
		Title:        title,
		ThemeKey:     theme,
		CanonicalKey: canonical,
		FullText:     fullText,
		Segments:     SegmentText(fullText),
		Model:        model,
	}, nil
}

func loadLLMConfig(db *gorm.DB, configID string) (*models.Config, map[string]any, error) {
	if db == nil {
		return nil, nil, errors.New("db unavailable")
	}
	var cfg models.Config
	configID = strings.TrimSpace(configID)
	var err error
	if configID != "" {
		err = db.Where("type = ? AND config_id = ? AND enabled = ?", "llm", configID, true).First(&cfg).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("LLM 配置 %s 不存在或未启用", configID)
		}
	} else {
		err = db.Where("type = ? AND enabled = ? AND is_default = ?", "llm", true, true).First(&cfg).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = db.Where("type = ? AND enabled = ?", "llm", true).Order("id ASC").First(&cfg).Error
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("未找到可用的 LLM 配置，请先在「LLM 配置」中设置")
		}
	}
	if err != nil {
		return nil, nil, err
	}
	cfgMap := map[string]any{}
	if strings.TrimSpace(cfg.JsonData) != "" {
		if err := json.Unmarshal([]byte(cfg.JsonData), &cfgMap); err != nil {
			return nil, nil, errors.New("LLM json_data 无效")
		}
	}
	return &cfg, cfgMap, nil
}

func buildAdminStorySystemPrompt(req GenerateRequest) string {
	narration := strings.TrimSpace(req.NarrationMode)
	if narration == "" {
		if req.PoolKind == PoolNamed {
			narration = "canonical"
		} else {
			narration = "creative"
		}
	}
	var b strings.Builder
	b.WriteString("你是儿童故事作家。请创作适合朗读的中文故事正文。\n")
	b.WriteString("输出格式：\n")
	b.WriteString("第一行：[[meta:title=故事名称|theme=主题]]\n")
	b.WriteString("第二行起为正文，不要 Markdown，不要前言后记。\n")
	if narration == "canonical" {
		b.WriteString("讲述方式：经典/神话正篇，忠于广为流传的情节，语言温和适合儿童。\n")
	} else {
		b.WriteString("讲述方式：原创温暖故事，积极正面。\n")
	}
	if age := strings.TrimSpace(req.AgeBand); age != "" {
		b.WriteString("年龄档：")
		b.WriteString(age)
		b.WriteString("\n")
	}
	b.WriteString("篇幅：约 600～1200 字。")
	return b.String()
}

func buildAdminStoryUserPrompt(req GenerateRequest, theme string) string {
	var b strings.Builder
	b.WriteString("请写一篇主题为「")
	b.WriteString(theme)
	b.WriteString("」的故事。")
	if extra := strings.TrimSpace(req.ExtraPrompt); extra != "" {
		b.WriteString("\n补充要求：")
		b.WriteString(extra)
	}
	return b.String()
}

func joinChatCompletionsURL(baseURL string) string {
	u := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(u, "/chat/completions") {
		return u
	}
	if strings.HasSuffix(u, "/v1") {
		return u + "/chat/completions"
	}
	return u + "/v1/chat/completions"
}

func extractChatContent(respBody []byte) (string, error) {
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("解析 LLM 响应失败: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", errors.New(parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", errors.New("LLM 返回空内容")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func stripLeadingMetaTitle(raw string) (title, body string) {
	raw = strings.TrimSpace(raw)
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return "", raw
	}
	first := strings.TrimSpace(lines[0])
	if strings.HasPrefix(first, "[[meta:") && strings.HasSuffix(first, "]]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(first, "[[meta:"), "]]")
		for _, part := range strings.Split(inner, "|") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 && strings.TrimSpace(kv[0]) == "title" {
				title = strings.TrimSpace(kv[1])
			}
		}
		body = strings.TrimSpace(strings.Join(lines[1:], "\n"))
		return title, body
	}
	return "", raw
}

func firstNonEmpty(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
