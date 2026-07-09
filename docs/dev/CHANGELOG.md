# Changelog

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/)。

## [Unreleased]

### Changed

- 儿童故事：LLM 首行输出 `[[meta:title=…|genre=…|theme=…]]` 元信息（不播报），管理端标题/题材分别展示故事名称与类型（童话、神话、寓言、冒险等）
- 儿童故事：TTS 打断时同步停止 LLM 生成；未完成生成的故事管理端不展示播放进度、提示「故事未完成生成」；续讲 B 方案：过渡语后先按整句补播未听草稿，再从全文末 LLM 自然续写；已完整生成的故事仍从断点 TTS 复播
- 管理端设备故事：标题优先按题材展示（如「普罗米修斯的故事」）；状态 `abandoned` 文案改为「生成失败」

### Fixed

- 儿童故事：管理端标题误显示正文首句 — 落库与列表展示均优先用 LLM meta / 题材生成标题，旧数据在 API 层修正
- 儿童故事：TTS 打断被误判为「已放弃」— 生成取消与 TTS 取消统一记为 interrupted
- 儿童故事：生成失败/中断时过早 `tts stop` — 故事流 TTS 排空等待不受 ctx 取消影响，须播完已入队音频后再发 stop
- 儿童故事：生成失败原因不明 — 新增结构化日志（`reason=user_tts_interrupt|llm_timeout|generation_canceled|llm_error:…`、partial_runes、story_id）
- 儿童故事：追问「刚才讲了什么故事」误走复播/列表导致「内容为空」— 故事追问改走主 LLM；播完或听过即写入最近故事上下文；list_recent 不再返回空 ready
- 儿童故事：长故事 TTS 中途 `push timeout (10s)` — TTS 队列改为阻塞入队（随会话 ctx 取消），容量 10→128；双流缓冲 16→64
- 儿童故事：续讲先播一小段后又从头开始 — 续讲正文改为按 CharOffset 定位；流式 TTS 打断后不再向播报通道送句
- 儿童故事：流式故事未播完即收到 `tts stop` — 移除 LLM 通道 `IsEnd` 时过早标记完成/发 stop；故事流改为 TTS 排空后再发 `tts_stop`；打断进度在落库 segments 后正确写入
- 儿童故事：打断后「继续讲」从头开始 — 续讲回退一段并口述上次讲到的摘要；修正流式打断时 segments 为空导致进度始终为 0 的问题
- 儿童故事：打断后进度误显 100%、重讲从结尾开始 — 进度按实际 TTS 已播字符计；TTS 取消时停止 LLM 生成；复播/续讲兜底异常末段进度
- 儿童故事：流式开场过渡语重复播报 — 同主题 8 秒内防重复启动流式生成；故事 LLM 禁止再输出开场白（过渡语已由系统播报）；主 LLM 规则强调工具调用前勿口头铺垫
- 儿童故事：讲完后追问「刚才的故事」设备不知道 — 故事播完写入「最近故事」上下文并注入 system prompt，同时补写对话历史供后续 LLM 引用
- 儿童故事：打断后 Redis 正文为空导致复播/续讲提示「内容为空」— 落库改用不受会话取消影响的 context；流式打断时即时保存已生成片段；按主题复播优先找有正文的记录，无正文时自动重新生成
- 儿童故事：过渡语含具体主题（如「好呀，我给你讲经典故事，龟兔赛跑的故事。」）；无「故事」二字但含经典/神话名（如「讲龟兔赛跑」）可识别；经典童话与神话改用 classic/myth 模式正篇讲述、勿魔改；流式开始前草稿落库、打断/失败时保存片段，管理端可见 interrupted 记录
- 儿童故事流式播报：主 LLM 工具调用后过早 tts_stop 导致过渡语播完即停；关键词扩展「讲…故事」直达路由；生成上下文与会话绑定
- 儿童故事：重复同一主题 — 故事 LLM 使用独立 session 避免 Dify/Coze 对话污染；关键词路由提取用户主题
- 管理端设备故事页：修正 Redis `key_prefix` 须与主服务一致（生产为 `dili`）
- 儿童故事：「讲个故事」仅一句短回复 — 增加关键词直达 `create_child_story`、工具结果直接 TTS 朗读（绕过 50 字角色限制与第二轮 LLM）；故事生成提高 max_tokens/超时
- Memobase 接入：`config.yaml` 默认 MCP/Memory 改为本地 Memobase（API 6019、MCP SSE 6050）；修复 `search_top_k` 配置项未被 memobase 客户端读取的问题
- 长期记忆：多设备共用同一智能体时，Memobase/mem0 等长记忆按设备 ID 隔离，不再共用 agent 级记忆
- 设备对话记录：播放家长文字留言时不再重复展示 TTS 正文，保留家长留言气泡并支持双击回放 TTS 音频
- 管理端对话记录：播放按钮移至消息气泡外侧
- Memobase：修复服务端未启用 Event embedding 时搜索失败导致无法读取记忆的问题；Search 方法现在会同时获取 Profile 画像信息（包含家乡等个人信息）
- Memobase：修复 GetContext 方法使用 user.Context() API 无法正确获取画像数据的问题，改为使用 user.Profile() 获取完整画像信息并格式化为上下文
- Memobase：添加详细日志输出，包括画像详情、搜索结果、SystemPrompt 注入内容等，方便调试记忆是否生效

