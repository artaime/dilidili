package device_memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"dili-esp32-server-golang/pkg/memobaseuserid"
	"dili/manager/backend/models"

	"github.com/memodb-io/memobase/src/client/memobase-go/core"
	"gorm.io/gorm"
)

var (
	ErrDeviceNotFound      = errors.New("设备不存在")
	ErrDeviceMissingSN     = errors.New("设备缺少 SN")
	ErrLongMemoryDisabled  = errors.New("设备未启用长记忆")
	ErrMemobaseNotConfigured = errors.New("未配置 Memobase 长期记忆")
)

type ProfileItem struct {
	Topic     string `json:"topic"`
	SubTopic  string `json:"sub_topic"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

type EventItem struct {
	EventTip  string   `json:"event_tip"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
}

type DeviceMemoryView struct {
	DeviceID       uint          `json:"device_id"`
	DeviceSN       string        `json:"device_sn"`
	AgentID        uint          `json:"agent_id"`
	MemoryMode     string        `json:"memory_mode"`
	Provider       string        `json:"provider"`
	MemobaseUserID string        `json:"memobase_user_id"`
	LegacyUserID   string        `json:"legacy_user_id,omitempty"`
	UsingLegacy    bool          `json:"using_legacy"`
	Profiles       []ProfileItem `json:"profiles"`
	Events         []EventItem   `json:"events"`
	Context        string        `json:"context"`
}

type memoryConfigPayload struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) GetDeviceMemory(ctx context.Context, deviceID uint) (*DeviceMemoryView, error) {
	device, agent, memCfg, err := s.loadDeviceMemoryContext(deviceID)
	if err != nil {
		return nil, err
	}

	payload, err := parseMemobaseConfig(memCfg.JsonData)
	if err != nil {
		return nil, err
	}

	client, err := core.NewMemoBaseClient(payload.BaseURL, payload.APIKey)
	if err != nil {
		return nil, fmt.Errorf("连接 Memobase 失败: %w", err)
	}

	primaryID := memobaseuserid.MemobaseUserID(device.DeviceName)
	legacyID := memobaseuserid.LegacyMemobaseUserID(device.DeviceName)

	view := &DeviceMemoryView{
		DeviceID:       device.ID,
		DeviceSN:       device.DeviceName,
		AgentID:        device.AgentID,
		MemoryMode:     agent.MemoryMode,
		Provider:       memCfg.Provider,
		MemobaseUserID: primaryID,
		LegacyUserID:   legacyID,
	}

	activeID, usingLegacy, err := s.fetchIntoView(client, view, primaryID, legacyID)
	if err != nil {
		return nil, err
	}
	view.MemobaseUserID = activeID
	view.UsingLegacy = usingLegacy
	return view, nil
}

func (s *Service) DeleteDeviceMemory(ctx context.Context, deviceID uint) error {
	device, _, memCfg, err := s.loadDeviceMemoryContext(deviceID)
	if err != nil {
		return err
	}

	payload, err := parseMemobaseConfig(memCfg.JsonData)
	if err != nil {
		return err
	}

	client, err := core.NewMemoBaseClient(payload.BaseURL, payload.APIKey)
	if err != nil {
		return fmt.Errorf("连接 Memobase 失败: %w", err)
	}

	primaryID := memobaseuserid.MemobaseUserID(device.DeviceName)
	legacyID := memobaseuserid.LegacyMemobaseUserID(device.DeviceName)

	var deleteErr error
	for _, id := range []string{primaryID, legacyID} {
		if err := client.DeleteUser(id); err != nil {
			deleteErr = errors.Join(deleteErr, fmt.Errorf("删除用户 %s 失败: %w", id, err))
		}
	}
	return deleteErr
}

