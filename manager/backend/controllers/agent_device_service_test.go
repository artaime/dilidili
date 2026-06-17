package controllers

import (
	"strings"
	"testing"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAgentDeviceServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Config{},
		&models.Agent{},
		&models.Device{},
		&models.KnowledgeBase{},
		&models.AgentKnowledgeBase{},
		&models.Role{},
		&models.MCPMarketService{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func createServiceTestUser(t *testing.T, db *gorm.DB, username, role string) models.User {
	t.Helper()
	user := models.User{
		Username: username,
		Email:    username + "@example.test",
		Password: "secret",
		Role:     role,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return user
}

func createServiceTestConfig(t *testing.T, db *gorm.DB, typ, id, provider string) {
	t.Helper()
	if err := db.Create(&models.Config{
		Type:      typ,
		ConfigID:  id,
		Name:      id,
		Provider:  provider,
		Enabled:   true,
		IsDefault: true,
	}).Error; err != nil {
		t.Fatalf("create config %s/%s: %v", typ, id, err)
	}
}

func createServiceTestKnowledgeBase(t *testing.T, db *gorm.DB, userID uint, name string) models.KnowledgeBase {
	t.Helper()
	kb := models.KnowledgeBase{UserID: userID, Name: name, Content: "content", Status: "active"}
	if err := db.Create(&kb).Error; err != nil {
		t.Fatalf("create knowledge base %s: %v", name, err)
	}
	return kb
}

func strPtr(value string) *string {
	return &value
}

func TestAgentServicePermissionVoiceAndKnowledgeLinks(t *testing.T) {
	db := setupAgentDeviceServiceTestDB(t)
	userA := createServiceTestUser(t, db, "user-a", "user")
	userB := createServiceTestUser(t, db, "user-b", "user")
	createServiceTestConfig(t, db, "llm", "llm-default", "openai")
	createServiceTestConfig(t, db, "tts", "tts-default", "doubao")
	kbA := createServiceTestKnowledgeBase(t, db, userA.ID, "kb-a")
	kbB := createServiceTestKnowledgeBase(t, db, userB.ID, "kb-b")

	agentSvc := NewAgentService(db)
	kbIDs := []uint{kbA.ID}
	agent, err := agentSvc.Create(accessScope{ActorUserID: userA.ID}, AgentPayload{
		UserID:           userB.ID,
		Name:             "agent-a",
		Nickname:         strPtr("assistant-a"),
		CustomPrompt:     "prompt",
		LLMConfigID:      strPtr("llm-default"),
		TTSConfigID:      strPtr("tts-default"),
		Voice:            strPtr("exact-voice"),
		KnowledgeBaseIDs: &kbIDs,
	})
	if err != nil {
		t.Fatalf("create user agent: %v", err)
	}
	if agent.UserID != userA.ID {
		t.Fatalf("agent user_id = %d, want %d", agent.UserID, userA.ID)
	}
	if agent.Voice == nil || *agent.Voice != "exact-voice" {
		t.Fatalf("agent voice = %#v, want exact-voice", agent.Voice)
	}
	if len(agent.KnowledgeBaseIDs) != 1 || agent.KnowledgeBaseIDs[0] != kbA.ID {
		t.Fatalf("knowledge links = %#v, want [%d]", agent.KnowledgeBaseIDs, kbA.ID)
	}

	crossUserKBIDs := []uint{kbB.ID}
	if _, err := agentSvc.Update(accessScope{ActorUserID: userA.ID}, agent.ID, AgentPayload{
		Name:             "agent-a",
		Nickname:         strPtr("assistant-a"),
		KnowledgeBaseIDs: &crossUserKBIDs,
	}); err == nil || !strings.Contains(err.Error(), "知识库") {
		t.Fatalf("cross-user knowledge update error = %v, want knowledge ownership rejection", err)
	}

	if _, err := agentSvc.Get(accessScope{ActorUserID: userB.ID}, agent.ID); err == nil {
		t.Fatalf("other normal user should not read agent")
	}
	if _, err := agentSvc.Get(accessScope{ActorUserID: userB.ID, IsAdmin: true}, agent.ID); err != nil {
		t.Fatalf("admin should read agent: %v", err)
	}

	if _, err := agentSvc.Create(accessScope{ActorUserID: userB.ID, IsAdmin: true}, AgentPayload{
		UserID:           userA.ID,
		Name:             "admin-cross-kb",
		Nickname:         strPtr("admin-cross-kb"),
		KnowledgeBaseIDs: &crossUserKBIDs,
	}); err == nil || !strings.Contains(err.Error(), "知识库") {
		t.Fatalf("admin cross-user knowledge create error = %v, want rejection", err)
	}
}

func TestDeviceServiceBindingEnrichmentAndCrossUserRejection(t *testing.T) {
	db := setupAgentDeviceServiceTestDB(t)
	userA := createServiceTestUser(t, db, "device-user-a", "user")
	userB := createServiceTestUser(t, db, "device-user-b", "user")

	agentA := models.Agent{UserID: userA.ID, Name: "agent-a", Nickname: "agent-a"}
	agentB := models.Agent{UserID: userB.ID, Name: "agent-b", Nickname: "agent-b"}
	if err := db.Create(&agentA).Error; err != nil {
		t.Fatalf("create agent a: %v", err)
	}
	if err := db.Create(&agentB).Error; err != nil {
		t.Fatalf("create agent b: %v", err)
	}

	unbound := models.Device{DeviceCode: "123456", DeviceName: "dev-a", NickName: "dev-a", AgentID: agentA.ID}
	if err := db.Create(&unbound).Error; err != nil {
		t.Fatalf("create unbound device: %v", err)
	}
	ownedByB := models.Device{UserID: userB.ID, AgentID: agentB.ID, DeviceCode: "654321", DeviceName: "dev-b", NickName: "dev-b", Activated: true}
	if err := db.Create(&ownedByB).Error; err != nil {
		t.Fatalf("create owned device: %v", err)
	}

	deviceSvc := NewDeviceService(db)
	bound, err := deviceSvc.BindToAgent(accessScope{ActorUserID: userA.ID}, agentA.ID, DevicePayload{
		Code:     "123456",
		NickName: "living room",
	})
	if err != nil {
		t.Fatalf("bind unbound device: %v", err)
	}
	if bound.UserID != userA.ID || bound.AgentID != agentA.ID || !bound.Activated {
		t.Fatalf("bound device = user:%d agent:%d activated:%v, want user:%d agent:%d active", bound.UserID, bound.AgentID, bound.Activated, userA.ID, agentA.ID)
	}
	if bound.AgentName != "agent-a" {
		t.Fatalf("bound agent name = %q, want agent-a", bound.AgentName)
	}

	if _, err := deviceSvc.BindToAgent(accessScope{ActorUserID: userA.ID}, agentA.ID, DevicePayload{Code: "654321"}); err == nil {
		t.Fatalf("binding device owned by another user should fail")
	}

	noAgent := models.Device{DeviceCode: "999999", DeviceName: "dev-no-agent", NickName: "dev-no-agent"}
	if err := db.Create(&noAgent).Error; err != nil {
		t.Fatalf("create no-agent device: %v", err)
	}
	if _, err := deviceSvc.Bind(accessScope{ActorUserID: userA.ID}, DevicePayload{Code: "999999"}); err == nil || !strings.Contains(err.Error(), "智能体") {
		t.Fatalf("bind device without agent error = %v, want agent rejection", err)
	}

	if _, err := deviceSvc.Bind(accessScope{ActorUserID: userA.ID}, DevicePayload{Code: "000000"}); err == nil || !strings.Contains(err.Error(), "未登记") {
		t.Fatalf("bind unregistered device error = %v, want not registered", err)
	}

	if _, err := deviceSvc.Update(accessScope{ActorUserID: userA.ID, IsAdmin: true}, bound.ID, DevicePayload{
		UserID:   userB.ID,
		NickName: "cross",
		AgentID:  agentA.ID,
	}); err == nil || !strings.Contains(err.Error(), "智能体") {
		t.Fatalf("admin cross-user device-agent update error = %v, want rejection", err)
	}

	updated, err := deviceSvc.Update(accessScope{ActorUserID: userA.ID, IsAdmin: true}, bound.ID, DevicePayload{
		UserID:     userB.ID,
		NickName:   "moved",
		DeviceCode: "123456",
		DeviceName: "dev-a",
		AgentID:    agentB.ID,
	})
	if err != nil {
		t.Fatalf("admin same-user device-agent update: %v", err)
	}
	if updated.UserID != userB.ID || updated.AgentID != agentB.ID || updated.AgentName != "agent-b" || updated.Username != userB.Username {
		t.Fatalf("updated device enrichment = user:%d agent:%d agentName:%q username:%q", updated.UserID, updated.AgentID, updated.AgentName, updated.Username)
	}
}

func TestDeviceServiceAdminCreateWithoutUserRequiresDeviceName(t *testing.T) {
	db := setupAgentDeviceServiceTestDB(t)
	admin := createServiceTestUser(t, db, "device-admin", "admin")
	agent := models.Agent{UserID: admin.ID, Name: "admin-agent", Nickname: "admin-agent"}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create admin agent: %v", err)
	}
	deviceSvc := NewDeviceService(db)
	scope := accessScope{ActorUserID: admin.ID, IsAdmin: true}

	if _, err := deviceSvc.Create(scope, DevicePayload{}); err == nil || !strings.Contains(err.Error(), "设备标识") {
		t.Fatalf("create without device name error = %v, want device name required", err)
	}

	if _, err := deviceSvc.Create(scope, DevicePayload{
		DeviceName: "SN-PRE-REGISTER-002",
		NickName:   "预登记设备",
	}); err == nil || !strings.Contains(err.Error(), "智能体") {
		t.Fatalf("create without agent error = %v, want agent required", err)
	}

	created, err := deviceSvc.Create(scope, DevicePayload{
		DeviceName: "SN-PRE-REGISTER-001",
		AgentID:    agent.ID,
	})
	if err != nil {
		t.Fatalf("admin pre-register device: %v", err)
	}
	if created.UserID != 0 {
		t.Fatalf("pre-registered device user_id = %d, want 0", created.UserID)
	}
	if created.NickName != "" {
		t.Fatalf("pre-registered device nick_name = %q, want empty", created.NickName)
	}
	if created.AgentID != agent.ID {
		t.Fatalf("pre-registered device agent_id = %d, want %d", created.AgentID, agent.ID)
	}
	if created.Activated {
		t.Fatalf("pre-registered device activated = true, want false by default")
	}
	if created.DeviceName != "SN-PRE-REGISTER-001" {
		t.Fatalf("device_name = %q, want SN-PRE-REGISTER-001", created.DeviceName)
	}
}

func TestDeviceServiceCreateDuplicateDeviceName(t *testing.T) {
	db := setupAgentDeviceServiceTestDB(t)
	admin := createServiceTestUser(t, db, "dup-device-admin", "admin")
	agent := models.Agent{UserID: admin.ID, Name: "admin-agent", Nickname: "admin-agent"}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create admin agent: %v", err)
	}
	deviceSvc := NewDeviceService(db)
	scope := accessScope{ActorUserID: admin.ID, IsAdmin: true}

	_, err := deviceSvc.Create(scope, DevicePayload{
		DeviceName: "SN-DUP-001",
		AgentID:    agent.ID,
	})
	if err != nil {
		t.Fatalf("create first device: %v", err)
	}

	_, err = deviceSvc.Create(scope, DevicePayload{
		DeviceName: "SN-DUP-001",
		AgentID:    agent.ID,
	})
	if err == nil || !strings.Contains(err.Error(), "设备已添加") {
		t.Fatalf("duplicate create error = %v, want 设备已添加", err)
	}
}
