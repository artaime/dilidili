# Changelog

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/)。

## [Unreleased]

### Added

- 管理端设备列表：后端分页筛选（`page`/`page_size`/`device_id`/`nick_name`/`bind_user`/`activated`/`agent_id`），本人绑定置顶，响应含 `agent_stats`；新增 `GET /api/admin/devices/:id`。详见 `docs/features/admin-device-list-filter/FEATURE.md`
- 用户对话与留言落盘加密：AES-256-GCM + 设备级 DEK（`PRIVACY_KEK_BASE64` / `encryption.enabled`）；小程序读路径统一 `CanAccess`；存量迁移命令 `manager/backend/cmd/migrate_privacy_encrypt`。详见 `docs/dev/FEATURE_privacy_encryption.md`

### Fixed

- 家长留言「播放最近的留言」：同步已播历史时 API 为 `played_at DESC`，原先原样写入导致 `LastPlayedRef` 取到时间最早一条；现规范为升序并按 `PlayedAt` 取最近一次播放
- 管理端预录入设备：未选所属用户时允许关联任意已有智能体（此前误按管理员归属校验，报「智能体不存在或不属于指定用户」）
- 管理端添加设备：设备标识（SN）输入框与上方字段等宽；未选所属用户时不再按当前管理员过滤智能体（此前会误显示 No data）
- 设备在线状态：主服务对仍持有 `ChatManager` 的设备每 2 分钟刷新 `last_active_at`（可配 `device_online.heartbeat_interval`），避免连着超过 5 分钟被管理端误判离线
- 设备下线不再清空 `last_active_at`：此前 `/api/device/inactive` 置 NULL 导致管理端「最后活跃时间」显示「从未活跃」；在线仍按最近 5 分钟内活跃判定
- 管理端设备列表：进入页面与重置筛选时本人绑定设备置顶（加固后端 ORDER，前端无筛选时再兜底排序）
- 儿童故事开场过渡语重复两遍：开放流式不再先播通用 filler 再播标题告知，改为 meta 后单句开场；续讲正文已有「上次讲到」时不叠标题引导；跳过模型重复开场白。详见 `docs/features/story-narration-intro/FEATURE.md`
- 意图/故事旁路抢答：分类注入近期 Dialogue；去掉关键词 fallback 与故事召回快路径；`general` 一律交主对话；无名故事追问无正文时放行；「介绍……的故事」事实介绍不进儿童编故事。详见 `docs/features/INTENT_ROUTER_CONTEXT.md`

### Changed

