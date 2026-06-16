package chat

import (
	"testing"
	"time"

	"dili-esp32-server-golang/internal/domain/chat/intent"
)

func TestSelectParentMessageByFilterRoleAndTimeRange(t *testing.T) {
	loc := time.Local
	yesterdayMorning := time.Date(2026, 6, 14, 8, 0, 0, 0, loc)
	todayAfternoon := time.Date(2026, 6, 15, 15, 0, 0, 0, loc)

	messages := []parentMessageItem{
		{ID: 1, FamilyRole: "妈妈", CreatedAt: yesterdayMorning, TextContent: "a"},
		{ID: 2, FamilyRole: "爸爸", CreatedAt: todayAfternoon, TextContent: "b"},
		{ID: 3, FamilyRole: "妈妈", CreatedAt: todayAfternoon, TextContent: "c"},
	}

	got, ok := selectParentMessageByFilter(messages, parentMessageSelectFilter{
		FamilyRole: "妈妈",
		Start:      time.Date(2026, 6, 14, 5, 0, 0, 0, loc),
		End:        time.Date(2026, 6, 14, 11, 0, 0, 0, loc),
		HasStart:   true,
		HasEnd:     true,
	})
	if !ok || got.ID != 1 {
		t.Fatalf("expected message 1, got %+v ok=%v", got, ok)
	}

	got, ok = selectParentMessageByFilter(messages, parentMessageSelectFilter{
		FamilyRole: "爸爸",
		Start:      time.Date(2026, 6, 15, 12, 0, 0, 0, loc),
		End:        time.Date(2026, 6, 15, 19, 0, 0, 0, loc),
		HasStart:   true,
		HasEnd:     true,
	})
	if !ok || got.ID != 2 {
		t.Fatalf("expected message 2, got %+v ok=%v", got, ok)
	}
}

func TestBuildParentMessageSelectFilterFromIntentData(t *testing.T) {
	filter, ok := buildParentMessageSelectFilter(intent.MsgPlayData{
		FamilyRole: "妈妈",
		Start:      "2026-06-14T05:00:00",
		End:        "2026-06-14T11:00:00",
	})
	if !ok || filter.FamilyRole != "妈妈" || !filter.HasStart || !filter.HasEnd {
		t.Fatalf("unexpected filter: %+v ok=%v", filter, ok)
	}
}

func TestFamilyRoleMatchesAliases(t *testing.T) {
	if !familyRoleMatches("妈妈", "母亲") {
		t.Fatal("妈妈/母亲 should match")
	}
	if !familyRoleMatches("爸爸", "爸") {
		t.Fatal("爸爸/爸 should match")
	}
}
