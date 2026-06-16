package chat

import "testing"

func TestFilterNewPendingMessages(t *testing.T) {
	manager := &ChatManager{
		DeviceID: "00:50:47:ba:b8:e8",
	}
	messages := []parentMessageItem{
		{ID: 1, TextContent: "a"},
		{ID: 2, TextContent: "b"},
	}
	newOnes := manager.filterNewPendingMessages(messages)
	if len(newOnes) != 2 {
		t.Fatalf("expected 2 new messages, got %d", len(newOnes))
	}
	manager.markParentMessagePendingSnapshot(messages)
	newOnes = manager.filterNewPendingMessages(messages)
	if len(newOnes) != 0 {
		t.Fatalf("expected 0 new messages after snapshot, got %d", len(newOnes))
	}
	newOnes = manager.filterNewPendingMessages([]parentMessageItem{
		{ID: 1, TextContent: "a"},
		{ID: 3, TextContent: "c"},
	})
	if len(newOnes) != 1 || newOnes[0].ID != 3 {
		t.Fatalf("expected only message 3 as new, got %+v", newOnes)
	}
}

func TestResetParentMessagePendingSnapshotOnHello(t *testing.T) {
	manager := &ChatManager{DeviceID: "dev"}
	manager.markParentMessagePendingSnapshot([]parentMessageItem{{ID: 1, TextContent: "x"}})
	manager.resetParentMessagePendingSnapshot()
	newOnes := manager.filterNewPendingMessages([]parentMessageItem{{ID: 1, TextContent: "x"}})
	if len(newOnes) != 1 {
		t.Fatalf("expected reset snapshot to treat message as new again")
	}
}
