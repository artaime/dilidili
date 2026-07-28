package controllers

import (
	"fmt"
	"testing"
	"time"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupConversationRecordTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:conversation_record_test?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.ChatMessage{}, &models.ParentMessage{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestListConversationRecordsMergeAndOrder(t *testing.T) {
	db := setupConversationRecordTestDB(t)
	user := createServiceTestUser(t, db, "conv-user", "user")
	device := models.Device{UserID: user.ID, DeviceName: "SN-CONV-001", DeviceCode: "666666", Activated: true, AgentID: 1}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	base := time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)
	chat1 := models.ChatMessage{
		MessageID: "m1", DeviceID: device.DeviceName, AgentID: "1", UserID: user.ID,
		Role: "user", Content: "你好", CreatedAt: base,
	}
	chat2 := models.ChatMessage{
		MessageID: "m2", DeviceID: device.DeviceName, AgentID: "1", UserID: user.ID,
		Role: "assistant", Content: "你好呀", CreatedAt: base.Add(2 * time.Minute),
	}
	if err := db.Create(&chat1).Error; err != nil {
		t.Fatalf("create chat1: %v", err)
	}
	if err := db.Create(&chat2).Error; err != nil {
		t.Fatalf("create chat2: %v", err)
	}

	playedAt := base.Add(1 * time.Minute)
	parent := models.ParentMessage{
		UserID: user.ID, DeviceID: device.ID, TextContent: "宝贝加油",
		SourceType: "text", Status: "played", CreatedAt: base.Add(-time.Hour), PlayedAt: &playedAt,
	}
	pending := models.ParentMessage{
		UserID: user.ID, DeviceID: device.ID, TextContent: "未播留言",
		SourceType: "text", Status: "pending", CreatedAt: base.Add(3 * time.Minute),
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if err := db.Create(&pending).Error; err != nil {
		t.Fatalf("create pending: %v", err)
	}

	items, hasOlder, hasNewer, err := listConversationRecords(db, nil, device.DeviceName, device.ID, user.ID, conversationRecordQuery{Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 records, got %d: %+v", len(items), items)
	}
	if items[0].Type != "chat" || items[0].Content != "你好" {
		t.Fatalf("first item = %+v", items[0])
	}
	if items[1].Type != "parent_message" || items[1].Content != "宝贝加油" {
		t.Fatalf("second item = %+v", items[1])
	}
	if items[2].Type != "chat" || items[2].Content != "你好呀" {
		t.Fatalf("third item = %+v", items[2])
	}
	if hasNewer {
		t.Fatalf("expected has_newer=false")
	}
	if hasOlder {
		t.Fatalf("expected has_older=false")
	}
}

func TestListConversationRecordsPaginationBefore(t *testing.T) {
	db := setupConversationRecordTestDB(t)
	user := createServiceTestUser(t, db, "conv-page", "user")
	device := models.Device{UserID: user.ID, DeviceName: "SN-CONV-002", DeviceCode: "777777", Activated: true, AgentID: 1}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	base := time.Date(2026, 6, 16, 9, 0, 0, 0, time.Local)
	for i := 0; i < 5; i++ {
		msg := models.ChatMessage{
			MessageID: fmt.Sprintf("p%d", i), DeviceID: device.DeviceName, AgentID: "1", UserID: user.ID,
			Role: "user", Content: "msg", CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if err := db.Create(&msg).Error; err != nil {
			t.Fatalf("create chat: %v", err)
		}
	}

	firstPage, hasOlder, _, err := listConversationRecords(db, nil, device.DeviceName, device.ID, user.ID, conversationRecordQuery{Limit: 2})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(firstPage) != 2 || !hasOlder {
		t.Fatalf("first page = %+v hasOlder=%v", firstPage, hasOlder)
	}

	last := firstPage[0]
	olderPage, hasOlder2, hasNewer2, err := listConversationRecords(db, nil, device.DeviceName, device.ID, user.ID, conversationRecordQuery{
		Limit: 2,
		Before: &conversationRecordCursor{
			SortTime: last.SortTime, Type: last.Type, ID: last.ID,
		},
	})
	if err != nil {
		t.Fatalf("older page: %v", err)
	}
	if len(olderPage) != 2 {
		t.Fatalf("older page len=%d", len(olderPage))
	}
	if !hasNewer2 {
		t.Fatalf("expected has_newer on older page")
	}
	if !hasOlder2 {
		t.Fatalf("expected more older records")
	}
}

func TestListConversationRecordsDateFilter(t *testing.T) {
	db := setupConversationRecordTestDB(t)
	user := createServiceTestUser(t, db, "conv-date", "user")
	device := models.Device{UserID: user.ID, DeviceName: "SN-CONV-003", DeviceCode: "888888", Activated: true, AgentID: 1}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	day := time.Date(2026, 6, 17, 0, 0, 0, 0, time.Local)
	prev := models.ChatMessage{
		MessageID: "prev", DeviceID: device.DeviceName, AgentID: "1", UserID: user.ID,
		Role: "user", Content: "昨天", CreatedAt: day.Add(-2 * time.Hour),
	}
	inDay := models.ChatMessage{
		MessageID: "today", DeviceID: device.DeviceName, AgentID: "1", UserID: user.ID,
		Role: "user", Content: "今天", CreatedAt: day.Add(10 * time.Hour),
	}
	if err := db.Create(&prev).Error; err != nil {
		t.Fatalf("create prev: %v", err)
	}
	if err := db.Create(&inDay).Error; err != nil {
		t.Fatalf("create inDay: %v", err)
	}

	items, hasOlder, _, err := listConversationRecords(db, nil, device.DeviceName, device.ID, user.ID, conversationRecordQuery{
		Limit: 20,
		Date:  &day,
	})
	if err != nil {
		t.Fatalf("list by date: %v", err)
	}
	if len(items) != 1 || items[0].Content != "今天" {
		t.Fatalf("date filter items=%+v", items)
	}
	if !hasOlder {
		t.Fatalf("expected has_older before selected day")
	}
}

func TestDedupeParentMessageTTSItems(t *testing.T) {
	db := setupConversationRecordTestDB(t)
	user := createServiceTestUser(t, db, "conv-dedupe", "user")
	device := models.Device{UserID: user.ID, DeviceName: "SN-CONV-DEDUPE", DeviceCode: "999999", Activated: true, AgentID: 1}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	base := time.Date(2026, 6, 17, 15, 31, 0, 0, time.Local)
	transition := models.ChatMessage{
		MessageID: "t1", DeviceID: device.DeviceName, AgentID: "1", UserID: user.ID,
		Role: "assistant", Content: "好的，接下来将播放爸爸今天傍晚15点31分的留言。", CreatedAt: base,
		AudioPath: "tts/transition.wav",
	}
	dupTTS := models.ChatMessage{
		MessageID: "d1", DeviceID: device.DeviceName, AgentID: "1", UserID: user.ID,
		Role: "assistant", Content: "马上端午节了，端午安康", CreatedAt: base.Add(30 * time.Second),
		AudioPath: "tts/content.wav",
	}
	if err := db.Create(&transition).Error; err != nil {
		t.Fatalf("create transition: %v", err)
	}
	if err := db.Create(&dupTTS).Error; err != nil {
		t.Fatalf("create dup tts: %v", err)
	}

	playedAt := base.Add(time.Minute)
	parent := models.ParentMessage{
		UserID: user.ID, DeviceID: device.ID, TextContent: "马上端午节了，端午安康",
		SourceType: "text", Status: "played", CreatedAt: base.Add(-time.Hour), PlayedAt: &playedAt,
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}

	items, _, _, err := listConversationRecords(db, nil, device.DeviceName, device.ID, user.ID, conversationRecordQuery{Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 records after dedupe, got %d: %+v", len(items), items)
	}
	if items[0].Type != "chat" || items[0].Content != transition.Content {
		t.Fatalf("first item = %+v", items[0])
	}
	if items[1].Type != "parent_message" {
		t.Fatalf("second item type = %s", items[1].Type)
	}
	if !items[1].HasAudio || items[1].ChatAudioID != dupTTS.ID {
		t.Fatalf("parent item audio = %+v", items[1])
	}
}
