package device_story

import (
	"context"
	"testing"

	"dili/manager/backend/models"
	"dili/manager/backend/services/story_persist"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDeleteDeviceStoryMySQLOnly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_story_del?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Device{}, &models.StoryAsset{}, &models.StoryPlayback{}); err != nil {
		t.Fatal(err)
	}
	dev := models.Device{UserID: 1, DeviceName: "SN-DEL-1"}
	if err := db.Create(&dev).Error; err != nil {
		t.Fatal(err)
	}

	persist := story_persist.NewService(db)
	ctx := context.Background()
	_ = persist.UpsertAsset(ctx, story_persist.UpsertAssetRequest{
		StoryID: "s-del", Title: "测试", PoolKind: story_persist.PoolOpen,
		FullText: "正文", GenerationComplete: true,
	})
	_ = persist.UpsertPlayback(ctx, story_persist.UpsertPlaybackRequest{
		DeviceSN: "SN-DEL-1", StoryID: "s-del",
	})

	svc := NewService(db, nil) // Redis 未配置 → skip
	result, err := svc.DeleteDeviceStory(ctx, dev.ID, "s-del")
	if err != nil {
		t.Fatal(err)
	}
	if result.PlaybackDeleted != 1 || !result.RedisSkipped {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := persist.GetAsset(ctx, "s-del"); err != nil {
		t.Fatal("asset must remain")
	}

	clr, err := svc.ClearDeviceStories(ctx, dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if clr.PlaybackDeleted != 0 {
		t.Fatalf("expected 0, got %d", clr.PlaybackDeleted)
	}
}

func TestClearDeviceStoriesPlaybacks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_story_clr?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Device{}, &models.StoryAsset{}, &models.StoryPlayback{}); err != nil {
		t.Fatal(err)
	}
	dev := models.Device{UserID: 1, DeviceName: "SN-CLR"}
	if err := db.Create(&dev).Error; err != nil {
		t.Fatal(err)
	}
	persist := story_persist.NewService(db)
	ctx := context.Background()
	for _, id := range []string{"a1", "a2"} {
		_ = persist.UpsertAsset(ctx, story_persist.UpsertAssetRequest{
			StoryID: id, Title: id, PoolKind: story_persist.PoolOpen,
			FullText: "t", GenerationComplete: true,
		})
		_ = persist.UpsertPlayback(ctx, story_persist.UpsertPlaybackRequest{
			DeviceSN: "SN-CLR", StoryID: id,
		})
	}
	svc := NewService(db, nil)
	res, err := svc.ClearDeviceStories(ctx, dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.PlaybackDeleted != 2 {
		t.Fatalf("want 2 got %d", res.PlaybackDeleted)
	}
	for _, id := range []string{"a1", "a2"} {
		if _, err := persist.GetAsset(ctx, id); err != nil {
			t.Fatalf("asset %s should remain: %v", id, err)
		}
	}
}
