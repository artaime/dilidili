package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// 用户模型
type User struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	Username   string    `json:"username" gorm:"type:varchar(50);uniqueIndex:idx_users_username;not null"`
	Password   string    `json:"-" gorm:"type:varchar(255);not null"`
	Email      string    `json:"email" gorm:"type:varchar(100);uniqueIndex:idx_users_email"`
	Role       string    `json:"role" gorm:"type:varchar(20);not null;default:'user'"` // admin, user
	WxOpenid   *string   `json:"wx_openid,omitempty" gorm:"type:varchar(64);uniqueIndex:idx_users_wx_openid"`
	WxUnionid  *string   `json:"wx_unionid,omitempty" gorm:"type:varchar(64)"`
	Nickname   string    `json:"nickname,omitempty" gorm:"type:varchar(100)"`
	AvatarURL  string    `json:"avatar_url,omitempty" gorm:"type:varchar(512)"`
	Phone      string    `json:"phone,omitempty" gorm:"type:varchar(20)"`
	FamilyRole string    `json:"family_role,omitempty" gorm:"type:varchar(20);default:'其他'"` // 爸爸, 妈妈, 爷爷, 奶奶, 外公, 外婆, 其他
	Source     string    `json:"source,omitempty" gorm:"type:varchar(20);default:'web'"`     // web, miniprogram
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// APIToken 对外OpenAPI访问令牌（仅保存哈希，不保存明文）
type APIToken struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	UserID      uint       `json:"user_id" gorm:"not null;index"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	TokenPrefix string     `json:"token_prefix" gorm:"type:varchar(20);index"`
	TokenHash   string     `json:"-" gorm:"type:char(64);uniqueIndex;not null"`
	IsActive    bool       `json:"is_active" gorm:"default:true;index"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at" gorm:"index"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// 设备模型
type Device struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	UserID       uint       `json:"user_id" gorm:"not null"`
	AgentID      uint       `json:"agent_id" gorm:"not null;default:0"`                                       // 智能体ID，一台设备只能属于一个智能体
	RoleID       *uint      `json:"role_id" gorm:"index"`                                                     // 角色ID（可选，覆盖智能体配置）
	NickName     string     `json:"nick_name" gorm:"type:varchar(100)"`                                       // 设备昵称，用户可修改
	DeviceCode   string     `json:"device_code" gorm:"type:varchar(100);"`                                    // 6位激活码
	DeviceName   string     `json:"device_name" gorm:"type:varchar(100);uniqueIndex:idx_devices_device_name"` // 设备 SN / Device-ID，设备端上报使用
	Challenge    string     `json:"challenge" gorm:"type:varchar(128)"`                                       // 激活挑战码
	PreSecretKey string     `json:"pre_secret_key" gorm:"type:varchar(128)"`                                  // 预激活密钥
	Activated    bool       `json:"activated" gorm:"default:false"`                                           // 设备是否已激活
	LastActiveAt *time.Time `json:"last_active_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// DeviceMember 设备家庭成员（含属主行）
type DeviceMember struct {
	ID        uint       `json:"id" gorm:"primarykey"`
	DeviceID  uint       `json:"device_id" gorm:"not null;uniqueIndex:idx_device_member_unique,priority:1;index"`
	UserID    uint       `json:"user_id" gorm:"not null;uniqueIndex:idx_device_member_unique,priority:2;index"`
	Role      string     `json:"role" gorm:"type:varchar(20);not null;default:'member'"`   // owner, member
	Status    string     `json:"status" gorm:"type:varchar(20);not null;default:'active'"` // active, revoked
	InvitedBy *uint      `json:"invited_by,omitempty" gorm:"index"`
	JoinedAt  time.Time  `json:"joined_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (DeviceMember) TableName() string {
	return "device_members"
}

