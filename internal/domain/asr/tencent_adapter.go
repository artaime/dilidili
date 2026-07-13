package asr

import (
	"context"

	"dili-esp32-server-golang/internal/domain/asr/tencent"
	asrtypes "dili-esp32-server-golang/internal/domain/asr/types"
)

type TencentAdapter struct {
	engine *tencent.ASR
}

func NewTencentAdapter(config map[string]interface{}) (AsrProvider, error) {
	cfg, err := tencent.ConfigFromMap(config)
	if err != nil {
		return nil, err
	}
	engine, err := tencent.New(cfg)
	if err != nil {
		return nil, err
	}
	return &TencentAdapter{engine: engine}, nil
}

func (a *TencentAdapter) Process(pcmData []float32) (string, error) {
	return a.engine.Process(pcmData)
}

func (a *TencentAdapter) StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan asrtypes.StreamingResult, error) {
	return a.engine.StreamingRecognize(ctx, audioStream)
}

func (a *TencentAdapter) Close() error {
	if a.engine != nil {
		return a.engine.Close()
	}
	return nil
}

func (a *TencentAdapter) IsValid() bool {
	return a != nil && a.engine != nil && a.engine.IsValid()
}
