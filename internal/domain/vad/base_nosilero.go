//go:build nosilero

package vad

import (
	"errors"

	"xiaozhi-esp32-server-golang/constants"
	"xiaozhi-esp32-server-golang/internal/domain/vad/inter"
	"xiaozhi-esp32-server-golang/internal/domain/vad/ten_vad"
)

func AcquireVAD(provider string, config map[string]interface{}) (inter.VAD, error) {
	if configProvider, ok := config["provider"].(string); ok && configProvider != "" {
		provider = configProvider
	}
	if provider == "" {
		return nil, errors.New("vad provider is empty, please set provider in config (supported: ten_vad)")
	}
	switch provider {
	case constants.VadTypeTenVad:
		return ten_vad.AcquireVAD(config)
	case constants.VadTypeSileroVad:
		return nil, errors.New("silero_vad is not compiled in this build (use build tag nosilero without silero or full CI build)")
	default:
		return nil, errors.New("invalid vad provider: " + provider + " (supported in this build: ten_vad)")
	}
}

func ReleaseVAD(vad inter.VAD) error {
	switch vad.(type) {
	case *ten_vad.TenVAD:
		return ten_vad.ReleaseVAD(vad)
	default:
		return errors.New("invalid vad type")
	}
}
