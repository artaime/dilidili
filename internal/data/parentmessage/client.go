package parentmessage

import (
	"context"
	"encoding/json"
	"fmt"
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
	ID          uint   `json:"id"`
	UserID      uint   `json:"user_id"`
	DeviceID    uint   `json:"device_id"`
	TextContent string `json:"text_content"`
	SourceType  string `json:"source_type"`
	Status      string `json:"status"`
}

type Client struct {
	client  *http.ManagerClient
	enabled bool
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
	}
}

func (c *Client) GetPendingMessage(ctx context.Context, deviceName string) (*PendingMessage, error) {
	if c == nil || !c.enabled {
		return nil, nil
	}
	body, err := c.client.DoRequestRaw(ctx, http.RequestOptions{
		Method: "GET",
		Path:   "/api/internal/devices/" + deviceName + "/parent-messages/pending",
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data *PendingMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析留言响应失败: %w", err)
	}
	return resp.Data, nil
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