// DeviceInvite 设备家庭成员邀请码
type DeviceInvite struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	DeviceID  uint      `json:"device_id" gorm:"not null;index"`
	Code      string    `json:"code" gorm:"type:varchar(16);not null;uniqueIndex"`
	CreatedBy uint      `json:"created_by" gorm:"not null;index"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	MaxUses   int       `json:"max_uses" gorm:"not null;default:5"`
	UsedCount int       `json:"used_count" gorm:"not null;default:0"`
	Status    string    `json:"status" gorm:"type:varchar(20);not null;default:'active';index"` // active, exhausted, revoked, expired
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DeviceInvite) TableName() string {
	return "device_invites"
}

// 智能体模型
type Agent struct {
	ID              uint    `json:"id" gorm:"primarykey"`
	UserID          uint    `json:"user_id" gorm:"not null"`
	Name            string  `json:"name" gorm:"type:varchar(100);not null"`                  // 名称，管理侧识别用
	Nickname        string  `json:"nickname" gorm:"type:varchar(100)"`                       // 昵称，给大模型/Prompt 使用
	CustomPrompt    string  `json:"custom_prompt" gorm:"type:text"`                          // 角色介绍(prompt)
	LLMConfigID     *string `json:"llm_config_id" gorm:"type:varchar(100)"`                  // 语言模型配置ID
	TTSConfigID     *string `json:"tts_config_id" gorm:"type:varchar(100)"`                  // 音色配置ID
	Voice           *string `json:"voice" gorm:"type:varchar(200)"`                          // 音色值
	ASRSpeed        string  `json:"asr_speed" gorm:"type:varchar(20);default:'normal'"`      // 语音识别速度: normal/patient/fast
	MemoryMode      string  `json:"memory_mode" gorm:"type:varchar(20);default:'short'"`     // 记忆模式: none/short/long
	SpeakerChatMode string  `json:"speaker_chat_mode" gorm:"type:varchar(32);default:'off'"` // 声纹聊天模式: off/identified_only
	MCPServiceNames string  `json:"mcp_service_names" gorm:"type:text"`                      // 逗号分隔的MCP服务名，空=使用全部已启用全局MCP服务
	// OpenClaw 配置，JSON字符串，结构：
	// {"allowed":true,"enter_keywords":["进入openclaw"],"exit_keywords":["退出openclaw"]}
	OpenClawConfig string    `json:"openclaw_config" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// KnowledgeBase 用户知识库（每用户独立）
