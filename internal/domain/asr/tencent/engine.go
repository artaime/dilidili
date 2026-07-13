package tencent

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	asrtypes "dili-esp32-server-golang/internal/domain/asr/types"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const signatureHostPrefix = "asr.cloud.tencent.com/asr/v2/"

// ASR 腾讯云实时语音识别 WebSocket v2 实现。
type ASR struct {
	config Config
}

func New(cfg Config) (*ASR, error) {
	if cfg.AppID <= 0 || cfg.SecretID == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("tencent_asr missing required credentials")
	}
	return &ASR{config: cfg}, nil
}

func (a *ASR) StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan asrtypes.StreamingResult, error) {
	resultChan := make(chan asrtypes.StreamingResult, 20)

	voiceID := uuid.NewString()
	signedURL, err := a.buildSignedURL(voiceID)
	if err != nil {
		return nil, err
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: a.config.ConnectTimeout,
	}
	conn, _, err := dialer.DialContext(ctx, signedURL, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("tencent_asr dial failed: %w", err)
	}

	go a.handleStreaming(ctx, conn, audioStream, resultChan)
	return resultChan, nil
}

func (a *ASR) Process(pcmData []float32) (string, error) {
	ctx := context.Background()
	if a.config.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.ConnectTimeout)
		defer cancel()
	}

	audioStream := make(chan []float32, 1)
	go func() {
		audioStream <- pcmData
		close(audioStream)
	}()

	resultChan, err := a.StreamingRecognize(ctx, audioStream)
	if err != nil {
		return "", err
	}

	var finalText string
	for result := range resultChan {
		if result.Error != nil {
			return "", result.Error
		}
		if result.Text != "" {
			finalText = result.Text
		}
		if result.IsFinal {
			return finalText, nil
		}
	}
	if finalText != "" {
		return finalText, nil
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return "", fmt.Errorf("no tencent_asr result")
}

func (a *ASR) Close() error  { return nil }
func (a *ASR) IsValid() bool { return a != nil }

func (a *ASR) handleStreaming(ctx context.Context, conn *websocket.Conn, audioStream <-chan []float32, resultChan chan asrtypes.StreamingResult) {
	defer close(resultChan)
	defer conn.Close()

	if err := a.readHandshake(ctx, conn); err != nil {
		resultChan <- asrtypes.StreamingResult{Error: err, IsFinal: true}
		return
	}

	sendDone := make(chan error, 1)
	go func() {
		sendDone <- a.sendAudio(ctx, conn, audioStream)
	}()

	var (
		latestText string
		recvCount  int
		endSent    bool
		endSentAt  time.Time
	)

	readTimeout := a.config.ConnectTimeout
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	finalWaitAfterEnd := readTimeout
	if finalWaitAfterEnd > 15*time.Second {
		finalWaitAfterEnd = 15 * time.Second
	}

	for {
		if endSent && !endSentAt.IsZero() && time.Since(endSentAt) > finalWaitAfterEnd {
			finalText := latestText
			emptyReason := asrtypes.EmptyReasonNone
			if finalText == "" {
				if recvCount == 0 {
					emptyReason = asrtypes.EmptyReasonNoServerResponse
				} else {
					emptyReason = asrtypes.EmptyReasonProviderEmptyFinal
				}
			}
			resultChan <- asrtypes.StreamingResult{Text: finalText, IsFinal: true, EmptyReason: emptyReason}
			return
		}

		if a.config.ConnectTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(a.config.ConnectTimeout))
		}

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				resultChan <- asrtypes.StreamingResult{Error: ctx.Err(), IsFinal: true}
				return
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				finalText := latestText
				emptyReason := asrtypes.EmptyReasonNone
				if finalText == "" {
					if recvCount == 0 {
						emptyReason = asrtypes.EmptyReasonNoServerResponse
					} else {
						emptyReason = asrtypes.EmptyReasonProviderEmptyFinal
					}
				}
				resultChan <- asrtypes.StreamingResult{Text: finalText, IsFinal: true, EmptyReason: emptyReason}
				return
			}
			resultChan <- asrtypes.StreamingResult{Error: fmt.Errorf("tencent_asr read message failed: %w", err), IsFinal: true}
			return
		}
		if messageType != websocket.TextMessage {
			continue
		}
		recvCount++

		var msg serverMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			resultChan <- asrtypes.StreamingResult{Error: fmt.Errorf("tencent_asr decode response failed: %w", err), IsFinal: true}
			return
		}
		if msg.Code != 0 {
			resultChan <- asrtypes.StreamingResult{
				Error:       fmt.Errorf("tencent_asr error code=%d message=%s voice_id=%s", msg.Code, msg.Message, msg.VoiceID),
				IsFinal:     true,
				RetryReason: classifyTencentRetryReason(msg.Code, msg.Message),
			}
			return
		}

		text := msg.sentenceText()
		if text != "" {
			latestText = text
			if msg.isPartial() || msg.isStableSentence() {
				resultChan <- asrtypes.StreamingResult{Text: latestText, IsFinal: false}
			}
		}

		if msg.isFinal() {
			finalText := latestText
			emptyReason := asrtypes.EmptyReasonNone
			if finalText == "" {
				if recvCount == 0 {
					emptyReason = asrtypes.EmptyReasonNoServerResponse
				} else {
					emptyReason = asrtypes.EmptyReasonProviderEmptyFinal
				}
			}
			resultChan <- asrtypes.StreamingResult{Text: finalText, IsFinal: true, EmptyReason: emptyReason}
			return
		}

		select {
		case sendErr := <-sendDone:
			if sendErr != nil && ctx.Err() == nil {
				resultChan <- asrtypes.StreamingResult{Error: sendErr, IsFinal: true}
				return
			}
			if sendErr == nil {
				endSent = true
				endSentAt = time.Now()
			}
		default:
		}
	}
}

