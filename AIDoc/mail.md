# 邮件系统实现说明（`server/logic/mail`）

## 1. 当前实现（已同步到代码）

### 1.1 核心实体
- `MailEntity`（表：`mail`）
  - 玩家个人邮件
  - 关键字段：`mail_id`、`user_id`、`server_mail_id`、`status`、`attachments`
- `ServerMailEntity`（表：`server_mail`）
  - 统一承载：
    - 全服邮件（`alliance_id = 0`）
    - 联盟邮件（`alliance_id > 0`）
  - 关键字段：`server_mail_id`、`server_id`、`alliance_id`、`unlock_list`、`status`、`expire_time`

### 1.2 联盟邮件实现方式（已改为单表）
- 不再使用独立 `alliance_mail` 表。
- 联盟邮件直接写入 `server_mail`，通过 `alliance_id` 区分。
- 玩家邮件侧仍通过 `mail.server_mail_id` 做去重映射。

### 1.3 派发流程
1. `GetMailList` 前置执行：
   - `checkAndCreateServerMails(userId)`：仅处理 `alliance_id = 0` 的全服邮件
   - `checkAndCreateAllianceMails(userId)`：处理当前玩家所属联盟的邮件（`alliance_id = 玩家联盟ID`）
2. 两者都写入 `mail` 表副本，保证客户端拉列表时是玩家私有邮件视图。

### 1.4 玩家联盟归属来源（关键）
- **不查 `alliance_member` 表**。
- 使用 Redis 玩家联盟信息：
  - `REDIS_PLAYER_ALLIANCE_INFO`
  - 代码路径：`logicCommon.GetPlayerAllianceInfoFromRedis(userId)`
- 规则：
  - `allianceId <= 0`：不派发联盟邮件
  - `allianceId > 0`：才派发联盟邮件

### 1.5 登录红点逻辑
- `OnPlayerLogin` 的未读红点 = 
  - 全服邮件可见数量（`alliance_id=0`，并满足 unlock 条件）+
  - 联盟邮件数量（`alliance_id=玩家联盟ID`，`status=Sent` 且未过期）

### 1.6 GM 发送逻辑（`GmSendMail`）
- `ul` 为空：发全服邮件（写 `server_mail`，`alliance_id=0`）
- `ul` 正数：发个人邮件（写 `mail`）
- `ul` 负数：发联盟邮件（取绝对值为 `alliance_id`，写 `server_mail`）
- 联盟邮件发送后：
  - 查 `alliance_member` 成员
  - 写 Redis `REDIS_MAIL_REFRESH_USERS`，触发成员邮件缓存刷新

### 1.7 排行榜联盟结算邮件来源（新增）
- 来源：Rank 节点结算联盟榜时写入联盟邮件（`server_mail`，`alliance_id>0`）。
- 覆盖榜单：
  - 联盟竞技场积分周榜（`pointType=10`）
  - 联盟荣耀竞技场轮次胜场榜（`pointType=9`）
- 发送实现：`rankBoardPlatform.SendRankBoardAllianceRewardMail`
  - 奖励内容来自排行榜 `rankReward` 掉落；
  - 每次写入后会将联盟成员加入 `REDIS_MAIL_REFRESH_USERS`，触发邮件缓存刷新；
  - 结算幂等由 Rank 侧 Redis 锁控制（按 `rankId` 一次性）。

---

## 2. 风险点

1. **DB 结构依赖**
   - 需确保 `server_mail` 存在 `alliance_id` 字段（建议 bigint，默认 0，带索引）。

2. **Redis 依赖**
   - 联盟派发与红点依赖 `REDIS_PLAYER_ALLIANCE_INFO` 的及时性。
   - 若联盟信息更新延迟，会出现“延迟收到联盟邮件/红点”的短暂窗口。

3. **并发重复副本风险**
   - 当前靠 `mail.server_mail_id` 查询去重。
   - 高并发下仍建议数据库唯一约束兜底。

---

## 3. 优化点

### P0
1. 给 `mail(user_id, server_mail_id)` 增加唯一索引，硬性防重。
2. 给 `server_mail(alliance_id, status, expire_time)` 建组合索引，降低联盟邮件查询成本。

### P1
1. 将联盟/全服派发合并为统一派发函数（按 `alliance_id` 分流），减少重复逻辑。
2. 增加派发与红点统计监控（耗时、命中量、失败率）。

### P2
1. 在联盟变更（入盟/退盟）时补偿刷新一次邮件缓存，进一步缩小 Redis 最终一致窗口。

---

## 4. 当前结论

联盟邮件功能已按最新方案实现：
- 单表 `server_mail` 承载全服与联盟邮件；
- 通过 `alliance_id` 区分；
- 玩家联盟归属从 Redis 获取，且仅 `allianceId > 0` 才领取联盟邮件。
