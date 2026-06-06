package chat

import (
	"testing"

	"github.com/spf13/viper"
)

func TestTrimFirstSpeechAudioKeepsCurrentFrameAndMaxPreSpeech(t *testing.T) {
	allData := make([]float32, 1000)
	for i := range allData {
		allData[i] = float32(i)
	}

	got := trimFirstSpeechAudio(allData, 20, 1000, 1)

	if len(got) != 820 {
		t.Fatalf("len = %d, want 820", len(got))
	}
	if got[0] != 180 {
		t.Fatalf("first sample = %v, want 180", got[0])
	}
	if got[len(got)-1] != 999 {
		t.Fatalf("last sample = %v, want 999", got[len(got)-1])
	}
}

func TestTrimFirstSpeechAudioUsesConfiguredPreAudio(t *testing.T) {
	viper.Set("chat.first_speech_pre_audio_ms", 50)
	defer viper.Reset()

	allData := make([]float32, 1000)
	for i := range allData {
		allData[i] = float32(i)
	}

	got := trimFirstSpeechAudio(allData, 20, 1000, 1)

	if len(got) != 70 {
		t.Fatalf("len = %d, want 70", len(got))
	}
	if got[0] != 930 {
		t.Fatalf("first sample = %v, want 930", got[0])
	}
	if got[len(got)-1] != 999 {
		t.Fatalf("last sample = %v, want 999", got[len(got)-1])
	}
}

func TestTrimFirstSpeechAudioKeepsShortBuffer(t *testing.T) {
	allData := []float32{1, 2, 3, 4, 5}

	got := trimFirstSpeechAudio(allData, 2, 1000, 1)

	if len(got) != len(allData) {
		t.Fatalf("len = %d, want %d", len(got), len(allData))
	}
	for i := range allData {
		if got[i] != allData[i] {
			t.Fatalf("sample[%d] = %v, want %v", i, got[i], allData[i])
		}
	}
}

func TestTrimFirstSpeechAudioInvalidFormatKeepsOriginal(t *testing.T) {
	allData := []float32{1, 2, 3, 4, 5}

	got := trimFirstSpeechAudio(allData, 2, 0, 1)

	if len(got) != len(allData) {
		t.Fatalf("len = %d, want %d", len(got), len(allData))
	}
}