- 管理端用户列表：「用户名」显示 `nickname`，新增「账号」列显示 `username`
- 管理端设备管理：仅可查看**本人绑定**设备的对话记录、设备记忆、故事等隐私内容（前后端双重门禁）；操作栏右对齐且不透明。详见 `docs/features/admin-device-privacy-gate/FEATURE.md`
- 意图路由：分类器注入近期 Dialogue，删除关键词 fallback；`general`/`device`/`needs_dialogue` 交主 LLM；事实介绍 vs 儿童讲故事消歧。详见 `docs/features/INTENT_ROUTER_CONTEXT.md`
- 家长留言音频默认落盘路径由 `./storage/parent_messages/audio` 迁至 `./data/parent_messages/audio`（与聊天录音同属 `data/`）。详见 `docs/features/parent-message-audio-data-dir/FEATURE.md`
- 小程序语音球 / 扫描球：空闲与结束后用静帧 `voice-ball.png` / `ble-ball.png`，仅录音·试听·列表播放 / 扫描中播对应 GIF；绑定设备页进入不再自动扫描。详见 `docs/features/mp-ball-still-anim/FEATURE.md`
- 小程序素材：`dili-ble-connect.gif` gifsicle 有损压缩（约 7.0MB → 4.4MB），分辨率仍为 480×480、25fps / 121 帧
- 设备蓝牙绑定页：顶部吉祥物 `dili-ble-connect.gif`，中间扫描动效 `ble-ball.gif`；语音中间球文件对齐为 `voice-ball.gif`。详见 `docs/features/mp-mascot-remote/FEATURE.md`
- 小程序素材：语音留言中间录音球改用 `dili-voice-ball.gif`；「我的」页顶部改用 `dili-screen.gif`；设备蓝牙绑定页改用 `dili-ble-ball.gif`。详见 `docs/features/mp-mascot-remote/FEATURE.md`
- 「我的」页与设备绑定页吉祥物：统一使用 `ble-connect.gif` 网络下发（替换原 `profile-mascot.svg` / 包内 `hero-mascot.png`）。详见 `docs/features/mp-mascot-remote/FEATURE.md`
- 新建留言页英雄区动效：语音模式展示 `dili-voice-record.gif`，文字模式展示 `dili-type-record.gif`（替换原统一 `record.webp`）；切换模式时同步切换，并预热另一模式缓存。详见 `docs/features/mp-mascot-remote/FEATURE.md`
- 小程序首页背景 `home-bg.png`：pngquant 压缩（约 770KB → 138KB），尺寸仍为 750×842
- 首页头部：改为 `home-bg.png` 背景 + 右侧 `dili-flower.gif` 吉祥物（不再使用 `home.mp4`）；logo / 问候语叠在左侧。详见 `docs/features/mp-mascot-remote/FEATURE.md`
- 首页头部：`home.mp4` 底层循环播放（VideoContext 强制起播）；logo / 问候语叠在视频上层左侧；原透明 WebP 改名为 `record.webp` 用于留言页。详见 `docs/features/mp-mascot-remote/FEATURE.md`
- 首页吉祥物：透明 animated WebP `thankful_plush.webp`（960² / 16fps，约 5MB），`<image webp>` 播放，失败回退透明海报；源片备份为 `thankful_plush_source.mp4`（不嵌入）。详见 `docs/features/mp-mascot-remote/FEATURE.md`
- `thankful_plush.svg`：去掉白底，内嵌透明 PNG，并加上 SMIL/CSS 轻浮动呼吸动画（体积约 53KB）
- 小程序「绑定设备」「系统配置」按 Figma（296:799 / 402:344）复刻：扫描球与设备列表「链接设备」、空态重新扫描；配置页滑块调亮度/音量、Wi-Fi 单选与毛玻璃风格卡片。详见 `docs/features/mp-bind-config-ui/FEATURE.md`
- 小程序「我的设备」「家庭成员」按 Figma（488:788 / 585:3171 / 523:1875）复刻：设备卡菜单行、空态插画、邀请码与成员列表；改昵称 / 输入邀请码为毛玻璃弹窗并随键盘上移。空态吉祥物 `devices-empty-mascot.svg`（>50KB）走 `/static/mp/`。详见 `docs/features/mp-devices-members-ui/FEATURE.md`
- 小程序「对话记录」按 Figma（325:1536）改造：淡蓝底、半透明筛选条+日历、左右气泡与圆角方头像；「我的」页按 Figma（300:1152）复刻毛玻璃用户卡（吉祥物压在卡下）、无头像用默认吉祥物头像。详见 `docs/features/mp-profile-records-ui/FEATURE.md`
- 小程序新建留言页按 Figma（语音待录/录音中/录音完成）重构：英雄区问候+吉祥物、分段切换、音球与胶囊按钮；录音完成后不再强制弹标题，改为「试听留言 / 设置标题」；音球 `create-orb.png`（>50KB）走 `/static/mp/` 网络下发；标题弹窗对齐授权页毛玻璃蒙层；多设备时「新建留言」先选对象，单设备直达创建页
- 儿童故事开放/随便生成：随机指定题材与切入点，强制新鲜主角名，并注入近 7 天人物名/主题回避名单，减轻题材雷同与人物名复用。详见 `docs/features/story-diversity/FEATURE.md`
- DeepSeek LLM：未配置 `thinking` 时默认注入 `thinking.type=disabled`（官方 API 默认开启思考）；管理端表单默认「关闭」，可通过 `thinking.mode: enabled` 开启。详见 `docs/features/deepseek-thinking-default-off/FEATURE.md`
- 儿童故事追问：去掉每轮 system 注入全文；会话只留 `story_id`/标题指针，意图判为 followup 后按需从 Redis→MySQL 取正文本轮作答；点名经典未讲过可直答，非经典未讲过澄清后礼貌收尾。详见 `docs/features/story-followup-ondemand/FEATURE.md`

