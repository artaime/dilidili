package story

import (
	"context"
	"testing"
	"time"
)

func TestStoreSaveGetReplay(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90}
	mem := newMemoryBackend()
	store := NewStoreWithBackend(mem, "test", cfg)
	ctx := context.Background()

	rec := &StoryRecord{
		DeviceID:  "dev1",
		AgentID:   "agent1",
		Title:     "测试故事",
		FullText:  "从前有一只小云。它很勇敢。",
		Segments:  []string{"从前有一只小云。", "它很勇敢。"},
		AgeBand:   "preschool",
		CreatedAt: time.Now(),
	}
	if err := store.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}
	if rec.StoryID == "" {
		t.Fatal("expected story id assigned")
	}

	got, err := store.GetLast(ctx, "dev1")
	if err != nil {
		t.Fatal(err)
	}
	if got.FullText != rec.FullText {
		t.Fatalf("full text mismatch")
	}

	_ = store.RecordPlayStart(ctx, "dev1", rec.StoryID)
	got2, _ := store.Get(ctx, "dev1", rec.StoryID)
	if got2.PlayCount < 1 {
		t.Fatalf("expected play count incremented, got %d", got2.PlayCount)
	}
}

func TestStoreListInWindow(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90}
	mem := newMemoryBackend()
	store := NewStoreWithBackend(mem, "test", cfg)
	ctx := context.Background()

	// 相对「此刻」构造昨晚窗口内的时间，避免固定日期被 retention 淘汰。
	now := time.Now()
	start, end := LastNightWindow(now, 18, 7)
	playTime := start.Add(3 * time.Hour)
	if !playTime.Before(end) {
		playTime = start.Add(30 * time.Minute)
	}
	rec := &StoryRecord{
		DeviceID: "dev1", Title: "睡前故事", FullText: "晚安。",
		Segments: []string{"晚安。"}, CreatedAt: playTime, LastPlayedAt: playTime,
	}
	if err := store.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}

	list, err := store.ListInWindow(ctx, "dev1", start, end, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 story in last night window, got %d (window %v~%v play %v)", len(list), start, end, playTime)
	}
}

func TestGetLastEmptyReturnsNilError(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90}
	store := NewStoreWithBackend(newMemoryBackend(), "test", cfg)
	rec, err := store.GetLast(context.Background(), "no-stories")
	if err == nil {
		t.Fatal("expected error when no stories")
	}
	if rec != nil {
		t.Fatal("expected nil record when no stories")
	}
}

