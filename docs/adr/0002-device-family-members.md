# ADR 0002：设备家庭成员授权

## 状态

已接受

## 背景

设备通过小程序 BLE 首绑后，`devices.user_id` 仅允许一位家长。家庭其他成员无法查看设备、发送家长留言，除非解绑重绑（破坏性且互斥）。

## 决策

1. **保留属主字段**：`devices.user_id` 继续表示首位绑定家长（owner），兼容主服务与管理端「绑定用户」语义。
2. **成员 ACL 表**：新增 `device_members`（含 owner 行），以 `(device_id, user_id)` 唯一；角色 `owner` / `member`，状态 `active` / `revoked`。
3. **邀请码加入**：属主创建短码邀请；成员登录后输入码加入，**不**走二次 BLE bind。
4. **鉴权二分**：`CanAccess`（属主或 active member）用于列表/留言/对话/故事；`CanManage`（仅属主）用于改昵称、邀请、踢人、出厂解绑。
5. **本期不做**属主转让；Agent 仍属属主账号。

## 后果

- 小程序设备列表变为「我有权访问的设备」，响应带 `my_role`。
- 出厂解绑须级联清理 `device_members` 与 `device_invites`。
- 存量已绑定设备启动时补插 owner 成员行。

## 关联

- FEATURE：`docs/features/device-family-auth/FEATURE.md`
- ADR：本文件