### Added

- 小程序素材分发阈值改为 **>50KB** 才网络下发（`/static/mp/*`）；≤50KB 打进包；首页 `thankful_plush.mp4` 固定远端。规范见 `.cursor/rules/05-miniprogram-assets.mdc`、`docs/features/mp-mascot-remote/FEATURE.md`
- 设备家庭成员授权：属主可邀请其他家长加入同一设备（邀请码）；成员可查看设备/发留言，仅属主可改孩子昵称、解绑与踢人。详见 `docs/features/device-family-auth/FEATURE.md`、`docs/adr/0002-device-family-members.md`
- 短时多轮衔接：跨 session 按 `user_id+device_id+agent_id` 从 Manager DB / Redis shortctx 灌入近期对话；配置 `chat.short_context`；fresh hello 可复用 SessionID；出厂重置清理 shortctx。详见 `docs/features/SHORT_CONTEXT_CONTINUITY.md`
- 设备固件状态问答与控制：IoT MCP 工具（`get_device_status` / `set_speaker_volume` / `set_screen_brightness` / `enter_sleep_mode` / `power_off_device`）转换时追加调用引导（问状态须主动 get，相对调节先 get 再 ±10）；能力地面补强状态查询与睡眠/关机完成态护栏。详见 `docs/features/DEVICE_FIRMWARE_STATUS.md`
- LLM 能力地面（防乱答）：按本轮 tools 注入能力白名单与诚实回答规则；无 tool call 时改写「已帮你关/调/设…」类虚构完成态话术；意图路由 general 同步约束。详见 `docs/features/LLM_CAPABILITY_GROUNDING.md`
- 管理端设备故事支持单条删除与清空（仅本机 playback + Redis，不删共享资产）；详见 `docs/features/device-story-delete/FEATURE.md`
- 管理端「故事管理」：共享资产列表/新增/编辑/删除，支持 AI 生成正文；详见 `docs/features/story-asset-admin/FEATURE.md`
- 故事意图 LLM 输出规范名 `canonical`（纠 ASR 谐音）并以之查共享池，口语 `theme_raw` 写入别名；详见 `docs/features/story-asset-playback/FEATURE.md`
- 故事共享池方案 A：点名池（canonical/别名）与开放池（open/bedtime）、设备近 7 天排斥 + Top-K 随机取用；详见 `docs/features/story-asset-playback/FEATURE.md`
- 儿童故事资产库 / 进度保护 / 聊天短卡片：MySQL `story_assets`+`story_playbacks`（Redis 热缓存 dual-write）、`protect_continue_threshold` 停播不停写、历史短文案「播放故事：标题」点击拉全文、canonical 跨用户复用已完成正文；详见 `docs/features/story-asset-playback/FEATURE.md`、`docs/adr/0001-story-asset-playback.md`
- 腾讯 ASR（`tencent_asr`）：接入腾讯云实时语音识别 WebSocket v2，支持流式识别与管理台配置测试；详见 `docs/dev/FEATURE_tencent_asr.md`
- 腾讯 TTS（`tencent_tts`）：接入腾讯云流式文本语音合成 v2（stream_wsv2），支持管理台配置、音色选择与双流式合成；详见 `docs/dev/FEATURE_tencent_tts.md`
- 百炼 CosyVoice TTS（`aliyun_cosyvoice`）：接入阿里云百炼官方 SpeechSynthesizer HTTP API（SSE 流式），支持管理台配置、按模型选择系统音色与配置测试；详见 `docs/dev/FEATURE_aliyun_cosyvoice.md`

### Fixed

