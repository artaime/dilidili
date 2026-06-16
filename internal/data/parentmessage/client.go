package parentmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dili-esp32-server-golang/internal/components/http"
)

type ClientConfig struct {
	BaseURL   string
	AuthToken string
	Timeout   time.Duration
	Enabled   bool
}

type PendingMessage struct {
	ID          uint       `json:"id"`
	UserID      uint       `json:"user_id"`
	DeviceID    uint       `json:"device_id"`
	Title       string     `json:"title"`
	TextContent string     `json:"text_content"`
	SourceType  string     `json:"source_type"`
	Status      string     `json:"status"`
	FamilyRole  string     `json:"family_role"`
	AudioURL    string     `json:"audio_url"`
	CreatedAt   time.Time  `json:"created_at"`
	PlayedAt    *time.Time `json:"played_at,omitempty"`
}

type Client struct {
	client  *http.ManagerClient
	enabled bool
	baseURL string
}

type SearchParams struct {
	Start string
	End   string
	Limit int
}

const defaultParentMessageSearchLimit = 100

func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Client{
		client: http.NewManagerClient(http.ManagerClientConfig{
			BaseURL:    cfg.BaseURL,
			AuthToken:  cfg.AuthToken,
			Timeout:    cfg.Timeout,
			MaxRetries: 2,
		}),
		enabled: cfg.Enabled,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
	}
}

func (c *Client) ListPendingMessages(ctx context.Context, deviceName string) ([]PendingMessage, error) {
	if c == nil || !c.enabled {
		return nil, nil
	}
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" {
		return nil, fmt.Errorf("device_name 不能为空")
	}
	body, err := c.client.DoRequestRaw(ctx, http.RequestOptions{
		Method: "GET",
		Path:   "/api/internal/devices/" + url.PathEscape(deviceName) + "/parent-messages/pending",
	})
	if err != nil {
		return nil, err
	}

	messages, err := parsePendingMessagesResponse(body)
	if err != nil {
		return nil, fmt.Errorf("解析留言响应失败: %w", err)
	}
	for i := range messages {
		messages[i].AudioURL = c.resolveAudioURL(messages[i].AudioURL)
	}
	return messages, nil
}

func (c *Client) ListPlayedMessages(ctx context.Context, deviceName string, limit int) ([]PendingMessage, error) {
	if c == nil || !c.enabled {
		return nil, nil
	}
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" {
		return nil, fmt.Errorf("device_name 不能为空")
	}
	path := "/api/internal/devices/" + url.PathEscape(deviceName) + "/parent-messages/played"
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}
	body, err := c.client.DoRequestRaw(ctx, http.RequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return nil, err
	}
	messages, err := parsePendingMessagesResponse(body)
	if err != nil {
		return nil, fmt.Errorf("解析已播留言响应失败: %w", err)
	}
	for i := range messages {
		messages[i].AudioURL = c.resolveAudioURL(messages[i].AudioURL)
	}
	return messages, nil
}

func (c *Client) SearchMessages(ctx context.Context, deviceName string, params SearchParams) ([]PendingMessage, error) {
	if c == nil || !c.enabled {
		return nil, nil
	}
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" {
		return nil, fmt.Errorf("device_name 不能为空")
	}
	path := "/api/internal/devices/" + url.PathEscape(deviceName) + "/parent-messages/search"
	query := url.Values{}
	if strings.TrimSpace(params.Start) != "" {
		query.Set("start", strings.TrimSpace(params.Start))
	}
	if strings.TrimSpace(params.End) != "" {
		query.Set("end", strings.TrimSpace(params.End))
	}
	limit := params.Limit
	if limit <= 0 {
		limit = defaultParentMessageSearchLimit
	}
	query.Set("limit", strconv.Itoa(limit))
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	body, err := c.client.DoRequestRaw(ctx, http.RequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return nil, err
	}
	messages, err := parsePendingMessagesResponse(body)
	if err != nil {
		return nil, fmt.Errorf("解析留言搜索响应失败: %w", err)
	}
	for i := range messages {
		messages[i].AudioURL = c.resolveAudioURL(messages[i].AudioURL)
	}
	return messages, nil
}

func (c *Client) GetMessage(ctx context.Context, messageID uint) (*PendingMessage, error) {
	if c == nil || !c.enabled {
		return nil, fmt.Errorf("parent message client disabled")
	}
	body, err := c.client.DoRequestRaw(ctx, http.RequestOptions{
		Method: "GET",
		Path:   fmt.Sprintf("/api/internal/parent-messages/%d", messageID),
	})
	if err != nil {
		return nil, err
	}
	messages, err := parsePendingMessagesResponse(body)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("留言不存在")
	}
	msg := messages[0]
	msg.AudioURL = c.resolveAudioURL(msg.AudioURL)
	return &msg, nil
}

func parsePendingMessagesResponse(body []byte) ([]PendingMessage, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("响应体不是合法 JSON: %w", err)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, nil
	}

	var messages []PendingMessage
	if err := json.Unmarshal(envelope.Data, &messages); err == nil {
		return messages, nil
	}

	var single PendingMessage
	if err := json.Unmarshal(envelope.Data, &single); err != nil {
		return nil, fmt.Errorf("data 字段无法解析为留言列表: %w", err)
	}
	if single.ID == 0 {
		return nil, nil
	}
	return []PendingMessage{single}, nil
}

func (c *Client) GetPendingMessage(ctx context.Context, deviceName string) (*PendingMessage, error) {
	messages, err := c.ListPendingMessages(ctx, deviceName)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	msg := messages[0]
	return &msg, nil
}

func (c *Client) UpdateStatus(ctx context.Context, messageID uint, status string) error {
	if c == nil || !c.enabled {
		return nil
	}
	return c.client.DoRequest(ctx, http.RequestOptions{
		Method: "PATCH",
		Path:   fmt.Sprintf("/api/internal/parent-messages/%d/status", messageID),
		Body: map[string]string{
			"status": status,
		},
	})
}

func (c *Client) DownloadMessageAudio(ctx context.Context, messageID uint) ([]byte, error) {
	if c == nil || !c.enabled {
		return nil, fmt.Errorf("parent message client disabled")
	}
	return c.client.DoRequestRaw(ctx, http.RequestOptions{
		Method: "GET",
		Path:   fmt.Sprintf("/api/internal/parent-messages/%d/audio", messageID),
	})
}

func (c *Client) ResolveAudioURL(relativePath string) string {
	return c.resolveAudioURL(relativePath)
}

func (c *Client) resolveAudioURL(relativePath string) string {
	relativePath = strings.TrimSpace(relativePath)
	if relativePath == "" {
		return ""
	}
	if strings.HasPrefix(relativePath, "http://") || strings.HasPrefix(relativePath, "https://") {
		return relativePath
	}
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}
	return c.baseURL + relativePath
}
