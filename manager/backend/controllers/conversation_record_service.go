package controllers

import (
	"fmt"
	"strings"
	"time"

	"dili/manager/backend/models"

	"gorm.io/gorm"
)

const defaultConversationRecordLimit = 20
const maxConversationRecordLimit = 100

// ConversationRecordItem 统一对话记录项
type ConversationRecordItem struct {
	Type        string     `json:"type"`
	ID          uint       `json:"id"`
	SortTime    time.Time  `json:"sort_time"`
	Role        string     `json:"role,omitempty"`
	Content     string     `json:"content"`
	HasAudio    bool       `json:"has_audio"`
	ChatAudioID uint       `json:"chat_audio_id,omitempty"` // 文字留言 TTS 回放对应的 chat_messages.id
	SourceType  string     `json:"source_type,omitempty"`
	Title       string     `json:"title,omitempty"`
	PlayedAt    *time.Time `json:"played_at,omitempty"`
}

type conversationRecordCursor struct {
	SortTime time.Time
	Type     string
	ID       uint
}

type conversationRecordQuery struct {
	Limit  int
	Before *conversationRecordCursor
	After  *conversationRecordCursor
	Date   *time.Time
}

type unifiedRecordRow struct {
	RecordType string     `gorm:"column:record_type"`
	ID         uint       `gorm:"column:id"`
	SortTime   time.Time  `gorm:"column:sort_time"`
	Role       string     `gorm:"column:role"`
	Content    string     `gorm:"column:content"`
	HasAudio   int        `gorm:"column:has_audio"`
	SourceType string     `gorm:"column:source_type"`
	Title      string     `gorm:"column:title"`
	PlayedAt   *time.Time `gorm:"column:played_at"`
}

func normalizeConversationLimit(limit int) int {
	if limit <= 0 {
		return defaultConversationRecordLimit
	}
	if limit > maxConversationRecordLimit {
		return maxConversationRecordLimit
	}
	return limit
}

func unifiedConversationSQL() string {
	return `
SELECT 'chat' AS record_type, id, created_at AS sort_time, role, content,
       CASE WHEN COALESCE(audio_path, '') != '' THEN 1 ELSE 0 END AS has_audio,
       '' AS source_type, '' AS title, NULL AS played_at
FROM chat_messages
WHERE device_id = ? AND user_id = ? AND is_deleted = 0 AND role IN ('user','assistant')
UNION ALL
SELECT 'parent_message', id, played_at, 'parent', COALESCE(text_content, ''),
       CASE WHEN source_type = 'voice' AND COALESCE(audio_path, '') != '' THEN 1 ELSE 0 END,
       source_type, COALESCE(title, ''), played_at
FROM parent_messages
WHERE device_id = ? AND user_id = ? AND status = 'played' AND played_at IS NOT NULL`
}

func rowToItem(row unifiedRecordRow) ConversationRecordItem {
	item := ConversationRecordItem{
		Type:       row.RecordType,
		ID:         row.ID,
		SortTime:   row.SortTime,
		Content:    row.Content,
		HasAudio:   row.HasAudio == 1,
		SourceType: row.SourceType,
		Title:      row.Title,
		PlayedAt:   row.PlayedAt,
	}
	if row.RecordType == "chat" {
		item.Role = row.Role
	}
	return item
}

