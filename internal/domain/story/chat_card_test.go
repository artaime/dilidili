package story

import "testing"

func TestStoryCardContent(t *testing.T) {
	if got := StoryCardContent("女娲补天"); got != "播放故事：女娲补天" {
		t.Fatalf("got %q", got)
	}
	if got := StoryCardContent("  "); got != "播放故事：故事" {
		t.Fatalf("empty title got %q", got)
	}
}

func TestMergeStoryCardMetadata(t *testing.T) {
	dst := map[string]any{"timestamp": "x"}
	MergeStoryCardMetadata(dst, StoryCardExtra("sid", "标题", true))
	if dst[ExtraKeyKind] != MessageKindStoryCard || dst[ExtraKeyStoryID] != "sid" {
		t.Fatalf("merged=%v", dst)
	}
	MergeStoryCardMetadata(dst, map[string]any{ExtraKeyKind: "other"})
	if dst[ExtraKeyStoryID] != "sid" {
		t.Fatal("should ignore non-story extra")
	}
}