func (s *Service) loadDeviceMemoryContext(deviceID uint) (*models.Device, *models.Agent, *models.Config, error) {
	var device models.Device
	if err := s.DB.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrDeviceNotFound
		}
		return nil, nil, nil, err
	}
	if strings.TrimSpace(device.DeviceName) == "" {
		return nil, nil, nil, ErrDeviceMissingSN
	}
	if device.AgentID == 0 {
		return nil, nil, nil, ErrLongMemoryDisabled
	}

	var agent models.Agent
	if err := s.DB.First(&agent, device.AgentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrLongMemoryDisabled
		}
		return nil, nil, nil, err
	}
	if strings.TrimSpace(normalizeMemoryMode(agent.MemoryMode)) != "long" {
		return nil, nil, nil, fmt.Errorf("%w（当前: %s）", ErrLongMemoryDisabled, agent.MemoryMode)
	}

	var memCfg models.Config
	if err := s.DB.Where("type = ? AND is_default = ? AND enabled = ?", "memory", true, true).
		First(&memCfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrMemobaseNotConfigured
		}
		return nil, nil, nil, err
	}
	if strings.TrimSpace(memCfg.Provider) != "memobase" {
		return nil, nil, nil, fmt.Errorf("%w（当前 provider: %s）", ErrMemobaseNotConfigured, memCfg.Provider)
	}

	return &device, &agent, &memCfg, nil
}

func parseMemobaseConfig(jsonData string) (*memoryConfigPayload, error) {
	var payload memoryConfigPayload
	if strings.TrimSpace(jsonData) != "" {
		if err := json.Unmarshal([]byte(jsonData), &payload); err != nil {
			return nil, fmt.Errorf("解析 Memory 配置失败: %w", err)
		}
	}
	if strings.TrimSpace(payload.BaseURL) == "" || strings.TrimSpace(payload.APIKey) == "" {
		return nil, fmt.Errorf("%w: base_url 或 api_key 为空", ErrMemobaseNotConfigured)
	}
	return &payload, nil
}

func (s *Service) fetchIntoView(client *core.MemoBaseClient, view *DeviceMemoryView, primaryID, legacyID string) (string, bool, error) {
	profiles, events, contextText, err := fetchMemobaseUserData(client, primaryID)
	if err != nil {
		return "", false, err
	}
	if hasMemoryData(profiles, events, contextText) {
		view.Profiles = profiles
		view.Events = events
		view.Context = contextText
		return primaryID, false, nil
	}

	legacyProfiles, legacyEvents, legacyContext, err := fetchMemobaseUserData(client, legacyID)
	if err != nil {
		return "", false, err
	}
	if hasMemoryData(legacyProfiles, legacyEvents, legacyContext) {
		view.Profiles = legacyProfiles
		view.Events = legacyEvents
		view.Context = legacyContext
		return legacyID, true, nil
	}

	view.Profiles = []ProfileItem{}
	view.Events = []EventItem{}
	view.Context = ""
	return primaryID, false, nil
}

func fetchMemobaseUserData(client *core.MemoBaseClient, userID string) ([]ProfileItem, []EventItem, string, error) {
	user := &core.User{UserID: userID, ProjectClient: client}

	profilesRaw, err := user.Profile(&core.ProfileOptions{MaxTokenSize: 4000})
	if err != nil {
		return nil, nil, "", fmt.Errorf("读取 Memobase Profile 失败: %w", err)
	}

	eventsRaw, err := user.Event(30, nil, false)
	if err != nil {
		return nil, nil, "", fmt.Errorf("读取 Memobase Event 失败: %w", err)
	}

	contextText, err := user.Context(&core.ContextOptions{MaxTokenSize: 2000})
	if err != nil {
		return nil, nil, "", fmt.Errorf("读取 Memobase Context 失败: %w", err)
	}

	profiles := make([]ProfileItem, 0, len(profilesRaw))
	for _, p := range profilesRaw {
		profiles = append(profiles, ProfileItem{
			Topic:     p.Attributes.Topic,
			SubTopic:  p.Attributes.SubTopic,
			Content:   p.Content,
			UpdatedAt: p.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	events := make([]EventItem, 0, len(eventsRaw))
	for _, e := range eventsRaw {
		tags := make([]string, 0, len(e.EventData.EventTags))
		for _, tag := range e.EventData.EventTags {
			if tag.Value != "" {
				tags = append(tags, fmt.Sprintf("%s:%s", tag.Tag, tag.Value))
			} else {
				tags = append(tags, tag.Tag)
			}
		}
		events = append(events, EventItem{
			EventTip:  e.EventData.EventTip,
			Tags:      tags,
			CreatedAt: e.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return profiles, events, contextText, nil
}

func hasMemoryData(profiles []ProfileItem, events []EventItem, context string) bool {
	return len(profiles) > 0 || len(events) > 0 || strings.TrimSpace(context) != ""
}

func normalizeMemoryMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "none", "short", "long":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "short"
	}
}