func (a *ASR) readHandshake(ctx context.Context, conn *websocket.Conn) error {
	if a.config.ConnectTimeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(a.config.ConnectTimeout))
	}
	_, message, err := conn.ReadMessage()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("tencent_asr handshake read failed: %w", err)
	}
	var msg serverMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("tencent_asr handshake decode failed: %w", err)
	}
	if msg.Code != 0 {
		return fmt.Errorf("tencent_asr handshake failed code=%d message=%s", msg.Code, msg.Message)
	}
	return nil
}

func (a *ASR) sendAudio(ctx context.Context, conn *websocket.Conn, audioStream <-chan []float32) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pcm, ok := <-audioStream:
			if !ok {
				payload, err := json.Marshal(endMessage{Type: "end"})
				if err != nil {
					return err
				}
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					return fmt.Errorf("tencent_asr send end failed: %w", err)
				}
				return nil
			}
			if len(pcm) == 0 {
				continue
			}
			audioBytes := float32ToPCMBytes(pcm)
			if err := conn.WriteMessage(websocket.BinaryMessage, audioBytes); err != nil {
				return fmt.Errorf("tencent_asr send audio failed: %w", err)
			}
		}
	}
}

func (a *ASR) buildSignedURL(voiceID string) (string, error) {
	timestamp := time.Now().Unix()
	expired := timestamp + 86400
	nonce, err := randomNonce()
	if err != nil {
		return "", err
	}

	params := map[string]string{
		"convert_num_mode":  strconv.Itoa(a.config.ConvertNumMode),
		"engine_model_type": a.config.EngineModelType,
		"expired":           strconv.FormatInt(expired, 10),
		"filter_dirty":      strconv.Itoa(a.config.FilterDirty),
		"filter_modal":      strconv.Itoa(a.config.FilterModal),
		"filter_punc":       strconv.Itoa(a.config.FilterPunc),
		"needvad":           strconv.Itoa(a.config.NeedVAD),
		"nonce":             strconv.FormatInt(nonce, 10),
		"secretid":          a.config.SecretID,
		"timestamp":         strconv.FormatInt(timestamp, 10),
		"voice_format":      strconv.Itoa(a.config.VoiceFormat),
		"voice_id":          voiceID,
	}
	if a.config.InputSampleRate > 0 {
		params["input_sample_rate"] = strconv.Itoa(a.config.InputSampleRate)
	}
	if a.config.VadSilenceTime > 0 {
		params["vad_silence_time"] = strconv.Itoa(a.config.VadSilenceTime)
	}
	if a.config.HotwordID != "" {
		params["hotword_id"] = a.config.HotwordID
	}
	if a.config.CustomizationID != "" {
		params["customization_id"] = a.config.CustomizationID
	}

	signature, err := signTencentParams(params, a.config.SecretKey, a.config.AppID)
	if err != nil {
		return "", err
	}
	params["signature"] = signature

	return "wss://" + signatureHostPrefix + strconv.FormatInt(a.config.AppID, 10) + "?" + encodeSortedParams(params), nil
}

func signTencentParams(params map[string]string, secretKey string, appID int64) (string, error) {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key == "signature" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	query := strings.Join(parts, "&")
	signText := signatureHostPrefix + strconv.FormatInt(appID, 10) + "?" + query

	mac := hmac.New(sha1.New, []byte(secretKey))
	if _, err := mac.Write([]byte(signText)); err != nil {
		return "", fmt.Errorf("tencent_asr sign failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func encodeSortedParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(params[key]))
	}
	return strings.Join(parts, "&")
}

func randomNonce() (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000_0000))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

func float32ToPCMBytes(samples []float32) []byte {
	pcmBytes := make([]byte, len(samples)*2)
	for i, sample := range samples {
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		v := int16(sample * 32767)
		binary.LittleEndian.PutUint16(pcmBytes[i*2:], uint16(v))
	}
	return pcmBytes
}

// classifyTencentRetryReason 将腾讯 ASR 错误码映射为可恢复重试原因。
// 4008：客户端超过 15 秒未发送音频（TTS 长播报时常见，不应关闭会话）。
func classifyTencentRetryReason(code int, message string) string {
	if code == 4008 {
		return asrtypes.RetryReasonTencentNoAudioTimeout
	}
	msg := strings.ToLower(strings.TrimSpace(message))
	if strings.Contains(msg, "15秒") && strings.Contains(msg, "未发送音频") {
		return asrtypes.RetryReasonTencentNoAudioTimeout
	}
	return asrtypes.RetryReasonNone
}