- 智能体虚构能力推销（如主动说能查天气/定闹钟，被问后又说不会）：能力地面禁令补强；`general` 含推销话术改交主对话；无对应工具时落历史前改写为仅陪伴向短句。详见 `docs/features/LLM_CAPABILITY_GROUNDING.md`
- 家长留言播放被按键打断（context canceled）时不再播「播放留言失败了，稍后再试试吧」
- 家长留言「播放最近的留言」：未指定家长时重播刚播过的那条（多家长不再误播他人创建时间更新的留言）；指定家长（如「妈妈最近的留言」）则播该家长按创建时间最新一条。详见 `docs/features/PARENT_MESSAGE.md`
- 空拾音 `listen start`→`listen stop`（无语音、无 chat turn）触发 FunASR `EmptyAudio` 时不再 `fatal` 关会话并发 MQTT goodbye，仅结束本轮 ASR、保持 ChatSession
- 流畅对话：欢迎语/助手 TTS 期间 `listen start` 改为暂存、输出结束后补发，避免误 `StopSpeaking` 打断播报又丢掉下次拾音；助手输出中忽略 `detect`；soft `VoiceStop` 在 TTS 结束或 ASR 仍开着时自动清除；上行音频重置空闲计时，避免未满 `max_idle_duration` 就 goodbye；去掉逐包 UDP/收包 debug 刷屏
- auto 模式连续对话（如调音量 80 后再调 50）无回复并 goodbye：ASR 入队后误触发「拾音卡住恢复」，紧接着 listen stop 产生 FunASR `EmptyAudio` 被当成 fatal 关闭会话；对话处理中禁止自动恢复拾音，listen stop 走软停，EmptyAudio 等断开类错误不关会话
- ASR 识别正确却无回复（`DoLLmRequest` 起手即 `context canceled`）：本轮对话改绑独立 chat turn ctx（父级 SessionCtx），不再与 `AfterAsrSessionCtx` 共用 cancel；`realtime_mode=4` 空闲态 ASR 首字不再 `StopAssistantOutput`（避免与入队 turn / 意图路由竞态）
- 固件 set 成功后二次 LLM 把「已调好」误改成「做不到」：工具成功后标记本轮已落地，能力地面不再改写；完成态话术与 tool_calls 乱序时暂存而非立即改写；set 成功结果附明确成功语义。详见 `docs/features/DEVICE_FIRMWARE_STATUS.md`
- 意图路由 `general` 短路导致「问电量编造没有电表 / 调音量被当成闲聊」：新增 `device` 意图交主 LLM+MCP；general 若含虚构设备操作也回退主对话。详见 `docs/features/DEVICE_FIRMWARE_STATUS.md`
- TTS 未播完即收到 `tts stop`/`goodbye`：stop 等待改为 `playbackTail + sentenceControlDelay + 400ms`（覆盖 sentence_start 起播滞后与设备缓冲）；TTS/欢迎语期间设备 `listen stop` 仅停上行，不走 `OnManualStop`；欢迎语 TTS 绑 client ctx 并成对 `IsStart/IsEnd`
- 连说两次「讲一个故事」会串播两篇：空主题去重 + 播报中拒绝重复 generate；详见 `docs/features/story-playback-ux-fix/FEATURE.md`
- 未完整生成的故事不再入共享池（清 `pool_kind`/别名，`shareable=false`）
- 故事完播误标「打断」、进度停在开头：播报中周期性落进度，TTS 正常排空且生成完整则标 `completed`；管理端进度仅显示百分比
- 管理端设备对话记录与小程序一致：展示「播放故事：标题」，点击再查全文
- 管理台 TTS 保存百炼 CosyVoice 后 provider 被误判为 OpenAI，导致配置测试与运行时合成失败
- realtime/auto 模式下 ASR 重启后未恢复拾音（`VoiceStop` 仍为 true），导致 UDP 音频被持续跳过且无识别结果
- 腾讯 ASR 发送 `end` 后长时间无 final 响应时会阻塞 ASR 结果循环
- auto/realtime 模式 ASR 结果循环退出后 `VoiceStop` 未清除时，设备持续发音频会被永久跳过（新增自动恢复拾音）
- 腾讯 TTS 双流式合成结束时重复关闭 `audioFrameChan` 导致 `panic: close of closed channel`
- 故事流式播报期间扬声器回声触发 ASR/VAD 打断，导致 TTS 提前终止且 LLM 故事内容不再继续输出
- 腾讯 ASR 4008（15 秒无音频）在 TTS/故事长播报期间被当作致命错误关闭会话，导致 TTS 播放中途被打断
- auto 模式下 TTS/LLM 长播报仍累计 `max_idle_duration`（30s）空闲超时并关闭会话，打断故事播报（realtime 此前已豁免）
- auto 模式普通对话 TTS 播报期间扬声器回声触发 ASR 新轮次，导致腾讯双流式 TTS 合成被提前收口、播放话说一半
- 腾讯 ASR 4008 在助手输出期间反复释放 ASR 资源，长回复 TTS 期间识别链路抖动
- 腾讯 TTS 双流式在 LLM 句间停顿超过读超时时误断开 WebSocket，后续音频不再合成
- TTS/故事播报期间腾讯 ASR 通道关闭后空结果被快速累计，触发「3次/3s」保护并 fatal 关闭会话，导致 TTS 中途被打断、故事 LLM 不再继续输出

