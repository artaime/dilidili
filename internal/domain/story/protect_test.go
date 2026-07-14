package story

import "testing"

func TestShouldCancelGenerationOnInterrupt(t *testing.T) {
	if !ShouldCancelGenerationOnInterrupt(0, 300) {
		t.Fatal("0 heard should cancel")
	}
	if !ShouldCancelGenerationOnInterrupt(299, 300) {
		t.Fatal("299 should cancel")
	}
	if ShouldCancelGenerationOnInterrupt(300, 300) {
		t.Fatal("300 should protect")
	}
	if ShouldCancelGenerationOnInterrupt(500, 0) {
		t.Fatal("default threshold: 500 should protect")
	}
	if !ShouldCancelGenerationOnInterrupt(100, 0) {
		t.Fatal("default threshold: 100 should cancel")
	}
}
