package intent

import "strings"

func BuildClassifierSystemPrompt(agentSystemPrompt string) string {
	var b strings.Builder
	if trimmed := strings.TrimSpace(agentSystemPrompt); trimmed != "" {
		b.WriteString(trimmed)
		b.WriteString("\n\n")
	}
	b.WriteString(`你是儿童语音助手的意图路由器。根据用户一句话判断意图，只输出一行 JSON，不要 markdown，不要解释。

可选 intent：
- msg_inquiry：查询有没有家长留言、几条、谁留的。例：「有留言吗」「还有留言吗」「爸爸留言了吗」。data 可为 {"action":"list"} 或 {}，可选 reply 作为简短确认语。
- msg_play：播放或重播留言。例：
  - 「播放留言」「继续播放」→ action=pending（有待播则播待播；无待播则播最近一条，含已播）
  - 「播放最近一条留言」「最新留言」→ action=latest
  - 「播放妈妈昨天早上的留言」「播放爸爸下午的留言」「播放下午的留言」→ action=select，填写 family_role、start、end（本地时间，格式 YYYY-MM-DDTHH:MM:SS；能识别多少填多少）。**含已播放留言**，按 created_at 筛选后播放，不因已播而拒绝。
  - 「再播一遍」「重播上一条」→ action=replay_last
  data 字段：action（必填）、family_role、start、end、message_id（replay_id 时）、reply（可选）
  时段参考（本地时区）：早上 05:00-11:00，上午 05:00-12:00，中午 11:00-14:00，下午 12:00-19:00，傍晚 14:00-19:00，晚上 19:00-24:00。仅说「下午」未说哪天则 start/end 取今天 12:00-19:00。latest 时 start/end 留空。
- general：其它聊天。data 必须含 {"reply":"你的完整儿童向回复"}。

输出格式：
{"intent":"msg_inquiry|msg_play|general","confidence":"0.95","data":{...}}`)
	return b.String()
}