func TestHandleReplayLastWithNoHistory(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90, Enabled: true}
	store := NewStoreWithBackend(newMemoryBackend(), "test", cfg)
	svc := NewServiceWithStore(store, cfg, nil)
	res, err := svc.Handle(context.Background(), ToolRequest{
		Action:   ActionReplay,
		StoryRef: StoryRefLast,
		DeviceID: "dev-empty",
		Now:      time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Status != StatusNotFound {
		t.Fatalf("expected not_found, got %+v", res)
	}
}

func TestServiceGenerateAndReplay(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90, Enabled: true}
	mem := newMemoryBackend()
	store := NewStoreWithBackend(mem, "test", cfg)
	svc := NewServiceWithStore(store, cfg, func(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
		return "小兔子找胡萝卜。它找到了。很开心。", nil
	})
	ctx := context.Background()
	now := time.Now()

	gen, err := svc.Handle(ctx, ToolRequest{
		Action:   ActionGenerate,
		DeviceID: "dev1",
		AgentID:  "a1",
		StoryParams: StoryParams{
			AgeBand: "preschool",
		},
		Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gen.Status != StatusReady || gen.StoryID == "" {
		t.Fatalf("unexpected generate result: %+v", gen)
	}

	rep, err := svc.Handle(ctx, ToolRequest{
		Action:   ActionReplay,
		StoryRef: StoryRefLast,
		DeviceID: "dev1",
		Now:      now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != StatusReplay {
		t.Fatalf("expected replay, got %s", rep.Status)
	}
	if rep.TextToSpeak == "" {
		t.Fatal("expected text to speak")
	}
}

func TestFindLatestByTheme(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90}
	mem := newMemoryBackend()
	store := NewStoreWithBackend(mem, "test", cfg)
	ctx := context.Background()
	now := time.Now()

	emptyDraft := &StoryRecord{
		DeviceID: "dev1", Title: "女娲补天的故事", FullText: "",
		ParamsSnapshot: map[string]any{"theme": "女娲补天"},
		CreatedAt: now.Add(-time.Hour), LastPlayedAt: now,
	}
	if err := store.Save(ctx, emptyDraft); err != nil {
		t.Fatal(err)
	}

	full := &StoryRecord{
		DeviceID: "dev1", Title: "女娲补天的故事",
		FullText: "很久很久以前，天塌了。", Segments: []string{"很久很久以前，天塌了。"},
		ParamsSnapshot: map[string]any{"theme": "女娲补天"},
		CreatedAt: now, LastPlayedAt: now,
	}
	if err := store.Save(ctx, full); err != nil {
		t.Fatal(err)
	}

	got, err := store.FindLatestByTheme(ctx, "dev1", "女娲补天", true)
	if err != nil {
		t.Fatal(err)
	}
	if got.StoryID != full.StoryID {
		t.Fatalf("expected full story id, got %s", got.StoryID)
	}

	if _, err := store.FindLatestByTheme(ctx, "dev1", "共工怒触不周山", true); err == nil {
		t.Fatal("expected missing theme")
	}
}

func TestServiceReplayByTheme(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90, Enabled: true}
	mem := newMemoryBackend()
	store := NewStoreWithBackend(mem, "test", cfg)
	generateCalls := 0
	svc := NewServiceWithStore(store, cfg, func(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
		generateCalls++
		return "重新生成的故事正文。", nil
	})
	ctx := context.Background()
	now := time.Now()

	rec := &StoryRecord{
		DeviceID: "dev1", Title: "女娲补天的故事",
		FullText: "女娲炼五色石补天。", Segments: []string{"女娲炼五色石补天。"},
		ParamsSnapshot: map[string]any{"theme": "女娲补天"},
		CreatedAt: now, LastPlayedAt: now,
	}
	if err := store.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}

	rep, err := svc.Handle(ctx, ToolRequest{
		Action: ActionReplay,
		StoryParams: StoryParams{Theme: "女娲补天"},
		DeviceID: "dev1",
		Now:      now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != StatusReplay || rep.TextToSpeak == "" {
		t.Fatalf("unexpected replay: %+v", rep)
	}
	if generateCalls != 0 {
		t.Fatalf("expected no regenerate, got %d", generateCalls)
	}

	// 仅有空草稿时应回退重新生成。
	_ = store.Save(ctx, &StoryRecord{
		DeviceID: "dev1", Title: "共工怒触不周山的故事", FullText: "",
		ParamsSnapshot: map[string]any{"theme": "共工怒触不周山"},
		CreatedAt: now, LastPlayedAt: now,
	})
	regen, err := svc.Handle(ctx, ToolRequest{
		Action: ActionReplay,
		StoryParams: StoryParams{Theme: "共工怒触不周山"},
		DeviceID: "dev1",
		Now:      now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if regen.Status != StatusReady || generateCalls != 1 {
		t.Fatalf("expected regenerate, got status=%s calls=%d", regen.Status, generateCalls)
	}
}

func TestSegmentText(t *testing.T) {
	text := "第一句。第二句！第三句？"
	segs := SegmentText(text)
	if len(segs) == 0 {
		t.Fatal("expected segments")
	}
}

func TestStoreDeleteAllForDevice(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90}
	mem := newMemoryBackend()
	store := NewStoreWithBackend(mem, "test", cfg)
	ctx := context.Background()

	rec := &StoryRecord{
		DeviceID:  "dev1",
		AgentID:   "agent1",
		Title:     "待删故事",
		FullText:  "内容",
		Segments:  []string{"内容"},
		CreatedAt: time.Now(),
	}
	if err := store.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteAllForDevice(ctx, "dev1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "dev1", rec.StoryID); err == nil {
		t.Fatal("expected story removed")
	}
	if _, err := mem.Get(ctx, store.byTimeKey("dev1")); err == nil {
		t.Fatal("expected by_time key removed")
	}
}
