package client

import (
	"testing"
	"time"
)

func TestRecentStoryPointerTTL(t *testing.T) {
	cs := &ClientState{}
	cs.SetRecentStoryPointer("s1", "标题", "主题")
	if _, ok := cs.RecentStoryPointer(time.Hour); !ok {
		t.Fatal("expected valid pointer")
	}
	cs.recentStoryAt = time.Now().Add(-2 * time.Hour)
	if _, ok := cs.RecentStoryPointer(time.Hour); ok {
		t.Fatal("expected expired")
	}
}
