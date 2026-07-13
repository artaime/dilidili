package tencent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	asrtypes "dili-esp32-server-golang/internal/domain/asr/types"

	"github.com/gorilla/websocket"
)

func TestConfigFromMap(t *testing.T) {
	cfg, err := ConfigFromMap(map[string]interface{}{
		"app_id":            1300460000,
		"secret_id":         "AKIDTEST",
		"secret_key":        "secret-key",
		"engine_model_type": "16k_zh",
		"sample_rate":       16000,
		"timeout":           30,
	})
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}
	if cfg.AppID != 1300460000 {
		t.Fatalf("app_id = %d", cfg.AppID)
	}
	if cfg.EngineModelType != "16k_zh" {
		t.Fatalf("engine_model_type = %q", cfg.EngineModelType)
	}
}

func TestSignTencentParams(t *testing.T) {
	params := map[string]string{
		"engine_model_type": "16k_zh",
		"expired":           "1688697305",
		"needvad":           "0",
		"nonce":             "8743357",
		"secretid":          "AKIDTEST",
		"timestamp":         "1688610905",
		"voice_format":      "1",
		"voice_id":          "session-1",
	}
	signature, err := signTencentParams(params, "secret-key", 1300460000)
	if err != nil {
		t.Fatalf("signTencentParams failed: %v", err)
	}
	if signature == "" {
		t.Fatal("expected non-empty signature")
	}
}

func TestBuildSignedURLContainsRequiredParams(t *testing.T) {
	a := &ASR{config: Config{
		AppID:           1300460000,
		SecretID:        "AKIDTEST",
		SecretKey:       "secret-key",
		EngineModelType: "16k_zh",
		VoiceFormat:     1,
		SampleRate:      16000,
		ConnectTimeout:  5 * time.Second,
	}}
	signedURL, err := a.buildSignedURL("voice-abc")
	if err != nil {
		t.Fatalf("buildSignedURL failed: %v", err)
	}
	for _, part := range []string{"engine_model_type=", "secretid=", "signature=", "voice_id=", "asr.cloud.tencent.com/asr/v2/1300460000"} {
		if !strings.Contains(signedURL, part) {
			t.Fatalf("signed url missing %s: %s", part, signedURL)
		}
	}
}

func TestHandleStreamingMockServer(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"success","voice_id":"voice-1"}`))

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if messageType == websocket.TextMessage {
				var end endMessage
				if json.Unmarshal(message, &end) == nil && end.Type == "end" {
					_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"success","voice_id":"voice-1","result":{"slice_type":2,"voice_text_str":"你好"}}`))
					_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"success","voice_id":"voice-1","final":1}`))
					return
				}
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial mock server failed: %v", err)
	}

	a := &ASR{config: Config{
		AppID:           1300460000,
		SecretID:        "AKIDTEST",
		SecretKey:       "secret-key",
		EngineModelType: "16k_zh",
		VoiceFormat:     1,
		SampleRate:      16000,
		ConnectTimeout:  5 * time.Second,
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	audioStream := make(chan []float32, 1)
	audioStream <- []float32{0.1, -0.1, 0.2}
	close(audioStream)

	resultChan := make(chan asrtypes.StreamingResult, 10)
	a.handleStreaming(ctx, conn, audioStream, resultChan)

	var finalText string
	for result := range resultChan {
		if result.Error != nil {
			t.Fatalf("streaming error: %v", result.Error)
		}
		if result.Text != "" {
			finalText = result.Text
		}
	}
	if finalText != "你好" {
		t.Fatalf("expected 你好, got %q", finalText)
	}
}

func TestFloat32ToPCMBytes(t *testing.T) {
	pcm := float32ToPCMBytes([]float32{0.5, -0.5})
	if len(pcm) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(pcm))
	}
}

func TestClassifyTencentRetryReason(t *testing.T) {
	if got := classifyTencentRetryReason(4008, "客户端超过15秒未发送音频数据"); got != asrtypes.RetryReasonTencentNoAudioTimeout {
		t.Fatalf("4008: got %q", got)
	}
	if got := classifyTencentRetryReason(5000, "客户端超过15秒未发送音频数据"); got != asrtypes.RetryReasonTencentNoAudioTimeout {
		t.Fatalf("message match: got %q", got)
	}
	if got := classifyTencentRetryReason(5000, "internal error"); got != asrtypes.RetryReasonNone {
		t.Fatalf("non-retryable: got %q", got)
	}
}