### Added

- 儿童故事流式播报：生成路径先 TTS 过渡语（P0），LLM 流式输出按句入 TTS（P1），生成完成后落 Redis；配置 `story.stream_enabled` / `story.filler_*`
- 管理端设备故事页：查看设备故事列表、播放进度、字数、题材、年龄段与正文详情（详见 `docs/features/DEVICE_STORY_VIEW.md`）
- 儿童故事 MCP 与 Story Memory：Local MCP 工具 `create_child_story`（生成/复播/续讲），Redis 存全文与播放进度，支持「再讲一遍昨晚的故事」与复听加权保留（详见 `docs/features/CHILD_STORY.md`）
- 管理端设备管理：「设备记忆」子页，查看 Memobase Profile/Event/Context，支持清空长期记忆；API `GET/DELETE /api/admin/devices/:id/memory`（详见 `docs/features/DEVICE_MEMORY_VIEW.md`）
- 设备对话记录：小程序首页「看记录」跳转子页，合并展示 AI 聊天与设备端已播家长留言；支持游标分页、按日期搜索、音频播放（不写库）
- 管理端设备管理：操作栏「对话记录」跳转子页，能力同上；API `/api/mp/devices/:id/conversation-records`、`/api/admin/devices/:id/conversation-records`

### Changed

- 小程序 BLE 首次连接：收集 SN 阶段忽略 `sta_code=4`（未激活）等非致命状态，不再阻断绑定确认流程；绑定成功后再进入 WiFi 配网
- 用户绑定设备：不再自动创建智能体，仅更新绑定用户、昵称与激活状态；设备须出厂预关联内置智能体，未关联时提示联系厂商；Web 端绑定改用 `POST /user/devices/bind`，无需先创建或选择智能体；小程序解绑保留出厂智能体关联
- 管理端添加设备：关联智能体按所属用户或当前操作者筛选；默认未激活且必须关联智能体；设备昵称默认为空；展示时不再用 SN 代替昵称；重复 SN 提示「设备已添加」；修复重复提示
- 管理端添加设备：所属用户改为选填且下拉仅显示非普通用户；设备标识（SN）改为必填
- 设备唯一标识全局改用 SN：OTA 仅认 `board.sn`（AIToy 必填）；MQTT topic/clientId 中间段不再做 MAC `_`↔`:` 转换；小程序 BLE 绑定仅认 `sn`，移除 `ble_mac` 回退（详见 `docs/features/DEVICE_SN_IDENTITY.md`）
- 设备唯一标识从 MAC 切换为 SN：`device_name` 存 SN；小程序 check/bind 改用 `sn` 参数；BLE Notify 解析 `sn`；管理端预登记与 Web 绑定 UI 同步；MQTT clientId 中间段与 Device-Id 均使用 SN；激活时校验 `serial_number == device_id`
- 小程序绑定页仅保留蓝牙扫描绑定，移除手动输入 SN

### Fixed

- 家长留言询问语未播完即播放原声：询问/重试 TTS 结束后再开启 ASR 确认监听，避免「要听吗」等播报被误识别为肯定；留言流程进行中阻止主 LLM 并发接管
- 家长留言确认播放后缺少过渡语：孩子明确同意后先播报「好的，接下来将播放…」再播放原声/正文
- 小程序 BLE 扫描漏设备：完整解析广播 AD 结构、名称前缀大小写不敏感、已识别设备不因后续回调丢失 `advertisData` 被过滤；进入页面自动扫描
- OTA 激活检查仍用 Device-Id Header MAC 导致查不到设备：改为优先使用 OTA 请求体 `board.sn` 作为 `device_id` 查询激活状态

