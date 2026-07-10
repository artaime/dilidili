package llm_memory

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestResetMemoryDeletesHistoryAndSystemPrompt(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	mem := &Memory{
		redisClient: client,
		keyPrefix:   "test",
	}
	ctx := context.Background()
	deviceID := "SN-LLM-001"

	historyKey := mem.getMemoryKey(deviceID)
	systemKey := mem.getSystemPromptKey(deviceID)
	if err := client.Set(ctx, historyKey, "payload", 0).Err(); err != nil {
		t.Fatalf("set history: %v", err)
	}
	if err := client.Set(ctx, systemKey, "prompt", 0).Err(); err != nil {
		t.Fatalf("set system: %v", err)
	}

	if err := mem.ResetMemory(ctx, deviceID); err != nil {
		t.Fatalf("ResetMemory: %v", err)
	}

	if mr.Exists(historyKey) {
		t.Fatal("history key still exists")
	}
	if mr.Exists(systemKey) {
		t.Fatal("system prompt key still exists")
	}
}
