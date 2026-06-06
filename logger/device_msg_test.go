package logger

import (
	"strings"
	"testing"
)

func TestSummarizeDevicePayloadHello(t *testing.T) {
	payload := []byte(`{"type":"hello","transport":"udp","version":3,"audio_params":{"format":"opus"}}`)
	summary := SummarizeDevicePayload(payload)
	if !strings.Contains(summary, "type=hello") {
		t.Fatalf("expected hello type in summary, got %s", summary)
	}
	if !strings.Contains(summary, "transport=udp") {
		t.Fatalf("expected transport in summary, got %s", summary)
	}
}

func TestSummarizeDevicePayloadRedactsUdpKey(t *testing.T) {
	payload := []byte(`{"type":"hello","udp":{"server":"1.2.3.4","port":8990,"key":"secret","encryption":"aes-128-ctr"}}`)
	summary := SummarizeDevicePayload(payload)
	if strings.Contains(summary, "secret") {
		t.Fatalf("udp key should be redacted, got %s", summary)
	}
	if !strings.Contains(summary, "udp_key=<redacted>") {
		t.Fatalf("expected redacted marker, got %s", summary)
	}
}