- 小程序留言列表播放语音 401：补充 `GET /api/mp/messages/:id/audio`（JWT 鉴权），列表返回 `audio_url` 指向该接口
- 固件端家长留言 JSON 解析失败：主服务兼容 pending API 旧版单对象/新版数组响应；失败后允许重试
- `config.yaml` 默认 `manager.auth_token` 与控制台 `internal_auth_token` 对齐，避免内部 API 401
- 家长留言 MQTT 播报：`hello` 完成后再启动；`speak_ready` 超时等可重试错误保持 `pending`
- AI 玩具协议兼容：默认关闭 `speak_request` 握手，主动播报（含家长留言）改走 hello + session + UDP + tts 标准链路
- 家长留言通知超时：hello/UDP 就绪后重置 60s 检测窗口；UDP 首次绑定时再触发；日志输出分环节就绪状态
- pending 查询包含 `notified` 状态，避免询问后列表漏条
- `msg_play` 按时间/亲属筛选时搜不到留言：新增内部 search API，在库内按 `created_at` 检索含 `played` 的留言；`latest`/筛选播放不再受已播列表 20 条上限影响
- 小程序留言列表：「新建留言」移至顶部；支持删除任意状态留言（含已播放）
- 小程序留言列表：`看记录` 仅显示当前设备留言；底部 tab「留言」显示全部设备留言

### Added

- 留言意图路由：`msg_inquiry`（查询）、`msg_play`（播放/重播/按条件筛选）、`general`（常规聊天）；JSON schema 见 `internal/domain/chat/intent/schemas.go`
- 设备留言档案 `DeviceMessageProfile`（新留言状态、已播历史、重播上一条）
- 内部 API：`GET .../parent-messages/played`、`GET .../parent-messages/:id`
- 内部 API：`GET .../parent-messages/search`（按时间检索含已播留言，供 `msg_play` select/latest）
- 初始化项目治理（full 档）：`governance-kit/`、`llms.txt`、`scripts/check-governance.ps1`；精简 `AGENTS.md`

### Changed

- 删除留言 `skipped` 状态；拒绝收听保持 `pending`，不可播标 `expired`；历史 `skipped` 迁移为 `pending`
- 取消 `manager_created` / `mqtt_transport_ready` 过早触发家长留言通知
- 家长留言：开机 hello+UDP 就绪后主动询问；使用中轮询检测新留言并主动询问（拒绝后不重复问同一条）
- `msg_play` 支持 `latest`（最近一条）与 `select`（按亲属 + start/end 时间范围筛选播放）
- 修复 `msg_play`/留言重播多段 TTS 时过渡语触发 goodbye 导致正文未播放
- 文字留言播放：过渡语/确认语使用 `ttsTurnEndPolicyNone`，正文 TTS 播完后再 goodbye；多段播报间等待 TTS 排空
- MQTT 监听端口 6887、TLS 6888；管理前端 dev 代理指向远程 API

### Added

- 家长留言升级：亲属角色（`family_role`）、留言标题、语音原声存储（最长 60 秒，无需 ASR）
- 小程序留言 UX：按住说话、上滑取消、松手填标题、语音/文字列表徽章
- 内部 API：`GET /api/internal/parent-messages/:id/audio`；pending 返回全部待播留言
- 主服务留言编排：3 小时间隔批量播放、LLM 意图识别、语音 MediaPlayer 原声 / 文字 TTS

### Changed

- 家长留言设备端：由固定文案 + 关键词改为 Agent 人设询问 + LLM 意图（关键词降级）
- `PATCH /api/mp/profile` 支持更新亲属角色；登录/资料返回 `family_role`

### Added
- 家长留言微信小程序（子模块 `manager/miniprogram` → `dilidili_mp`）：微信登录、设备绑定、文字/录音留言
- 小程序 API（`/api/mp/*`）：鉴权、设备 check/bind、留言 CRUD
- `parent_messages` 数据模型与内部 pending 查询 API
- 主服务设备上线家长留言 TTS 询问 + ASR 确认播报流程
- 管理端设备 MAC 出厂预登记校验提示

### Added

- 小程序绑定流程优化：BLE 获取 MAC 后先绑定再 WiFi 配网，配网完成后开放系统设置
- 小程序登录页 chooseAvatar + 昵称；首页/我的页展示头像昵称
- 小程序「我的设备」解绑（`DELETE /api/mp/devices/:id`）
- 文档 [`docs/features/MP_BIND_FLOW.md`](../features/MP_BIND_FLOW.md)

### Changed

- 小程序绑定自动创建智能体时使用光曜星角色介绍模板（支持 `{{assistant_name}}` 占位符）
- 设备表唯一索引由 `device_code` 单列改为 `device_name` + `device_code` 联合唯一
- 小程序 device-config 移除底部手动绑定卡片，绑定前置至 ble-connect
- 更新 [`AI玩具BLE流程.md`](../AI玩具BLE流程.md) 绑定时序说明

### Fixed

- 绑定放在流程末尾导致用户配网后未绑定时首页/我的设备仍显示「未绑定」

### Removed

## [0.1.0] - 2026-06-10

### Added

- 初始版本

[Unreleased]: compare/link/here
