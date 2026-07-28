package eventbus

import (
	. "dili-esp32-server-golang/internal/data/client"
	"time"

	"github.com/cloudwego/eino/schema"
)

// AddMessageEvent 统一的消息添加事件
type AddMessageEvent struct {
	// 客户端状态
	ClientState *ClientState

	// 消息内容（统一使用 schema.Message）
	// schema.Message 是标准的 LLM 消息格式，包含：
	// - Role: 消息角色（User/Assistant/System/Tool）
	// - Content: 消息文本内容
	// - ToolCalls: 工具调用列表（可选）
	// - ToolCallID: 工具调用ID（Tool 角色使用）
	Msg schema.Message

	// 消息ID（用于关联两阶段保存）
	MessageID string

	// 音频数据（可选，不属于 schema.Message 标准格式）
	// 聊天历史仅持久化用户 ASR 音频：用户消息可在新增时携带 AudioData；
	// AI TTS 不再写入 chat_history（IsUpdate 仅兼容旧路径，非 user 会被跳过）。
	AudioData [][]byte // ASR 音频帧数组（PCM float32 字节）
	AudioSize int      // 音频大小（字节）

	// 音频格式信息（不属于 schema.Message 标准格式）
	SampleRate int // 采样率
	Channels   int // 通道数

	// 元数据（不属于 schema.Message 标准格式）
	Timestamp   time.Time
	TTSDuration int // TTS 耗时（毫秒）

	// 阶段标识
	IsUpdate bool // true=更新音频（仅 user），false=新增消息
}
