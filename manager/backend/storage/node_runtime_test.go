package storage

import (
	"testing"
	"time"
)

func TestNodeRuntimeStoreUpsertAndSummary(t *testing.T) {
	store := &NodeRuntimeStore{
		nodes:        make(map[string]*NodeRuntimeSnapshot),
		offlineAfter: 30 * time.Second,
	}
	store.Upsert(RuntimeReportInput{
		NodeID:   "node-a",
		NodeName: "节点A",
		ReportedAt: time.Now(),
		App: AppMetricsSnapshot{
			ChatManagerCount:   10,
			ActiveSessionCount: 2,
		},
	}, true)
	store.Upsert(RuntimeReportInput{
		NodeID:   "node-b",
		NodeName: "节点B",
		ReportedAt: time.Now(),
		App: AppMetricsSnapshot{
			ChatManagerCount:   5,
			ActiveSessionCount: 1,
		},
	}, false)

	summary := store.GetClusterSummary()
	if summary.TotalNodes != 2 {
		t.Fatalf("expected 2 nodes, got %d", summary.TotalNodes)
	}
	if summary.OnlineNodes != 2 {
		t.Fatalf("expected 2 online nodes, got %d", summary.OnlineNodes)
	}
	if summary.TotalChatManagers != 15 {
		t.Fatalf("expected 15 chat managers, got %d", summary.TotalChatManagers)
	}
	if summary.TotalActiveSessions != 3 {
		t.Fatalf("expected 3 active sessions, got %d", summary.TotalActiveSessions)
	}
}

func TestNodeRuntimeStoreOfflineAfterTimeout(t *testing.T) {
	store := &NodeRuntimeStore{
		nodes:        make(map[string]*NodeRuntimeSnapshot),
		offlineAfter: 30 * time.Second,
	}
	store.Upsert(RuntimeReportInput{
		NodeID:     "node-offline",
		ReportedAt: time.Now().Add(-60 * time.Second),
	}, false)

	node, ok := store.Get("node-offline")
	if !ok {
		t.Fatal("expected node to exist")
	}
	if node.Status != NodeStatusOffline {
		t.Fatalf("expected offline status, got %s", node.Status)
	}
}
