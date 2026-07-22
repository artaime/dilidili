# 设备家庭成员授权

## 状态

done

## 需求

- 背景：一台设备目前只能绑定一位家长；首位绑定者需能授权其他家庭成员共同使用（看设备、发留言等），且不破坏属主与出厂解绑语义。
- 绑定写入的孩子昵称（`nick_name`）仅属主可修改。
- 验收标准：
  1. 首位绑定者仍是唯一属主，可解绑、邀请、改昵称、踢人。
  2. 属主邀请后，成员可不经 BLE 看到该设备并发留言。
  3. 成员无法解绑、无法邀请、无法改昵称、无法踢人；可主动退出。
  4. 踢出/退出后成员立即失去访问。
  5. 出厂解绑后成员关系与邀请全部清空，设备可被新人首绑。
  6. 每设备最多 6 人（1 owner + 5 member）；邀请码 6 位、24h、最多 5 次。

## 设计

### 权限矩阵

| 能力 | Owner | Member |
|------|-------|--------|
| 查看设备 / 对话记录 / 故事 | ✓ | ✓ |
| 家长留言（发/看自己的） | ✓ | ✓ |
| 改孩子昵称 | ✓ | ✗ |
| 邀请 / 踢人 | ✓ | ✗ |
| 出厂解绑 | ✓ | ✗ |
| 主动退出 | ✗ | ✓ |

### 影响模块

- Manager 模型 / AutoMigrate / 存量补 owner 行
- `services/device_acl`、`device_reset`、小程序设备/留言/故事鉴权
- 小程序：设备列表、家庭成员页、加入入口、改昵称

### API / 配置变更

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| PATCH | `/api/mp/devices/:id` | manage | 改 `nick_name` |
| POST | `/api/mp/devices/:id/invites` | manage | 创建邀请码 |
| POST | `/api/mp/devices/join` | 登录用户 | `{code}` 加入 |
| GET | `/api/mp/devices/:id/members` | access | 成员列表 |
| DELETE | `/api/mp/devices/:id/members/:userId` | manage | 踢人 |
| POST | `/api/mp/devices/:id/leave` | member | 退出 |

`GET /api/mp/devices` 返回可访问设备并带 `my_role`。

## 开发计划

- [x] ADR 0002
- [x] Model + AutoMigrate + 存量补行
- [x] device_acl + 首绑/解绑挂钩
- [x] 邀请/成员/改昵称 API + 鉴权改造
- [x] 小程序 UI
- [x] `go test ./...` + CHANGELOG + DOC_SYNC

## 测试

- 首绑写入 owner member；存量迁移补行
- 邀请加入、满员、过期、重复加入
- member / owner 权限边界
- 解绑清空 members/invites

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-17 | 立项并实现：ACL + 邀请码 + 小程序 UI |