### Changed

- 新增 `chat.debug_log_tts_only`：开启后仅打印 TTS 合成正文（`设备 xxx TTS合成完成` / `故事TTS合成完成`），抑制 ASR/LLM 流式调试与助手输出保护类日志；兼容旧配置 `story.debug_log_tts_synthesized`
- 儿童故事：新增配置 `story.debug_log_tts_synthesized`，开启后仅在每句 TTS 合成并发送完成时打印故事正文（调试用，已废弃，请用 `chat.debug_log_tts_only`）
- 主服务运行时监控自研看板：多节点自动注册、`runtime_report` 上报、Admin SSE 实时看板（CPU/内存/磁盘/带宽、连接在线、会话活跃、资源池）；详见 `docs/features/SERVER_RUNTIME_MONITOR.md`
- 小程序解绑出厂重置：删除设备全部记忆/故事/对话/留言数据，恢复出厂登记状态（保留 SN、激活码、出厂智能体）；详见 `docs/features/DEVICE_UNBIND_RESET.md`
- 管理端设备出厂重置：`POST /api/admin/devices/:id/factory-reset`；删除设备前先清理业务数据；编辑禁止静默解绑

### Changed

- MQTT/UDP 主动播报（`auto_listen=false`）TTS 结束后不再立即发送 goodbye，改为等待 `chat.max_idle_duration` 空闲超时；客户端上传音频/信令会重置计时，`listen start` 会取消等待
- 未激活设备 TTS 提示改为「请在小程序上绑定我，双击电源键进入配网模式」
- 小程序解绑：由仅清 `user_id`/`activated` 改为全量数据清理 + 重置 `nick_name`/`role_id`/`last_active_at`
- 管理端设备列表：绑定用户列、筛选、出厂重置按钮；设备记忆页「清空 Memobase 长期记忆」文案与全量重置区分

- 儿童故事：LLM 首行输出 `[[meta:title=…|genre=…|theme=…]]` 元信息（不播报），管理端标题/题材分别展示故事名称与类型（童话、神话、寓言、冒险等）
- 儿童故事：TTS 打断时同步停止 LLM 生成；未完成生成的故事管理端不展示播放进度、提示「故事未完成生成」；续讲 B 方案：过渡语后先按整句补播未听草稿，再从全文末 LLM 自然续写；已完整生成的故事仍从断点 TTS 复播
- 管理端设备故事：标题优先按题材展示（如「普罗米修斯的故事」）；状态 `abandoned` 文案改为「生成失败」

### Fixed

- 家长留言：说「播放留言」且无待播时，回退播放最近一条留言（含已播），支持重复收听
- 小程序/管理端解绑：Memobase 中 primary/legacy 用户不存在时不再阻断解绑（幂等跳过 `User not found`）
- 家长留言确认：主动询问仅自然提问是否播放（不引导口令）；儿童回复改由 LLM 判断 play/skip/unknown，意图不清时自然过渡聊天并继续等待
- 小程序语音留言：修复录音启动/结束竞态（`onStart` 前保存、超时晚到 `onStop` 丢录音）、放宽静音检测、录音前补隐私授权；上传 401 跳转登录；对话记录单击播放；留言页保留已选设备
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
