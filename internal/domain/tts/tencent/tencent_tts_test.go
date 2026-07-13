package tencent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSignTencentParams(t *testing.T) {
	params := map[string]string{
		"Action":     actionTextToStreamAudioWSv2,
		"AppId":      "1300460000",
		"Codec":      "pcm",
		"Expired":    "1688697305",
		"SampleRate": "16000",
		"SecretId":   "AKIDTEST",
		"SessionId":  "session-1",
		"Speed":      "0",
		"Timestamp":  "1688610905",
		"VoiceType":  "101001",
		"Volume":     "0",
	}

	signature, err := signTencentParams(params, "secret-key")
	if err != nil {
		t.Fatalf("signTencentParams failed: %v", err)
	}
	if signature == "" {
		t.Fatal("expected non-empty signature")
	}

	encoded := encodeSortedParams(map[string]string{
		"Signature": signature,
		"Action":    params["Action"],
	})
	if !strings.Contains(encoded, "Signature=") {
		t.Fatalf("expected encoded signature, got %q", encoded)
	}
}

func TestBuildSignedURLContainsRequiredParams(t *testing.T) {
	p := NewTencentTTSProvider(map[string]interface{}{
		"app_id":     1300460000,
		"secret_id":  "AKIDTEST",
		"secret_key": "secret-key",
		"voice_type": 101001,
		"codec":      "pcm",
		"sample_rate": 16000,
	})

	signedURL, err := p.buildSignedURL("session-abc")
	if err != nil {
		t.Fatalf("buildSignedURL failed: %v", err)
	}
	for _, part := range []string{"Action=", "AppId=", "SecretId=", "Signature=", "SessionId=", "VoiceType="} {
		if !strings.Contains(signedURL, part) {
			t.Fatalf("signed url missing %s: %s", part, signedURL)
		}
	}
}

func TestParseVoiceType(t *testing.T) {
	if got := parseVoiceType(map[string]interface{}{"voice_type": 101002}); got != 101002 {
		t.Fatalf("voice_type int: got %d", got)
	}
	if got := parseVoiceType(map[string]interface{}{"voice": "101003"}); got != 101003 {
		t.Fatalf("voice string: got %d", got)
	}
}

func TestEnsureSentencePunctuation(t *testing.T) {
	if got := ensureSentencePunctuation("你好"); got != "你好。" {
		t.Fatalf("expected trailing punctuation, got %q", got)
	}
	if got := ensureSentencePunctuation("你好。"); got != "你好。" {
		t.Fatalf("expected unchanged text, got %q", got)
	}
}

func TestTextToSpeechStreamMockServer(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"success","ready":0,"final":0}`))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"success","ready":1,"final":0}`))

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var cmd tencentClientMessage
			if err := json.Unmarshal(message, &cmd); err != nil {
				return
			}
			if cmd.Action == "ACTION_SYNTHESIS" {
				pcm := make([]byte, 320)
				for i := range pcm {
					pcm[i] = byte(i % 128)
				}
				_ = conn.WriteMessage(websocket.BinaryMessage, pcm)
			}
			if cmd.Action == "ACTION_COMPLETE" {
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"success","ready":0,"final":1}`))
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	p := NewTencentTTSProvider(map[string]interface{}{
		"app_id":      1300460000,
		"secret_id":   "AKIDTEST",
		"secret_key":  "secret-key",
		"voice_type":  101001,
		"codec":       "pcm",
		"sample_rate": 16000,
		"ws_url":      wsURL,
	})

	ctx, cancel := contextWithTimeout(t, 5*time.Second)
	defer cancel()

	ch, err := p.TextToSpeechStream(ctx, "你好", 16000, 1, 60)
	if err != nil {
		t.Fatalf("TextToSpeechStream failed: %v", err)
	}

	frameCount := 0
	for range ch {
		frameCount++
	}
	if frameCount == 0 {
		t.Fatal("expected at least one audio frame")
	}
}

func TestNewTencentTTSProviderDefaults(t *testing.T) {
	p := NewTencentTTSProvider(map[string]interface{}{
		"app_id":     1,
		"secret_id":  "id",
		"secret_key": "key",
	})
	if p.Codec != defaultTencentCodec {
		t.Fatalf("codec default: got %q", p.Codec)
	}
	if p.SampleRate != defaultTencentSampleRate {
		t.Fatalf("sample_rate default: got %d", p.SampleRate)
	}
	if p.VoiceType != defaultTencentVoiceType {
		t.Fatalf("voice_type default: got %d", p.VoiceType)
	}
}

func TestStreamingSynthesizeMultiChunkMockServer(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	synthesisCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"success","ready":0,"final":0}`))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"success","ready":1,"final":0}`))

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var cmd tencentClientMessage
			if err := json.Unmarshal(message, &cmd); err != nil {
				return
			}
			if cmd.Action == "ACTION_SYNTHESIS" {
				synthesisCount++
				pcm := make([]byte, 320)
				_ = conn.WriteMessage(websocket.BinaryMessage, pcm)
			}
			if cmd.Action == "ACTION_COMPLETE" {
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"success","ready":0,"final":1}`))
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	p := NewTencentTTSProvider(map[string]interface{}{
		"app_id":      1300460000,
		"secret_id":   "AKIDTEST",
		"secret_key":  "secret-key",
		"voice_type":  101001,
		"codec":       "pcm",
		"sample_rate": 16000,
		"ws_url":      wsURL,
	})

	ctx, cancel := contextWithTimeout(t, 5*time.Second)
	defer cancel()

	textChan := make(chan string, 4)
	eventChan, err := p.StreamingSynthesize(ctx, textChan, 16000, 1, 60)
	if err != nil {
		t.Fatalf("StreamingSynthesize failed: %v", err)
	}

	go func() {
		textChan <- "从前有座山。"
		textChan <- "山里有座庙。"
		close(textChan)
	}()

	frameCount := 0
	for event := range eventChan {
		frameCount += len(event.Audio)
	}
	if synthesisCount != 2 {
		t.Fatalf("expected 2 ACTION_SYNTHESIS, got %d", synthesisCount)
	}
	if frameCount == 0 {
		t.Fatal("expected audio frames from multi-chunk stream")
	}
}

func contextWithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), timeout)
}
