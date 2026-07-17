package client

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestApplyOwnerIdentityClearsDialogueOnChange(t *testing.T) {
	state := &ClientState{
		Dialogue:    &Dialogue{},
		OwnerUserID: 1,
		AgentID:     "10",
	}
	state.AddMessage(schema.UserMessage("hello"))
	if n := len(state.GetMessages(10)); n != 1 {
		t.Fatalf("setup messages=%d", n)
	}

	state.ApplyOwnerIdentity(2, "10")
	if n := len(state.GetMessages(10)); n != 0 {
		t.Fatalf("expected Dialogue cleared after user change, got %d", n)
	}
	if state.OwnerUserID != 2 {
		t.Fatalf("OwnerUserID=%d", state.OwnerUserID)
	}

	state.AddMessage(schema.UserMessage("again"))
	state.ApplyOwnerIdentity(2, "10")
	if n := len(state.GetMessages(10)); n != 1 {
		t.Fatalf("same identity should keep Dialogue, got %d", n)
	}

	state.ApplyOwnerIdentity(2, "11")
	if n := len(state.GetMessages(10)); n != 0 {
		t.Fatalf("expected Dialogue cleared after agent change, got %d", n)
	}
}
