package chat

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	. "dili-esp32-server-golang/internal/data/client"
	"dili-esp32-server-golang/internal/domain/story"
)

func TestRecentPointerMatchesTheme(t *testing.T) {
	ptr := RecentStoryPointer{Title: "哪吒闹海", Theme: "哪吒闹海"}
	if !recentPointerMatchesTheme(ptr, "哪吒闹海") {
		t.Fatal("expected match")
	}
	if recentPointerMatchesTheme(ptr, "女娲补天") {
		t.Fatal("expected no match")
	}
}

func TestBuildStoryFollowupBriefMaxRunes(t *testing.T) {
	body := strings.Repeat("啊", 100)
	rec := &story.StoryRecord{Title: "测试", FullText: body}
	brief := story.BuildStoryFollowupBrief(rec, 20)
	if !strings.Contains(brief, "标题：测试") {
		t.Fatalf("missing title: %s", brief)
	}
	if utf8.RuneCountInString(brief) < 20 {
		t.Fatalf("brief too short: %d", utf8.RuneCountInString(brief))
	}
}

func TestClientRecentStoryPointer(t *testing.T) {
	cs := &ClientState{}
	cs.SetRecentStoryPointer("s1", "标题", "主题")
	ptr, ok := cs.RecentStoryPointer(time.Hour)
	if !ok || ptr.StoryID != "s1" || ptr.Title != "标题" || ptr.Theme != "主题" {
		t.Fatalf("unexpected ptr: %+v ok=%v", ptr, ok)
	}
	cs2 := &ClientState{}
	if _, ok := cs2.RecentStoryPointer(time.Hour); ok {
		t.Fatal("empty pointer should be invalid")
	}
}
