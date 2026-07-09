package util

import (
	"context"
	"testing"
)

func TestQueueClearAndDrainReturnsBufferedItems(t *testing.T) {
	q := NewQueue[int](4)
	if err := q.Push(1); err != nil {
		t.Fatalf("push 1 failed: %v", err)
	}
	if err := q.Push(2); err != nil {
		t.Fatalf("push 2 failed: %v", err)
	}

	drained := q.ClearAndDrain()
	if len(drained) != 2 {
		t.Fatalf("expected 2 drained items, got %d", len(drained))
	}
	if drained[0] != 1 || drained[1] != 2 {
		t.Fatalf("expected drained items [1 2], got %v", drained)
	}

	if err := q.Push(3); err != nil {
		t.Fatalf("push after clear failed: %v", err)
	}

	item, err := q.Pop(context.Background(), 0)
	if err != nil {
		t.Fatalf("pop after clear failed: %v", err)
	}
	if item != 3 {
		t.Fatalf("expected to pop new item 3 after clear, got %d", item)
	}
}

func TestQueuePushBlockingWaitsForSpace(t *testing.T) {
	q := NewQueue[int](1)
	if err := q.Push(1); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- q.PushBlocking(context.Background(), 2)
	}()
	item, err := q.Pop(context.Background(), 0)
	if err != nil || item != 1 {
		t.Fatalf("pop failed: item=%d err=%v", item, err)
	}
	if err := <-done; err != nil {
		t.Fatalf("blocking push failed: %v", err)
	}
}
