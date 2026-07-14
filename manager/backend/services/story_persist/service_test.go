package story_persist

import (
	"context"
	"errors"
	"testing"
	"time"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.StoryAsset{}, &models.StoryAssetAlias{}, &models.StoryPlayback{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUpsertNamedAliasAndFind(t *testing.T) {
	db := setupDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.UpsertAsset(ctx, UpsertAssetRequest{
		StoryID:            "s1",
		Title:              "哪吒闹海",
		ThemeKey:           "哪吒闹海",
		PoolKind:           PoolNamed,
		CanonicalKey:       "哪吒闹海",
		Aliases:            []string{"哪吒三太子闹海", "哪吒脑海"},
		NarrationMode:      "canonical",
		FullText:           "很久很久以前哪吒闹海……",
		Segments:           []string{"很久很久以前哪吒闹海……"},
		GenerationComplete: true,
		CreatorDeviceSN:    "SN1",
	})
	if err != nil {
		t.Fatal(err)
	}

	view, err := svc.FindShareable(ctx, FindShareableQuery{
		PoolKind: PoolNamed,
		Theme:    "哪吒三太子闹海",
		TopK:     5,
		RandSeed: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.StoryID != "s1" {
		t.Fatalf("got %s", view.StoryID)
	}

	// typo alias
	view2, err := svc.FindShareable(ctx, FindShareableQuery{
		PoolKind: PoolNamed,
		Theme:    "哪吒脑海",
		TopK:     5,
		RandSeed: 2,
	})
	if err != nil || view2.StoryID != "s1" {
		t.Fatalf("alias typo: err=%v id=%v", err, view2)
	}
}

func TestFindOpenExcludeRecent(t *testing.T) {
	db := setupDB(t)
	svc := NewService(db)
	ctx := context.Background()

	_ = svc.UpsertAsset(ctx, UpsertAssetRequest{
		StoryID: "o1", Title: "开放一", PoolKind: PoolOpen,
		FullText: "故事一。", GenerationComplete: true,
	})
	_ = svc.UpsertAsset(ctx, UpsertAssetRequest{
		StoryID: "o2", Title: "开放二", PoolKind: PoolOpen,
		FullText: "故事二。", GenerationComplete: true,
	})
	_ = svc.UpsertPlayback(ctx, UpsertPlaybackRequest{
		DeviceSN: "DEV1", StoryID: "o1", LastPlayedAt: time.Now(),
	})

	view, err := svc.FindShareable(ctx, FindShareableQuery{
		PoolKind:    PoolOpen,
		DeviceSN:    "DEV1",
		ExcludeDays: 7,
		TopK:        5,
		RandSeed:    42,
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.StoryID == "o1" {
		t.Fatal("should exclude recently played o1")
	}
	if view.StoryID != "o2" {
		t.Fatalf("want o2 got %s", view.StoryID)
	}
}

func TestBedtimeFallsBackToOpen(t *testing.T) {
	db := setupDB(t)
	svc := NewService(db)
	ctx := context.Background()
	_ = svc.UpsertAsset(ctx, UpsertAssetRequest{
		StoryID: "open-only", Title: "随便讲", PoolKind: PoolOpen,
		FullText: "开放故事。", GenerationComplete: true,
	})
	view, err := svc.FindShareable(ctx, FindShareableQuery{
		PoolKind: PoolBedtime,
		TopK:     5,
		RandSeed: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.StoryID != "open-only" {
		t.Fatalf("want open fallback, got %s", view.StoryID)
	}
}

func TestDeletePlaybackKeepsAsset(t *testing.T) {
	db := setupDB(t)
	svc := NewService(db)
	ctx := context.Background()
	_ = svc.UpsertAsset(ctx, UpsertAssetRequest{
		StoryID: "keep-asset", Title: "保留资产", PoolKind: PoolNamed, CanonicalKey: "后羿射日",
		FullText: "正文。", GenerationComplete: true,
	})
	_ = svc.UpsertPlayback(ctx, UpsertPlaybackRequest{
		DeviceSN: "DEV-X", StoryID: "keep-asset", LastPlayedAt: time.Now(),
	})
	n, err := svc.DeletePlayback(ctx, "DEV-X", "keep-asset")
	if err != nil || n != 1 {
		t.Fatalf("delete playback: n=%d err=%v", n, err)
	}
	if _, err := svc.GetAsset(ctx, "keep-asset"); err != nil {
		t.Fatal("asset should remain")
	}
	n2, err := svc.DeletePlaybacksByDevice(ctx, "DEV-X")
	if err != nil || n2 != 0 {
		t.Fatalf("clear empty: n=%d err=%v", n2, err)
	}
}

func TestListAndDeleteAssets(t *testing.T) {
	db := setupDB(t)
	svc := NewService(db)
	ctx := context.Background()
	_ = svc.UpsertAsset(ctx, UpsertAssetRequest{
		StoryID: "list1", Title: "可共享一", PoolKind: PoolNamed, CanonicalKey: "后羿射日",
		FullText: "很久以前。", GenerationComplete: true,
	})
	_ = svc.UpsertAsset(ctx, UpsertAssetRequest{
		StoryID: "list2", Title: "开放篇", PoolKind: PoolOpen,
		FullText: "随便讲。", GenerationComplete: true,
	})

	shareable := true
	res, err := svc.ListAssets(ctx, ListAssetsQuery{Shareable: &shareable, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total < 2 {
		t.Fatalf("total=%d", res.Total)
	}
	named, err := svc.ListAssets(ctx, ListAssetsQuery{PoolKind: PoolNamed, Q: "后羿", Page: 1, PageSize: 10})
	if err != nil || named.Total != 1 {
		t.Fatalf("named filter: err=%v total=%d", err, named.Total)
	}
	if err := svc.DeleteAsset(ctx, "list1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetAsset(ctx, "list1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want not found after delete, got %v", err)
	}
}

func TestCreativeNotShareable(t *testing.T) {
	db := setupDB(t)
	svc := NewService(db)
	ctx := context.Background()
	_ = svc.UpsertAsset(ctx, UpsertAssetRequest{
		StoryID:            "c1",
		Title:              "小恐龙",
		ThemeKey:           "小恐龙",
		PoolKind:           "", // 不入池
		NarrationMode:      "creative",
		FullText:           "恐龙出去玩。",
		GenerationComplete: true,
	})
	if _, err := svc.FindShareable(ctx, FindShareableQuery{PoolKind: PoolNamed, Theme: "小恐龙"}); err == nil {
		t.Fatal("creative without pool should not be found")
	}
}
