package tencent

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"dili-esp32-server-golang/internal/data/audio"
	"dili-esp32-server-golang/internal/domain/tts/streaming"
	"dili-esp32-server-golang/internal/util"
	log "dili-esp32-server-golang/logger"

	"github.com/google/uuid"
	"github.com/gopxl/beep"
	"github.com/gorilla/websocket"
)

const (
	defaultTencentWSURL          = "wss://tts.cloud.tencent.com/stream_wsv2"
	defaultTencentVoiceType      = 101001
	defaultTencentCodec          = "pcm"
	defaultTencentSampleRate     = 16000
	defaultTencentFrameDuration  = audio.FrameDuration
	defaultTencentConnectTimeout = 10
	defaultTencentReadTimeout    = 60
	signatureHostPath            = "tts.cloud.tencent.com/stream_wsv2"
	actionTextToStreamAudioWSv2  = "TextToStreamAudioWSv2"
)

var defaultTencentDialer = websocket.Dialer{
	ReadBufferSize:   16 * 1024,
	WriteBufferSize:  16 * 1024,
	HandshakeTimeout: defaultTencentConnectTimeout * time.Second,
}

// TencentTTSProvider 腾讯云流式文本语音合成 v2 提供者。
type TencentTTSProvider struct {
	AppID           int64
	SecretID        string
	SecretKey       string
	VoiceType       int
	Codec           string
	SampleRate      int
	Speed           float64
	Volume          float64
	WSURL           string
	FrameDuration   int
	ConnectTimeout  int
	ReadTimeout     int
	EnableSubtitle  bool
	EmotionCategory string
	EmotionIntensity int

	connMu sync.Mutex
	conn   *websocket.Conn
}

type tencentServerMessage struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
	RequestID string `json:"request_id"`
	MessageID string `json:"message_id"`
	Ready     int    `json:"ready"`
	Final     int    `json:"final"`
	Heartbeat int    `json:"heartbeat"`
	Result    *struct {
		Subtitles []tencentSubtitle `json:"subtitles"`
	} `json:"result"`
}

type tencentSubtitle struct {
	Text       string `json:"Text"`
	BeginTime  int    `json:"BeginTime"`
	EndTime    int    `json:"EndTime"`
	BeginIndex int    `json:"BeginIndex"`
	EndIndex   int    `json:"EndIndex"`
}

type tencentClientMessage struct {
	SessionID string `json:"session_id"`
	MessageID string `json:"message_id"`
	Action    string `json:"action"`
	Data      string `json:"data"`
}

// NewTencentTTSProvider 从配置 map 创建腾讯 TTS 提供者。
func NewTencentTTSProvider(config map[string]interface{}) *TencentTTSProvider {
	p := &TencentTTSProvider{
		AppID:           parseInt64Config(config, "app_id"),
		SecretID:        strings.TrimSpace(stringConfig(config, "secret_id")),
		SecretKey:       strings.TrimSpace(stringConfig(config, "secret_key")),
		VoiceType:       parseVoiceType(config),
		Codec:           strings.ToLower(strings.TrimSpace(stringConfig(config, "codec"))),
		SampleRate:      parseIntConfig(config, "sample_rate", defaultTencentSampleRate),
		Speed:           parseFloatConfig(config, "speed", 0),
		Volume:          parseFloatConfig(config, "volume", 0),
		WSURL:           strings.TrimSpace(stringConfig(config, "ws_url")),
		FrameDuration:   parseIntConfig(config, "frame_duration", defaultTencentFrameDuration),
		ConnectTimeout:  parseIntConfig(config, "connect_timeout", defaultTencentConnectTimeout),
		ReadTimeout:     parseIntConfig(config, "read_timeout", defaultTencentReadTimeout),
		EnableSubtitle:  parseBoolConfig(config, "enable_subtitle"),
		EmotionCategory: strings.TrimSpace(stringConfig(config, "emotion_category")),
		EmotionIntensity: parseIntConfig(config, "emotion_intensity", 100),
	}
	if p.Codec == "" {
		p.Codec = defaultTencentCodec
	}
	if p.WSURL == "" {
		p.WSURL = defaultTencentWSURL
	}
	if p.VoiceType <= 0 {
		p.VoiceType = defaultTencentVoiceType
	}
	return p
}

