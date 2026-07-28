package intent

import "encoding/json"

const (
	IntentMsgInquiry = "msg_inquiry"
	IntentMsgPlay    = "msg_play"
	// IntentDevice 本机固件状态查询/控制（音量、电量、亮度、睡眠、关机等），意图路由器不短路，交给主 LLM + 设备 MCP tools。
	IntentDevice  = "device"
	IntentGeneral = "general"
)

type RouterResponse struct {
	Intent        string          `json:"intent"`
	Confidence    string          `json:"confidence"`
	NeedsDialogue bool            `json:"needs_dialogue,omitempty"`
	Data          json.RawMessage `json:"data"`
}

type GeneralData struct {
	Reply string `json:"reply"`
}

type MsgInquiryData struct {
	Action string `json:"action,omitempty"`
	Reply  string `json:"reply,omitempty"`
}

type MsgPlayData struct {
	Action     string `json:"action"` // pending | latest | select | replay_last | replay_id
	MessageID  uint   `json:"message_id,omitempty"`
	FamilyRole string `json:"family_role,omitempty"` // 爸爸/妈妈/爷爷/奶奶...
	Start      string `json:"start,omitempty"`         // 筛选起始时间，RFC3339 或 YYYY-MM-DDTHH:MM:SS
	End        string `json:"end,omitempty"`           // 筛选结束时间
	Reply      string `json:"reply,omitempty"`
}

const (
	MsgPlayActionPending    = "pending"
	MsgPlayActionLatest     = "latest"
	MsgPlayActionSelect     = "select"
	MsgPlayActionReplayLast = "replay_last"
	MsgPlayActionReplayID   = "replay_id"
)
