# 用户对话与留言加密保存

## 背景

对话正文、家长留言正文及对应音频当前明文落盘，存在库备份/磁盘泄漏风险。本功能在 Manager 后端对敏感字段做应用层加密，并用设备 ACL 限制谁能看到明文。

## 范围

| 资产 | 处理 |
|------|------|
| `chat_messages.content` | AES-256-GCM 字段加密 |
| 对话音频文件 | 整文件加密后落盘 |
| `parent_messages.text_content` | AES-256-GCM 字段加密 |
| 留言语音文件 | 整文件加密后落盘 |
| `tool_calls` / metadata / 索引字段 | 不加密 |

不做客户端 E2E；主服务经 internal API 仍接收明文（供 ASR/LLM/TTS）。

## 密钥

- **KEK**：环境变量 `PRIVACY_KEK_BASE64`（32 字节密钥的 Base64），禁止入库/提交。
- **DEK**：每设备一把，KEK 信封包装后存 `device_encryption_keys`。
- 配置：`encryption.enabled`、`encryption.key_id`（如 `k1`，便于轮换标识）。

## 密文格式

文本与音频文件内容统一：

```
v1|<key_id>|<base64(nonce || ciphertext || tag)>
```

无此前缀视为历史明文（双读兼容）。

## ACL

小程序读路径：`device_acl.CanAccess`（属主或 active 成员）。

- 对话列表：已按设备校验，保持。
- 对话音频：经消息所属设备 `CanAccess`（不再仅比 `message.user_id`）。
- 留言 List/Get/Audio/Delete：按设备 `CanAccess`；带 `device_id` 时返回该设备全部留言。

Admin 保留解密查看，写审计日志。Internal 接口凭 internal token 解密下发。

## 迁移

```bash
cd manager/backend
PRIVACY_KEK_BASE64=... go run ./cmd/migrate_privacy_encrypt -c config/config.json
```

幂等：已是 `v1|` 前缀则跳过。`encryption.enabled=true` 时新写入强制密文。

## 验收

1. 新数据 DB/磁盘均为密文。
2. 非成员访问列表/音频 → 404。
3. 属主与 active 成员可正常查看与播放。
4. 设备端留言播放与短上下文加载正常。
5. `enabled=true` 且未设 KEK 时进程拒绝启动。

## 实现位置

- `manager/backend/privacy/`
- `manager/backend/models`（`DeviceEncryptionKey`）
- `manager/backend/controllers/chat_history.go` 等
- `manager/backend/cmd/migrate_privacy_encrypt/`