func (p *TencentTTSProvider) validate() error {
	if p.AppID <= 0 {
		return fmt.Errorf("tencent_tts 未配置 app_id")
	}
	if p.SecretID == "" {
		return fmt.Errorf("tencent_tts 未配置 secret_id")
	}
	if p.SecretKey == "" {
		return fmt.Errorf("tencent_tts 未配置 secret_key")
	}
	if p.VoiceType <= 0 {
		return fmt.Errorf("tencent_tts 未配置 voice_type")
	}
	switch p.Codec {
	case "pcm", "mp3":
	default:
		return fmt.Errorf("tencent_tts 不支持的 codec: %s", p.Codec)
	}
	switch p.SampleRate {
	case 8000, 16000, 24000:
	default:
		return fmt.Errorf("tencent_tts 不支持的 sample_rate: %d", p.SampleRate)
	}
	return nil
}

func (p *TencentTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	ch, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}
	var frames [][]byte
	for frame := range ch {
		frameCopy := make([]byte, len(frame))
		copy(frameCopy, frame)
		frames = append(frames, frameCopy)
	}
	return frames, nil
}

func (p *TencentTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		out := make(chan []byte)
		close(out)
		return out, nil
	}

	targetSampleRate := sampleRate
	if targetSampleRate <= 0 {
		targetSampleRate = p.SampleRate
	}
	targetFrameDuration := frameDuration
	if targetFrameDuration <= 0 {
		targetFrameDuration = p.FrameDuration
	}

	outputChan := make(chan []byte, 100)
	startTs := time.Now().UnixMilli()
	go func() {
		if err := p.streamSynthesis(ctx, text, targetSampleRate, targetFrameDuration, startTs, outputChan); err != nil && ctx.Err() == nil {
			log.Errorf("tencent_tts 流式合成失败: %v", err)
		}
	}()
	return outputChan, nil
}

func (p *TencentTTSProvider) StreamingSynthesize(ctx context.Context, textChan <-chan string, sampleRate int, channels int, frameDuration int) (chan streaming.SynthesisEvent, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	targetSampleRate := sampleRate
	if targetSampleRate <= 0 {
		targetSampleRate = p.SampleRate
	}
	targetFrameDuration := frameDuration
	if targetFrameDuration <= 0 {
		targetFrameDuration = p.FrameDuration
	}

	outputChan := make(chan streaming.SynthesisEvent, 100)
	startTs := time.Now().UnixMilli()
	go func() {
		if err := p.streamingSynthesisLoop(ctx, textChan, targetSampleRate, targetFrameDuration, startTs, outputChan); err != nil && ctx.Err() == nil {
			log.Errorf("tencent_tts 双流式合成失败: %v", err)
		}
	}()
	return outputChan, nil
}

func (p *TencentTTSProvider) streamSynthesis(ctx context.Context, text string, targetSampleRate int, frameDuration int, startTs int64, outputChan chan []byte) error {
	sessionID := uuid.NewString()
	conn, err := p.dial(ctx, sessionID)
	if err != nil {
		close(outputChan)
		return err
	}
	defer conn.Close()

	if err := p.waitReady(ctx, conn); err != nil {
		close(outputChan)
		return err
	}

	pipeReader, pipeWriter := io.Pipe()
	decoderDone := make(chan struct{})
	go func() {
		defer close(decoderDone)
		if err := p.runAudioDecoder(ctx, pipeReader, outputChan, frameDuration, targetSampleRate, startTs); err != nil && ctx.Err() == nil {
			log.Errorf("tencent_tts 音频解码失败: %v", err)
		}
	}()

	readDone := make(chan error, 1)
	go func() {
		readDone <- p.readAudioUntilFinal(ctx, conn, pipeWriter, nil, 0)
	}()

	if err := p.sendAction(conn, sessionID, "ACTION_SYNTHESIS", ensureSentencePunctuation(text)); err != nil {
		_ = pipeWriter.CloseWithError(err)
		<-readDone
		<-decoderDone
		return err
	}
	if err := p.sendAction(conn, sessionID, "ACTION_COMPLETE", ""); err != nil {
		_ = pipeWriter.CloseWithError(err)
		<-readDone
		<-decoderDone
		return err
	}

	readErr := <-readDone
	if readErr != nil {
		_ = pipeWriter.CloseWithError(readErr)
	} else {
		_ = pipeWriter.Close()
	}
	<-decoderDone
	if readErr == nil && ctx.Err() == nil {
		log.Infof("tencent_tts 耗时: 从输入至获取音频数据结束耗时: %d ms", time.Now().UnixMilli()-startTs)
	}
	return readErr
}

