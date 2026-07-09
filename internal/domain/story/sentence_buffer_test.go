package story

import "testing"

func TestSentenceBufferAppend(t *testing.T) {
	var buf SentenceBuffer
	s1 := buf.Append("从前有座山。")
	if len(s1) != 1 || s1[0] != "从前有座山。" {
		t.Fatalf("unexpected s1: %+v", s1)
	}
	s2 := buf.Append("山里有座庙，庙里")
	if len(s2) != 0 {
		t.Fatalf("expected no complete sentence yet, got %+v", s2)
	}
	s3 := buf.Append("有个小和尚！")
	if len(s3) != 1 || s3[0] != "山里有座庙，庙里有个小和尚！" {
		t.Fatalf("unexpected s3: %+v", s3)
	}
	if tail := buf.Flush(); tail != "" {
		t.Fatalf("expected empty tail, got %q", tail)
	}
}

func TestSentenceBufferFlushRemainder(t *testing.T) {
	var buf SentenceBuffer
	if got := buf.Append("还没写完"); len(got) != 0 {
		t.Fatalf("expected no sentence")
	}
	if tail := buf.Flush(); tail != "还没写完" {
		t.Fatalf("unexpected tail: %q", tail)
	}
}
