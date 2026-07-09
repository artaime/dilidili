package chat

import (
	"context"
	"errors"
	"testing"
)

func TestStoryFailureReason(t *testing.T) {
	if r := storyFailureReason(true, context.Canceled, context.Canceled); r != "user_tts_interrupt" {
		t.Fatalf("got %q", r)
	}
	if r := storyFailureReason(false, nil, context.DeadlineExceeded); r != "llm_timeout" {
		t.Fatalf("got %q", r)
	}
	if r := storyFailureReason(false, nil, errors.New("api error")); r == "" || r == "unknown" {
		t.Fatalf("got %q", r)
	}
}