func (p *TencentTTSProvider) streamingSynthesisLoop(ctx context.Context, textChan <-chan string, targetSampleRate int, frameDuration int, startTs int64, outputChan chan streaming.SynthesisEvent) error {
	defer close(outputChan)

	sessionID := uuid.NewString()
	conn, err := p.dial(ctx, sessionID)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := p.waitReady(ctx, conn); err != nil {
		return err
	}

	pipeReader, pipeWriter := io.Pipe()
	audioFrameChan := make(chan []byte, 100)
	mergeDone := make(chan struct{})
	go func() {
		defer close(mergeDone)
		for frame := range audioFrameChan {
			frameCopy := make([]byte, len(frame))
			copy(frameCopy, frame)
			select {
			case <-ctx.Done():
				return
			case outputChan <- streaming.SynthesisEvent{Audio: frameCopy}:
			}
		}
	}()

	decoderDone := make(chan struct{})
	go func() {
		defer close(decoderDone)
		if err := p.runAudioDecoder(ctx, pipeReader, audioFrameChan, frameDuration, targetSampleRate, startTs); err != nil && ctx.Err() == nil {
			log.Errorf("tencent_tts 双流式音频解码失败: %v", err)
		}
	}()

	readDone := make(chan error, 1)
	go func() {
		// 双流式需等待 LLM 逐句送文本，禁用读超时避免句间停顿导致连接被误判断开。
		readDone <- p.readAudioUntilFinal(ctx, conn, pipeWriter, nil, -1)
	}()

	textDone := make(chan struct{})
	go func() {
		defer close(textDone)
		for {
			select {
			case <-ctx.Done():
				return
			case chunk, ok := <-textChan:
				if !ok {
					if err := p.sendAction(conn, sessionID, "ACTION_COMPLETE", ""); err != nil && ctx.Err() == nil {
						log.Errorf("tencent_tts 发送 ACTION_COMPLETE 失败: %v", err)
					}
					return
				}
				chunk = strings.TrimSpace(chunk)
				if chunk == "" {
					continue
				}
				if err := p.sendAction(conn, sessionID, "ACTION_SYNTHESIS", ensureSentencePunctuation(chunk)); err != nil {
					if ctx.Err() == nil {
						log.Errorf("tencent_tts 发送 ACTION_SYNTHESIS 失败: %v", err)
					}
					return
				}
			}
		}
	}()

	<-textDone
	readErr := <-readDone
	if readErr != nil {
		_ = pipeWriter.CloseWithError(readErr)
	} else {
		_ = pipeWriter.Close()
	}
	<-decoderDone
	<-mergeDone
	return readErr
}

func (p *TencentTTSProvider) runAudioDecoder(ctx context.Context, pipeReader io.ReadCloser, outputChan chan []byte, frameDuration int, targetSampleRate int, startTs int64) error {
	audioFormat := p.Codec
	if audioFormat == "pcm" {
		audioFormat = "pcm"
	}
	decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, audioFormat, targetSampleRate)
	if err != nil {
		return fmt.Errorf("创建 tencent_tts 音频解码器失败: %v", err)
	}
	if audioFormat == "pcm" {
		decoder.WithFormat(beep.Format{
			SampleRate:  beep.SampleRate(p.SampleRate),
			NumChannels: 1,
		})
	}
	return decoder.Run(startTs)
}