type KnowledgeBase struct {
	ID                 uint       `json:"id" gorm:"primarykey"`
	UserID             uint       `json:"user_id" gorm:"not null;index"`
	Name               string     `json:"name" gorm:"type:varchar(100);not null"`
	Description        string     `json:"description" gorm:"type:text"`
	Content            string     `json:"content" gorm:"type:text"`
	RetrievalThreshold *float64   `json:"retrieval_threshold" gorm:"type:double"`         // 检索阈值（为空表示继承全局配置）
	ExternalKBID       string     `json:"external_kb_id" gorm:"type:varchar(255);index"`  // 外部知识库ID（Dify dataset_id）
	ExternalDocID      string     `json:"external_doc_id" gorm:"type:varchar(255);index"` // 外部文档ID（Dify document_id）
	AutoDataset        bool       `json:"auto_dataset" gorm:"default:false"`              // 是否由系统自动创建dataset
	SyncProvider       string     `json:"sync_provider" gorm:"type:varchar(50);index"`    // 同步provider（当前为dify）
	SyncStatus         string     `json:"sync_status" gorm:"type:varchar(20);default:'pending';index"`
	SyncError          string     `json:"sync_error" gorm:"type:text"`
	LastSyncedAt       *time.Time `json:"last_synced_at"`
	Status             string     `json:"status" gorm:"type:varchar(20);default:'active';index"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// KnowledgeBaseDocument 知识库文档（一个知识库可包含多个文档）
type KnowledgeBaseDocument struct {
	ID              uint       `json:"id" gorm:"primarykey"`
	KnowledgeBaseID uint       `json:"knowledge_base_id" gorm:"not null;index"`
	Name            string     `json:"name" gorm:"type:varchar(200);not null"`
	Content         string     `json:"content" gorm:"type:text"`
	ExternalDocID   string     `json:"external_doc_id" gorm:"type:varchar(255);index"` // Dify document_id
	SyncStatus      string     `json:"sync_status" gorm:"type:varchar(20);default:'pending';index"`
	SyncError       string     `json:"sync_error" gorm:"type:text"`
	LastSyncedAt    *time.Time `json:"last_synced_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AgentKnowledgeBase 智能体与知识库的多对多关联
type AgentKnowledgeBase struct {
	ID              uint      `json:"id" gorm:"primarykey"`
	AgentID         uint      `json:"agent_id" gorm:"not null;index;uniqueIndex:idx_agent_kb_unique,priority:1"`
	KnowledgeBaseID uint      `json:"knowledge_base_id" gorm:"not null;index;uniqueIndex:idx_agent_kb_unique,priority:2"`
	CreatedAt       time.Time `json:"created_at"`
}

// 通用配置模型
type Config struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Type      string    `json:"type" gorm:"type:varchar(50);not null;uniqueIndex:type_config_id,priority:1"` // vad, asr, llm, tts, ota, mqtt, udp, mqtt_server, vision
	Name      string    `json:"name" gorm:"type:varchar(100);not null"`
	ConfigID  string    `json:"config_id" gorm:"type:varchar(100);not null;uniqueIndex:type_config_id,priority:2"` // 配置ID，用于关联
	Provider  string    `json:"provider" gorm:"type:varchar(50)"`                                                  // 某些配置类型需要provider字段
	JsonData  string    `json:"json_data" gorm:"type:text"`                                                        // JSON配置数据
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MCPMarketService 市场导入的MCP服务配置
// 人工配置仍存放在 Config(type=mcp).json_data 中，市场配置拆分到独立表。
type MCPMarketService struct {
	ID               uint   `json:"id" gorm:"primarykey"`
	Name             string `json:"name" gorm:"type:varchar(150);not null"`
	Enabled          bool   `json:"enabled" gorm:"default:true;index"`
	Transport        string `json:"transport" gorm:"type:varchar(32);not null"` // sse / streamablehttp
	URL              string `json:"url" gorm:"type:text;not null"`
	URLHash          string `json:"url_hash" gorm:"type:char(64);not null;uniqueIndex:idx_mcp_market_services_url_hash"` // sha256(url) hex
	HeadersJSON      string `json:"headers_json" gorm:"type:text"`
	AllowedToolsJSON string `json:"allowed_tools_json" gorm:"type:text"`

	MarketID    *uint  `json:"market_id" gorm:"index"` // 关联 configs(type=mcp_market).id
	ProviderID  string `json:"provider_id" gorm:"type:varchar(50);index"`
	ServiceID   string `json:"service_id" gorm:"type:varchar(255);index"`
	ServiceName string `json:"service_name" gorm:"type:varchar(255)"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Role 角色模型（统一管理全局角色和用户角色）
type Role struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	UserID      *uint  `json:"user_id" gorm:"index"` // 所属用户ID，NULL表示全局角色
	Name        string `json:"name" gorm:"type:varchar(100);not null"`
	Description string `json:"description" gorm:"type:text"`
	Prompt      string `json:"prompt" gorm:"type:text"` // 系统提示词

	// LLM/TTS 配置（与 Agent 字段保持一致）
	LLMConfigID *string `json:"llm_config_id" gorm:"type:varchar(100)"` // LLM配置ID

	TTSConfigID *string `json:"tts_config_id" gorm:"type:varchar(100)"` // TTS配置ID
	Voice       *string `json:"voice" gorm:"type:varchar(200)"`         // 音色值

	// 角色类型和状态
	RoleType string `json:"role_type" gorm:"type:varchar(20);default:'user';index"` // global/system/user
	Status   string `json:"status" gorm:"type:varchar(20);default:'active';index"`  // active/inactive

	// 排序和默认
	SortOrder int  `json:"sort_order" gorm:"default:0"`           // 显示排序
	IsDefault bool `json:"is_default" gorm:"default:false;index"` // 是否默认角色（仅全局角色）

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Role) TableName() string {
	return "roles"
}

// 全局角色模型（保留兼容，后续可迁移至 Role）
type GlobalRole struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`
	Description string    `json:"description" gorm:"type:text"`
	Prompt      string    `json:"prompt" gorm:"type:text"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// 声纹组模型
type SpeakerGroup struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	UserID      uint      `json:"user_id" gorm:"not null;index;uniqueIndex:idx_speaker_groups_user_name,priority:1"`
	AgentID     uint      `json:"agent_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null;uniqueIndex:idx_speaker_groups_user_name,priority:2"`
	Prompt      string    `json:"prompt" gorm:"type:text"`
	Description string    `json:"description" gorm:"type:text"`
	TTSConfigID *string   `json:"tts_config_id" gorm:"type:varchar(100)"` // TTS配置ID
	Voice       *string   `json:"voice" gorm:"type:varchar(200)"`         // 音色值
	Status      string    `json:"status" gorm:"type:varchar(20);default:'active'"`
	SampleCount int       `json:"sample_count" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// 声纹样本模型
type SpeakerSample struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	SpeakerGroupID uint      `json:"speaker_group_id" gorm:"not null;index"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	UUID           string    `json:"uuid" gorm:"type:varchar(36);not null;uniqueIndex"`
	FilePath       string    `json:"file_path" gorm:"type:varchar(500);not null"`
	FileName       string    `json:"file_name" gorm:"type:varchar(255)"`
	FileSize       int64     `json:"file_size"`
	Duration       float32   `json:"duration"`
	Status         string    `json:"status" gorm:"type:varchar(20);default:'active'"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// VoiceClone 复刻音色模型
type VoiceClone struct {
	ID                 uint      `json:"id" gorm:"primarykey"`
	UserID             uint      `json:"user_id" gorm:"not null;index"`
	Name               string    `json:"name" gorm:"type:varchar(100);not null"`
	Provider           string    `json:"provider" gorm:"type:varchar(50);not null;index"`
	ProviderVoiceID    string    `json:"provider_voice_id" gorm:"type:varchar(200);not null;index"`
	TTSConfigID        string    `json:"tts_config_id" gorm:"type:varchar(100);not null;index"`
	SharedToAll        bool      `json:"shared_to_all" gorm:"default:false;index"`
	Status             string    `json:"status" gorm:"type:varchar(20);default:'active';index"`
	TranscriptRequired bool      `json:"transcript_required" gorm:"default:false"`
	MetaJSON           string    `json:"meta_json" gorm:"type:json"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// VoiceCloneAudio 复刻原始音频资产模型（保留上传/录音数据）
type VoiceCloneAudio struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	VoiceCloneID   *uint     `json:"voice_clone_id" gorm:"index"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	SourceType     string    `json:"source_type" gorm:"type:varchar(20);not null"` // upload/record
	FilePath       string    `json:"file_path" gorm:"type:varchar(500);not null"`
	FileName       string    `json:"file_name" gorm:"type:varchar(255)"`
	FileSize       int64     `json:"file_size"`
	ContentType    string    `json:"content_type" gorm:"type:varchar(100)"`
	Transcript     string    `json:"transcript" gorm:"type:text"`
	TranscriptLang string    `json:"transcript_lang" gorm:"type:varchar(20)"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// VoiceCloneTask 声音复刻异步任务模型
type VoiceCloneTask struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	TaskID       string     `json:"task_id" gorm:"type:varchar(64);not null;uniqueIndex"`
	UserID       uint       `json:"user_id" gorm:"not null;index"`
	VoiceCloneID uint       `json:"voice_clone_id" gorm:"not null;index"`
	Provider     string     `json:"provider" gorm:"type:varchar(50);not null;index"`
	Status       string     `json:"status" gorm:"type:varchar(20);not null;default:'queued';index"` // queued/processing/succeeded/failed
	Attempts     int        `json:"attempts" gorm:"not null;default:0"`
	LastError    string     `json:"last_error" gorm:"type:text"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	MetaJSON     string     `json:"meta_json" gorm:"type:json"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// UserVoiceCloneQuota 用户声音复刻额度（按 tts_config_id 维度）
type UserVoiceCloneQuota struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	UserID      uint      `json:"user_id" gorm:"not null;index;uniqueIndex:idx_user_tts_quota,priority:1"`
	TTSConfigID string    `json:"tts_config_id" gorm:"type:varchar(100);not null;index;uniqueIndex:idx_user_tts_quota,priority:2"`
	MaxCount    int       `json:"max_count" gorm:"not null;default:-1"` // -1 表示不限制，0 表示禁止创建
	UsedCount   int       `json:"used_count" gorm:"not null;default:0"` // 每次提交复刻任务即计数
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ChatMessage 聊天消息模型
type ChatMessage struct {
	ID        uint   `json:"id" gorm:"primarykey"`
	MessageID string `json:"message_id" gorm:"type:varchar(64);uniqueIndex:idx_chat_messages_message_id;not null"`

	// 关联信息（不使用外键）
	DeviceID  string `json:"device_id" gorm:"type:varchar(100);index:idx_device_id;not null"`
	AgentID   string `json:"agent_id" gorm:"type:varchar(64);index:idx_agent_id;not null"`
	UserID    uint   `json:"user_id" gorm:"index:idx_user_id;not null"`
	SessionID string `json:"session_id" gorm:"type:varchar(64);index:idx_session_id"` // 仅作分组标记

	// 消息内容
	Role    string `json:"role" gorm:"type:varchar(20);index;not null;comment:user|assistant|system|tool"`
	Content string `json:"content" gorm:"type:text;not null"`

	// 工具调用信息
	ToolCallID    string  `json:"tool_call_id,omitempty" gorm:"type:varchar(64);index;comment:工具调用ID（Tool角色使用）"`
	ToolCallsJSON *string `json:"tool_calls_json,omitempty" gorm:"type:json;column:tool_calls;comment:工具调用列表JSON（Assistant角色使用）"`

	// 音频文件信息 (文件系统存储，两级hash打散)
	AudioPath     string `json:"audio_path,omitempty" gorm:"type:varchar(512);comment:音频文件相对路径（两级hash打散）"`
	AudioDuration *int   `json:"audio_duration,omitempty" gorm:"comment:毫秒"`
	AudioSize     *int   `json:"audio_size,omitempty" gorm:"comment:字节"`
	AudioFormat   string `json:"audio_format,omitempty" gorm:"type:varchar(20);default:'wav';comment:音频格式（固定为wav）"`

	// 元数据
	MetadataJSON string                 `json:"-" gorm:"type:json;column:metadata"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" gorm:"-"`

	// 状态
	IsDeleted bool      `json:"is_deleted" gorm:"default:false;index"`
	CreatedAt time.Time `json:"created_at" gorm:"index:idx_created_at"`
}

// ParentMessage 家长留言
type ParentMessage struct {
	ID               uint       `json:"id" gorm:"primarykey"`
	UserID           uint       `json:"user_id" gorm:"not null;index"`
	DeviceID         uint       `json:"device_id" gorm:"not null;index"`
	Title            string     `json:"title,omitempty" gorm:"type:varchar(100)"`
	TextContent      string     `json:"text_content,omitempty" gorm:"type:text"`
	AudioPath        string     `json:"audio_path,omitempty" gorm:"type:varchar(512)"`
	AudioDurationSec int        `json:"audio_duration_sec,omitempty" gorm:"default:0"`
	SourceType       string     `json:"source_type" gorm:"type:varchar(20);not null;default:'text'"`     // text, voice
	Status           string     `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"` // pending, notified, played, expired
	CreatedAt        time.Time  `json:"created_at"`
	PlayedAt         *time.Time `json:"played_at,omitempty"`
}

func (ParentMessage) TableName() string {
	return "parent_messages"
}

// DeviceEncryptionKey 设备隐私 DEK（KEK 信封包装后的密文）。
type DeviceEncryptionKey struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	DeviceID   uint      `json:"device_id" gorm:"uniqueIndex:idx_device_encryption_keys_device_id;not null"`
	KeyID      string    `json:"key_id" gorm:"type:varchar(32);not null;default:'k1'"`
	WrappedDEK string    `json:"-" gorm:"type:text;not null;column:wrapped_dek"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (DeviceEncryptionKey) TableName() string {
	return "device_encryption_keys"
}

// TableName 指定表名
func (ChatMessage) TableName() string {
	return "chat_messages"
}

// BeforeSave GORM hook - 序列化metadata
func (m *ChatMessage) BeforeSave(tx *gorm.DB) error {
	if m.Metadata != nil {
		data, err := json.Marshal(m.Metadata)
		if err != nil {
			return err
		}
		m.MetadataJSON = string(data)
	}
	return nil
}

// AfterFind GORM hook - 反序列化metadata
func (m *ChatMessage) AfterFind(tx *gorm.DB) error {
	if m.MetadataJSON != "" {
		return json.Unmarshal([]byte(m.MetadataJSON), &m.Metadata)
	}
	return nil
}

// StoryAsset 可共享故事正文（持久 SoT）。
type StoryAsset struct {
	ID                 uint      `json:"id" gorm:"primarykey"`
	StoryID            string    `json:"story_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	Title              string    `json:"title" gorm:"type:varchar(200)"`
	ThemeKey           string    `json:"theme_key" gorm:"type:varchar(100);index"`
	PoolKind           string    `json:"pool_kind" gorm:"type:varchar(20);index"` // named|open|bedtime
	CanonicalKey       string    `json:"canonical_key" gorm:"type:varchar(100);index"`
	AliasesJSON        string    `json:"aliases_json" gorm:"type:json"`
	Mode               string    `json:"mode" gorm:"type:varchar(40)"`
	NarrationMode      string    `json:"narration_mode" gorm:"type:varchar(40);index"`
	AgeBand            string    `json:"age_band" gorm:"type:varchar(40);index"`
	FullText           string    `json:"full_text" gorm:"type:text"`
	SegmentsJSON       string    `json:"segments_json" gorm:"type:text"`
	GenerationComplete bool      `json:"generation_complete" gorm:"default:false;index"`
	Shareable          bool      `json:"shareable" gorm:"default:false;index"`
	CreatorDeviceSN    string    `json:"creator_device_sn" gorm:"type:varchar(100);index"`
	CreatorUserID      uint      `json:"creator_user_id" gorm:"index"`
	ParamsSnapshotJSON string    `json:"params_snapshot_json" gorm:"type:json"`
	ReuseCount         int       `json:"reuse_count" gorm:"default:0"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (StoryAsset) TableName() string { return "story_assets" }

// StoryAssetAlias 点名池别名索引（口语异名 → story_id）。
type StoryAssetAlias struct {
	ID           uint      `json:"id" gorm:"primarykey"`
	AliasKey     string    `json:"alias_key" gorm:"type:varchar(100);uniqueIndex;not null"`
	StoryID      string    `json:"story_id" gorm:"type:varchar(64);index;not null"`
	CanonicalKey string    `json:"canonical_key" gorm:"type:varchar(100);index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (StoryAssetAlias) TableName() string { return "story_asset_aliases" }

// StoryPlayback 设备级故事播放态。
type StoryPlayback struct {
	ID                uint      `json:"id" gorm:"primarykey"`
	DeviceSN          string    `json:"device_sn" gorm:"type:varchar(100);uniqueIndex:uk_story_playback_device_story,priority:1;not null"`
	StoryID           string    `json:"story_id" gorm:"type:varchar(64);uniqueIndex:uk_story_playback_device_story,priority:2;not null;index"`
	UserID            uint      `json:"user_id" gorm:"index"`
	AgentID           string    `json:"agent_id" gorm:"type:varchar(64)"`
	LastPlayStatus    string    `json:"last_play_status" gorm:"type:varchar(40)"`
	CharOffset        int       `json:"char_offset" gorm:"default:0"`
	SegmentIndex      int       `json:"segment_index" gorm:"default:0"`
	LastSentenceIndex int       `json:"last_sentence_index" gorm:"default:0"`
	LastSentence      string    `json:"last_sentence" gorm:"type:varchar(500)"`
	PlayCount         int       `json:"play_count" gorm:"default:0"`
	CompleteCount     int       `json:"complete_count" gorm:"default:0"`
	LastPlayedAt      time.Time `json:"last_played_at" gorm:"index"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (StoryPlayback) TableName() string { return "story_playbacks" }
