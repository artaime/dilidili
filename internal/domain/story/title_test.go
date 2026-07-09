package story

import "testing"

func TestResolveStoryTitlePrefersTheme(t *testing.T) {
	got := ResolveStoryTitle("普罗米修斯", "很久很久以前，天和地刚分开……", "很久很久以前，天和地刚分开")
	if got != "普罗米修斯的故事" {
		t.Fatalf("got %q want 普罗米修斯的故事", got)
	}
}

func TestResolveStoryTitleRejectsOpeningAsStoredTitle(t *testing.T) {
	got := ResolveStoryTitle("", "很久很久以前，有一个小村庄", "很久很久以前，有一个小村庄")
	if got != "儿童故事" {
		t.Fatalf("got %q want 儿童故事", got)
	}
}

func TestLooksLikeStoryOpening(t *testing.T) {
	if !looksLikeStoryOpening("好的，我来为你讲述叶公好龙") {
		t.Fatal("expected filler prefix to match")
	}
	if looksLikeStoryOpening("叶公好龙") {
		t.Fatal("short theme should not match opening heuristic")
	}
}