func (p *TencentTTSProvider) dial(ctx context.Context, sessionID string) (*websocket.Conn, error) {
	signedURL, err := p.buildSignedURL(sessionID)
	if err != nil {
		return nil, err
	}

	dialer := defaultTencentDialer
	if p.ConnectTimeout > 0 {
		dialer.HandshakeTimeout = time.Duration(p.ConnectTimeout) * time.Second
	}

	header := http.Header{}
	conn, resp, err := dialer.DialContext(ctx, signedURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("连接腾讯 TTS WebSocket 失败: %v, status=%d body=%s", err, resp.StatusCode, previewString(string(body), 300))
		}
		return nil, fmt.Errorf("连接腾讯 TTS WebSocket 失败: %v", err)
	}

	if err := p.readHandshakeSuccess(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func (p *TencentTTSProvider) buildSignedURL(sessionID string) (string, error) {
	timestamp := time.Now().Unix()
	expired := timestamp + 86400

	params := map[string]string{
		"Action":    actionTextToStreamAudioWSv2,
		"AppId":     strconv.FormatInt(p.AppID, 10),
		"Codec":     p.Codec,
		"Expired":   strconv.FormatInt(expired, 10),
		"SampleRate": strconv.Itoa(p.SampleRate),
		"SecretId":  p.SecretID,
		"SessionId": sessionID,
		"Speed":     formatFloatParam(p.Speed),
		"Timestamp": strconv.FormatInt(timestamp, 10),
		"VoiceType": strconv.Itoa(p.VoiceType),
		"Volume":    formatFloatParam(p.Volume),
	}
	if p.EnableSubtitle {
		params["EnableSubtitle"] = "True"
	}
	if p.EmotionCategory != "" {
		params["EmotionCategory"] = p.EmotionCategory
		if p.EmotionIntensity > 0 {
			params["EmotionIntensity"] = strconv.Itoa(p.EmotionIntensity)
		}
	}

	signature, err := signTencentParams(params, p.SecretKey)
	if err != nil {
		return "", err
	}
	params["Signature"] = signature

	query := encodeSortedParams(params)
	baseURL := strings.TrimSpace(p.WSURL)
	if baseURL == "" {
		baseURL = defaultTencentWSURL
	}
	if strings.Contains(baseURL, "?") {
		return baseURL + "&" + query, nil
	}
	return baseURL + "?" + query, nil
}

func signTencentParams(params map[string]string, secretKey string) (string, error) {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key == "Signature" {
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
	signText := "GET" + signatureHostPath + "?" + query

	mac := hmac.New(sha1.New, []byte(secretKey))
	if _, err := mac.Write([]byte(signText)); err != nil {
		return "", fmt.Errorf("生成腾讯 TTS 签名失败: %v", err)
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

func (p *TencentTTSProvider) readHandshakeSuccess(ctx context.Context, conn *websocket.Conn) error {
	msg, err := p.readServerMessage(ctx, conn)
	if err != nil {
		return err
	}
	if msg.Code != 0 {
		return fmt.Errorf("腾讯 TTS 握手失败 [%d]: %s", msg.Code, strings.TrimSpace(msg.Message))
	}
	return nil
}

func (p *TencentTTSProvider) waitReady(ctx context.Context, conn *websocket.Conn) error {
	for {
		msg, err := p.readServerMessage(ctx, conn)
		if err != nil {
			return err
		}
		if msg.Code != 0 {
			return fmt.Errorf("腾讯 TTS 错误 [%d]: %s", msg.Code, strings.TrimSpace(msg.Message))
		}
		if msg.Ready == 1 {
			return nil
		}
		if msg.Final == 1 {
			return fmt.Errorf("腾讯 TTS 在 READY 前收到 FINAL 事件")
		}
	}
}

func (p *TencentTTSProvider) readAudioUntilFinal(ctx context.Context, conn *websocket.Conn, pipeWriter *io.PipeWriter, onText func(tencentServerMessage), readTimeoutSec int) error {
	effectiveReadTimeout := p.ReadTimeout
	if readTimeoutSec > 0 {
		effectiveReadTimeout = readTimeoutSec
	} else if readTimeoutSec < 0 {
		effectiveReadTimeout = 0
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if effectiveReadTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(time.Duration(effectiveReadTimeout) * time.Second))
		}

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("读取腾讯 TTS WebSocket 消息失败: %v", err)
		}

		switch messageType {
		case websocket.BinaryMessage:
			if len(message) == 0 {
				continue
			}
			if _, err := pipeWriter.Write(message); err != nil {
				return fmt.Errorf("写入腾讯 TTS 音频数据失败: %v", err)
			}
		case websocket.TextMessage:
			var msg tencentServerMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				return fmt.Errorf("解析腾讯 TTS 响应失败: %v, body=%s", err, previewString(string(message), 300))
			}
			if onText != nil {
				onText(msg)
			}
			if msg.Code != 0 {
				return fmt.Errorf("腾讯 TTS 错误 [%d]: %s", msg.Code, strings.TrimSpace(msg.Message))
			}
			if msg.Heartbeat == 1 {
				continue
			}
			if msg.Final == 1 {
				return nil
			}
		default:
			continue
		}
	}
}

