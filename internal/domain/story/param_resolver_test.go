package story

import (
	"testing"
	"time"
)

func TestResolveParamsMissingAge(t *testing.T) {
	cfg := Config{MemoryHintMinConfidence: 0.75}
	res := ResolveParams(StoryParams{}, "", cfg, time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC))
	if res.Params.AgeBand != "primary_low" {
		t.Fatalf("expected default age band, got %s", res.Params.AgeBand)
	}
	if len(res.Missing) != 0 {
		t.Fatalf("expected no missing fields for voice default, got %+v", res.Missing)
	}
}

func TestResolveParamsFromMemoryHint(t *testing.T) {
	cfg := Config{MemoryHintMinConfidence: 0.75}
	y := 5
	res := ResolveParams(StoryParams{
		MemoryHints: []MemoryHint{
			{Field: "age", Value: "5", Confidence: 0.9, Source: "memory"},
		},
	}, "", cfg, time.Now())
	if res.Params.AgeBand != "preschool" {
		t.Fatalf("expected preschool, got %s", res.Params.AgeBand)
	}
	if res.Params.AgeYears == nil || *res.Params.AgeYears != y {
		t.Fatalf("expected age 5")
	}
}

func TestResolveParamsCasualDefault(t *testing.T) {
	cfg := Config{MemoryHintMinConfidence: 0.75}
	res := ResolveParams(StoryParams{UserSaidCasual: true}, "", cfg, time.Now())
	if res.Params.AgeBand != "primary_low" {
		t.Fatalf("expected default casual band, got %s", res.Params.AgeBand)
	}
	if len(res.Missing) != 0 {
		t.Fatalf("casual should not missing: %+v", res.Missing)
	}
}

func TestResolveParamsBedtimeInference(t *testing.T) {
	cfg := Config{MemoryHintMinConfidence: 0.75}
	res := ResolveParams(StoryParams{AgeBand: "preschool"}, "", cfg, time.Date(2026, 6, 30, 21, 0, 0, 0, time.UTC))
	if res.Params.IsBedtime == nil || !*res.Params.IsBedtime {
		t.Fatal("expected bedtime inference at 21:00")
	}
}
