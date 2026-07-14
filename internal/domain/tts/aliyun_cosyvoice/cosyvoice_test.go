package aliyun_cosyvoice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dili-esp32-server-golang/internal/data/audio"
)

func TestNewAliyunCosyVoiceProviderDefaultsAndSetVoice(t *testing.T) {
	provider := NewAliyunCosyVoiceProvider(map[string]interface{}{
		"api_key":       "sk-test",
		"workspace_id":  "ws-test",
		"model":         "cosyvoice-v3-flash",
		"voice":         "longjielidou_v3",
		"frame_duration": float64(60),
	})

	wantURL := "https://ws-test.cn-beijing.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer"
	if provider.APIURL != wantURL {
		t.Fatalf("APIURL = %q, want %q", provider.APIURL, wantURL)
	}
	if provider.Model != "cosyvoice-v3-flash" {
		t.Fatalf("Model = %q", provider.Model)
	}
	if provider.Voice != "longjielidou_v3" {
		t.Fatalf("Voice = %q", provider.Voice)
	}
	if provider.Format != defaultFormat {
		t.Fatalf("Format = %q", provider.Format)
	}
	if provider.SampleRate != defaultSampleRate {
		t.Fatalf("SampleRate = %d", provider.SampleRate)
	}
	if provider.FrameDuration != 60 {
		t.Fatalf("FrameDuration = %d", provider.FrameDuration)
	}
	if err := provider.validate(); err != nil {
		t.Fatalf("validate error = %v", err)
	}
	if err := provider.SetVoice(map[string]interface{}{"voice": "longanyang"}); err != nil {
		t.Fatalf("SetVoice error = %v", err)
	}
	if provider.Voice != "longanyang" {
		t.Fatalf("Voice = %q", provider.Voice)
	}
}

func TestNewAliyunCosyVoiceProviderCustomAPIURL(t *testing.T) {
	customURL := "https://llm-test.cn-beijing.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer"
	provider := NewAliyunCosyVoiceProvider(map[string]interface{}{
		"api_key": "sk-test",
		"api_url": customURL,
	})
	if provider.APIURL != customURL {
		t.Fatalf("APIURL = %q", provider.APIURL)
	}
}

func TestValidateRequiresAPIKeyAndEndpoint(t *testing.T) {
	p := NewAliyunCosyVoiceProvider(map[string]interface{}{})
	if err := p.validate(); err == nil || !strings.Contains(err.Error(), "api_key") {
		t.Fatalf("expected api_key error, got %v", err)
	}

	p.APIKey = "sk-test"
	if err := p.validate(); err == nil || !strings.Contains(err.Error(), "api_url") {
		t.Fatalf("expected endpoint error, got %v", err)
	}
}

func TestBuildAPIURL(t *testing.T) {
	if got := buildAPIURL("ws-abc"); got != "https://ws-abc.cn-beijing.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer" {
		t.Fatalf("buildAPIURL = %q", got)
	}
	if got := buildAPIURL(""); got != "" {
		t.Fatalf("buildAPIURL empty = %q", got)
	}
}

func TestTextToSpeechNonStreamMockServer(t *testing.T) {
	payload := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	wavBytes := makeTestWAV(payload)

	audioServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write(wavBytes)
	}))
	defer audioServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Fatal("missing bearer token")
		}
		resp := cosyVoiceResponse{
			StatusCode: 200,
			Output: cosyVoiceOutput{
				FinishReason: "stop",
				Audio: cosyVoiceAudio{
					URL: audioServer.URL,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	provider := NewAliyunCosyVoiceProvider(map[string]interface{}{
		"api_key": "sk-test",
		"api_url": apiServer.URL,
		"model":   "cosyvoice-v3-flash",
		"voice":   "longanyang",
		"format":  "wav",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	frames, err := provider.TextToSpeech(ctx, "配置测试", 24000, 1, 60)
	if err != nil {
		t.Fatalf("TextToSpeech error = %v", err)
	}
	if len(frames) == 0 {
		t.Fatal("expected audio frames")
	}
}

func TestTextToSpeechStreamMockSSE(t *testing.T) {
	payload := make([]byte, 4800)
	for i := range payload {
		payload[i] = byte(i % 256)
	}
	chunk := base64.StdEncoding.EncodeToString(payload)

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-DashScope-SSE") != "enable" {
			t.Fatal("expected SSE header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		event := cosyVoiceResponse{
			StatusCode: 200,
			Output: cosyVoiceOutput{
				Audio: cosyVoiceAudio{Data: chunk},
			},
		}
		body, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", body)
		flusher.Flush()

		stopEvent := cosyVoiceResponse{
			StatusCode: 200,
			Output: cosyVoiceOutput{
				FinishReason: "stop",
			},
		}
		stopBody, _ := json.Marshal(stopEvent)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", stopBody)
		flusher.Flush()
	}))
	defer apiServer.Close()

	provider := NewAliyunCosyVoiceProvider(map[string]interface{}{
		"api_key": "sk-test",
		"api_url": apiServer.URL,
		"model":   "cosyvoice-v3-flash",
		"voice":   "longanyang",
		"format":  "pcm",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	outputChan, err := provider.TextToSpeechStream(ctx, "配置测试", 24000, 1, 60)
	if err != nil {
		t.Fatalf("TextToSpeechStream error = %v", err)
	}

	total := 0
	deadline := time.After(8 * time.Second)
	for {
		select {
		case frame, ok := <-outputChan:
			if !ok {
				if total == 0 {
					t.Fatal("no audio received")
				}
				return
			}
			if frame != nil {
				total += len(frame)
			}
		case <-deadline:
			t.Fatalf("timeout waiting for stream audio, total=%d", total)
		}
	}
}

func TestValidateFailsBeforeStreamWithoutAPIKey(t *testing.T) {
	provider := NewAliyunCosyVoiceProvider(map[string]interface{}{
		"workspace_id": "ws-test",
	})
	_, err := provider.TextToSpeechStream(context.Background(), "test", 24000, 1, audio.FrameDuration)
	if err == nil || !strings.Contains(err.Error(), "api_key") {
		t.Fatalf("expected api_key error, got %v", err)
	}
}

func TestNormalizeLeadingAudioStripsWAVHeader(t *testing.T) {
	payload := []byte{1, 2, 3, 4}
	wav := makeTestWAV(payload)

	normalized, needMore, detectedWAV, err := normalizeLeadingAudio(wav)
	if err != nil {
		t.Fatalf("normalizeLeadingAudio error = %v", err)
	}
	if needMore || !detectedWAV {
		t.Fatalf("needMore=%v detectedWAV=%v", needMore, detectedWAV)
	}
	if !bytes.Equal(normalized, payload) {
		t.Fatalf("normalized = %v", normalized)
	}
}

func makeTestWAV(payload []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36+len(payload)))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(24000))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(48000))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(payload)))
	buf.Write(payload)
	return buf.Bytes()
}