func (p *TencentTTSProvider) readServerMessage(ctx context.Context, conn *websocket.Conn) (tencentServerMessage, error) {
	var msg tencentServerMessage
	for {
		select {
		case <-ctx.Done():
			return msg, ctx.Err()
		default:
		}
		if p.ReadTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(time.Duration(p.ReadTimeout) * time.Second))
		}
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return msg, ctx.Err()
			}
			return msg, fmt.Errorf("读取腾讯 TTS 消息失败: %v", err)
		}
		if messageType != websocket.TextMessage {
			continue
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			return msg, fmt.Errorf("解析腾讯 TTS 消息失败: %v", err)
		}
		return msg, nil
	}
}

func (p *TencentTTSProvider) sendAction(conn *websocket.Conn, sessionID, action, data string) error {
	payload, err := json.Marshal(tencentClientMessage{
		SessionID: sessionID,
		MessageID: uuid.NewString(),
		Action:    action,
		Data:      data,
	})
	if err != nil {
		return fmt.Errorf("序列化腾讯 TTS 指令失败: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		return fmt.Errorf("发送腾讯 TTS 指令失败: %v", err)
	}
	return nil
}

func (p *TencentTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voiceType := parseVoiceType(voiceConfig); voiceType > 0 {
		p.VoiceType = voiceType
		return nil
	}
	if voice := strings.TrimSpace(stringConfig(voiceConfig, "voice")); voice != "" {
		if parsed, err := strconv.Atoi(voice); err == nil && parsed > 0 {
			p.VoiceType = parsed
			return nil
		}
	}
	return fmt.Errorf("无效的音色配置: 缺少 voice_type")
}

func (p *TencentTTSProvider) Close() error {
	p.connMu.Lock()
	defer p.connMu.Unlock()
	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		return err
	}
	return nil
}

func (p *TencentTTSProvider) IsValid() bool {
	return p != nil
}

func ensureSentencePunctuation(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	last := []rune(text)[len([]rune(text))-1]
	switch last {
	case '。', '！', '？', '；', '.', '!', '?', ';':
		return text
	default:
		return text + "。"
	}
}

func previewString(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func stringConfig(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	value, ok := config[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func parseIntConfig(config map[string]interface{}, key string, defaultValue int) int {
	if config == nil {
		return defaultValue
	}
	value, ok := config[key]
	if !ok || value == nil {
		return defaultValue
	}
	switch v := value.(type) {
	case int:
		if v != 0 || key == "speed" || key == "volume" {
			return v
		}
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseInt64Config(config map[string]interface{}, key string) int64 {
	if config == nil {
		return 0
	}
	value, ok := config[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

func parseFloatConfig(config map[string]interface{}, key string, defaultValue float64) float64 {
	if config == nil {
		return defaultValue
	}
	value, ok := config[key]
	if !ok || value == nil {
		return defaultValue
	}
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseBoolConfig(config map[string]interface{}, key string) bool {
	if config == nil {
		return false
	}
	value, ok := config[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	case float64:
		return v != 0
	case int:
		return v != 0
	}
	return false
}

func parseVoiceType(config map[string]interface{}) int {
	if config == nil {
		return 0
	}
	if voiceType := parseIntConfig(config, "voice_type", 0); voiceType > 0 {
		return voiceType
	}
	if voice := strings.TrimSpace(stringConfig(config, "voice")); voice != "" {
		if parsed, err := strconv.Atoi(voice); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func formatFloatParam(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