func listConversationRecords(db *gorm.DB, deviceSN string, deviceDBID uint, userID uint, query conversationRecordQuery) ([]ConversationRecordItem, bool, bool, error) {
	limit := normalizeConversationLimit(query.Limit)
	fetchLimit := limit + 1

	baseSQL := fmt.Sprintf("SELECT * FROM (%s) AS unified", unifiedConversationSQL())
	args := []interface{}{deviceSN, userID, deviceDBID, userID}

	var (
		rows       []unifiedRecordRow
		hasOlder   bool
		hasNewer   bool
		extraWhere strings.Builder
		orderSQL   string
	)

	switch {
	case query.Date != nil:
		dayStart := time.Date(query.Date.Year(), query.Date.Month(), query.Date.Day(), 0, 0, 0, 0, query.Date.Location())
		dayEnd := dayStart.Add(24 * time.Hour)
		extraWhere.WriteString(" WHERE sort_time >= ? AND sort_time < ?")
		args = append(args, dayStart, dayEnd)
		orderSQL = " ORDER BY sort_time ASC, record_type ASC, id ASC"
	case query.After != nil:
		extraWhere.WriteString(` WHERE (
			sort_time > ? OR
			(sort_time = ? AND record_type > ?) OR
			(sort_time = ? AND record_type = ? AND id > ?)
		)`)
		c := query.After
		args = append(args, c.SortTime, c.SortTime, c.Type, c.SortTime, c.Type, c.ID)
		orderSQL = " ORDER BY sort_time ASC, record_type ASC, id ASC"
	case query.Before != nil:
		extraWhere.WriteString(` WHERE (
			sort_time < ? OR
			(sort_time = ? AND record_type < ?) OR
			(sort_time = ? AND record_type = ? AND id < ?)
		)`)
		c := query.Before
		args = append(args, c.SortTime, c.SortTime, c.Type, c.SortTime, c.Type, c.ID)
		orderSQL = " ORDER BY sort_time DESC, record_type DESC, id DESC"
	default:
		orderSQL = " ORDER BY sort_time DESC, record_type DESC, id DESC"
	}

	finalSQL := baseSQL + extraWhere.String() + orderSQL + " LIMIT ?"
	args = append(args, fetchLimit)

	if err := db.Raw(finalSQL, args...).Scan(&rows).Error; err != nil {
		return nil, false, false, err
	}

	if len(rows) > limit {
		switch {
		case query.Date != nil:
			hasNewer = true
		case query.After != nil:
			hasNewer = true
		case query.Before != nil:
			hasOlder = true
		default:
			hasOlder = true
		}
		rows = rows[:limit]
	}

	// 默认与 before 查询为 DESC，需反转为时间正序
	if query.Date == nil && query.After == nil {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}

	items := make([]ConversationRecordItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, rowToItem(row))
	}
	items = dedupeParentMessageTTSItems(items)

	if len(items) > 0 {
		first := items[0]
		last := items[len(items)-1]
		switch {
		case query.Date != nil:
			hasOlder = hasRecordsBefore(db, deviceSN, deviceDBID, userID, first)
		case query.After != nil:
			hasOlder = hasRecordsBefore(db, deviceSN, deviceDBID, userID, first)
		case query.Before != nil:
			hasNewer = hasRecordsAfter(db, deviceSN, deviceDBID, userID, unifiedRecordRow{
				RecordType: last.Type, ID: last.ID, SortTime: last.SortTime,
			})
		}
	}

	return items, hasOlder, hasNewer, nil
}

func hasRecordsBefore(db *gorm.DB, deviceSN string, deviceDBID uint, userID uint, first ConversationRecordItem) bool {
	cursor := conversationRecordCursor{SortTime: first.SortTime, Type: first.Type, ID: first.ID}
	var rows []unifiedRecordRow
	sql := fmt.Sprintf("SELECT * FROM (%s) AS unified WHERE sort_time < ? OR (sort_time = ? AND record_type < ?) OR (sort_time = ? AND record_type = ? AND id < ?) LIMIT 1",
		unifiedConversationSQL())
	args := []interface{}{deviceSN, userID, deviceDBID, userID, cursor.SortTime, cursor.SortTime, cursor.Type, cursor.SortTime, cursor.Type, cursor.ID}
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return false
	}
	return len(rows) > 0
}

