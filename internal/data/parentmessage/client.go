package parentmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"xiaozhi-esp32-server-golang/internal/components/http"
)

type ClientConfig struct {
	BaseURL   string
	AuthToken string
	Timeout   time.Duration
	Enabled   bool
}

type PendingMessage struct {
	ID          uint      `json:"id"`
	UserID      uint      `json:"user_id"`
	DeviceID    uint      `json:"device_id"`
	Title       string    `json:"title"`
	TextContent string    `json:"text_content"`
	SourceType  string    `json:"source_type"`
	Status      string    `json:"status"`
	FamilyRole  string    `json:"family_role"`
	AudioURL    string    `json:"audio_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type Client struct {
	client  *http.ManagerClient
	enabled bool
	baseURL string
}

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
