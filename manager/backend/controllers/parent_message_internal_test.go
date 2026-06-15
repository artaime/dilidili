package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
)

func TestGetPendingMessageJSONRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupMpMessageTestDB(t)
	user := createServiceTestUser(t, db, "pm-json", "user")
	user.FamilyRole = "妈妈"
	if err := db.Save(&user).Error; err != nil {
		t.Fatalf("save user: %v", err)
	}
	device := models.Device{UserID: user.ID, DeviceName: "11:22:33:44:55:66", DeviceCode: "123456", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	msg := models.ParentMessage{
		UserID: user.ID, DeviceID: device.ID,
		SourceType: "voice", AudioPath: "/tmp/test.mp3",
		Status: "pending", CreatedAt: time.Now(),
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	ctrl := &ParentMessageInternalController{DB: db}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/internal/devices/11:22:33:44:55:66/parent-messages/pending", nil)
	c.Params = gin.Params{{Key: "device_name", Value: "11:22:33:44:55:66"}}
	ctrl.GetPendingMessage(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	body := w.Body.Bytes()
	t.Logf("response: %s", string(body))

	var payload struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("data len = %d", len(payload.Data))
	}
	createdAt, ok := payload.Data[0]["created_at"].(string)
	if !ok {
		t.Fatalf("created_at type = %T value = %v", payload.Data[0]["created_at"], payload.Data[0]["created_at"])
	}
	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Fatalf("created_at not RFC3339: %q err=%v", createdAt, err)
	}
}