func hasRecordsAfter(db *gorm.DB, deviceSN string, deviceDBID uint, userID uint, last unifiedRecordRow) bool {
	cursor := conversationRecordCursor{SortTime: last.SortTime, Type: last.RecordType, ID: last.ID}
	var rows []unifiedRecordRow
	sql := fmt.Sprintf(`SELECT * FROM (%s) AS unified WHERE sort_time > ? OR (sort_time = ? AND record_type > ?) OR (sort_time = ? AND record_type = ? AND id > ?) LIMIT 1`,
		unifiedConversationSQL())
	args := []interface{}{deviceSN, userID, deviceDBID, userID, cursor.SortTime, cursor.SortTime, cursor.Type, cursor.SortTime, cursor.Type, cursor.ID}
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return false
	}
	return len(rows) > 0
}

func parseConversationCursor(sortTimeRaw, recordType string, idRaw uint) (*conversationRecordCursor, error) {
	sortTimeRaw = strings.TrimSpace(sortTimeRaw)
	recordType = strings.TrimSpace(recordType)
	if sortTimeRaw == "" || recordType == "" || idRaw == 0 {
		return nil, fmt.Errorf("游标参数不完整")
	}
	sortTime, err := time.Parse(time.RFC3339, sortTimeRaw)
	if err != nil {
		sortTime, err = time.Parse("2006-01-02T15:04:05Z07:00", sortTimeRaw)
		if err != nil {
			return nil, fmt.Errorf("sort_time 格式无效")
		}
	}
	return &conversationRecordCursor{SortTime: sortTime, Type: recordType, ID: idRaw}, nil
}

func loadDeviceForConversation(db *gorm.DB, deviceID uint, userID *uint) (*models.Device, error) {
	var device models.Device
	query := db.Where("id = ?", deviceID)
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if err := query.First(&device).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

type parentTextRef struct {
	id       uint
	content  string
	playedAt time.Time
}

// dedupeParentMessageTTSItems 隐藏与家长文字留言正文重复的 assistant 聊天记录，
// 并将对应 TTS 音频关联到家长留言项供回放。
func dedupeParentMessageTTSItems(items []ConversationRecordItem) []ConversationRecordItem {
	parentsByContent := make(map[string][]parentTextRef)
	for _, item := range items {
		if item.Type != "parent_message" || item.SourceType != "text" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		playedAt := item.SortTime
		if item.PlayedAt != nil {
			playedAt = *item.PlayedAt
		}
		parentsByContent[content] = append(parentsByContent[content], parentTextRef{
			id: item.ID, content: content, playedAt: playedAt,
		})
	}
	if len(parentsByContent) == 0 {
		return items
	}

	skipChat := make(map[uint]bool)
	audioForParent := make(map[uint]uint)
	bestAudioDelta := make(map[uint]time.Duration)

	for _, item := range items {
		if item.Type != "chat" || item.Role != "assistant" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		candidates, ok := parentsByContent[content]
		if !ok {
			continue
		}
		var best *parentTextRef
		var bestDelta time.Duration
		for i := range candidates {
			p := &candidates[i]
			delta := item.SortTime.Sub(p.playedAt)
			if delta < 0 {
				delta = -delta
			}
			if best == nil || delta < bestDelta {
				best = p
				bestDelta = delta
			}
		}
		if best == nil {
			continue
		}
		skipChat[item.ID] = true
		if !item.HasAudio {
			continue
		}
		if prev, ok := bestAudioDelta[best.id]; !ok || bestDelta < prev {
			audioForParent[best.id] = item.ID
			bestAudioDelta[best.id] = bestDelta
		}
	}

	out := make([]ConversationRecordItem, 0, len(items))
	for _, item := range items {
		if item.Type == "chat" && skipChat[item.ID] {
			continue
		}
		if item.Type == "parent_message" && item.SourceType == "text" {
			if chatID, ok := audioForParent[item.ID]; ok {
				item.HasAudio = true
				item.ChatAudioID = chatID
			}
		}
		out = append(out, item)
	}
	return out
}
