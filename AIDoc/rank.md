# 排行榜实现说明（当前代码）

更新时间：2026-06-06

本文只描述当前实现行为（以 `dhs_server` 现有代码为准）。

---

## 1. 进程与入口

### 1.1 Rank 进程启动

- 启动入口：`server/main/rankBoardMain.go`
- 启动顺序：
  1. `rankBoardPlatform.BootRankBoardPlatform()`
  2. 注册控制器与 Rank 节点消息处理
  3. `rankboardService.InitService()`
  4. `gloryArenaService.InitService()` + `gloryArenaService.StartService()`（荣耀擂台池巡检/重建）

### 1.2 Game 侧入口

- 客户端请求入口：`server/logic/gameController/rankBoardController.go`
  - `RANK_LIST_REQ`（查榜；活动榜支持一次传入多个 `actTargetId`）
  - `RANK_LIKE_REQ`（点赞）
  - `RANK_RECEIVE_BOX_REWARD_REQ`（领取满榜宝箱）
- Rank 节点 RPC 入口（同文件）：
  - `GET_RANK_INFO_REQ`（支持一次请求携带多个 `rankId`）
  - `UPDATE_PLAYER_RANK_INFO`
  - `THUMB_UP_RANK_INFO`
  - `CHECK_RANK_FULL_REQ`
  - `GET_ARENA_RANK_REQ`

---

## 2. 榜单唯一ID与版本规则

实现位置：`server/logic/logicCommon/rankBoardInterface.go`

### 2.1 唯一ID格式

- 常驻榜：
  - 无 version：`common_{rankId}_{serverId}`
  - 有 version：`common_{rankId}_{version}`（当前实现不拼 `serverId`）
- 活动榜：
  - `activity_{actId}_{actRankId}_{version}`

### 2.2 解析规则（当前行为）

- `GetRankRealIdFromUniqueId`：
  - `common_*`：取第 3 段作为 `version`。
  - `activity_*`：要求严格 4 段（`activity_{actId}_{actRankId}_{version}`）。
- `ParseArenaRankVersion` 严格要求：
  - `s{sid}:t{yyyyMMdd}`

### 2.3 竞技场 version 生成

- 函数：`GetArenaRankVersionByTime`
- 规则：
  - 周维度：取本周周一日期
  - 切换点：`00:30`
  - `00:00~00:29:59` 仍归前一日（避免过早切换到新周版本）

### 2.4 荣耀擂台 version

- 轮次版本（groupVersion）示例：`s{season}:ss{startSid}:c{size}:rs{yyyyMMdd}:ri{round}`
- 赛季版本（seasonVersion）示例：`s{season}:ss{startSid}:c{size}:st{yyyyMMdd}`
- 相关实现：`server/logic/logicCommon/gloryArenaLogic.go`

---

## 3. 配置加载与校验

实现：`server/logic/gameConfig/rank.go`  
配置源：`gameConfig/rank.json`（`rank` / `rankAct` / `rankReward`）

### 3.1 核心校验

- `RankType` / `PointType` / `SendRewardType` / `SettlementType` 必须合法。
- `SettlementType`、`RankRewardsId`、`MailId` 三数组长度必须一致。
- `PN > 0`，且 `PNMax >= PN`。
- `MailId`、`RankRewardsId`、活动榜 `AllDropId/LikeDropId` 必须能在配置中找到。

### 3.2 PointType 与 SettlementType 组合约束

- `LEVEL / MAIN_INSTANCE / BATTLE_POWER / TOWER / ALLIANCE_TOTAL_POWER / ARENA / ALLIANCE_ARENA`
  - 禁止搭配 `GLORY_ARENA_ROUND_OVER`、`GLORY_ARENA_SEASON_OVER`
- `GLORY_ARENA_ROUND_WIN_COUNT / ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT`
  - 只能搭配 `GLORY_ARENA_ROUND_OVER`
- `GLORY_ARENA_SEASON_WIN_COUNT`
  - 只能搭配 `GLORY_ARENA_SEASON_OVER`
- 常驻榜（`rank`）不允许 `ACTIVITY_OVER` 结算类型。

---

## 4. 实时榜逻辑

### 4.1 查榜

协议形态：
- `pb.RankListReq.actTargetId` 当前为 `repeated int32`。
- `pb.RankListResp.rankInfo` 当前为 `repeated RankInfo`。
- `rpcPb.ForwardRankBoardMessage.rankId` 当前为 `repeated string`。
- `rpcPb.GetRankInfoResp` 保留单页字段（`rankInfos` / `myRank` / `rankId` 等），并新增 `rankPageInfos` 用于批量查榜返回多页。
- `rpcPb.NotifyUpdateRankInfo` 当前字段语义：
  - `playerId`：玩家 ID，用于个人榜更新，或联盟榜中由玩家映射到联盟。
  - `allianceId`：联盟 ID，仅联盟榜直接更新时传入。
  - `score`：分数。
  - `incrementalUpdate`：是否增量更新。

链路：
1. Game 服校验请求参数。
   - 常驻榜：使用 `commonRankId` 生成 1 个 `rankId`。
   - 活动榜：遍历 `actTargetId` 生成多个 `rankId`，并按生成顺序去重。
2. Game 服通过 `SendMessageToRankBoardBatch` 一次 RPC 到 Rank 节点。
3. RPC 转发消息 `ForwardRankBoardMessage.rankId` 携带多个榜单唯一 ID；路由 hash 当前使用第一个 `rankId`。
4. Rank 节点在 `GetRankListFromRankBoardNode` 中按 `RankBoardSession.RankBoardIds` 顺序逐个查询榜。
5. Rank 节点返回 `rpcPb.GetRankInfoResp.rankPageInfos`。
   - 只有 1 页时，同时回填旧单页字段，兼容旧处理路径。
6. Game 服逐页补全玩家/联盟展示信息后，回包 `pb.RankListResp.rankInfo`。

联盟榜特殊点：
- `ALLIANCE_ARENA`、`ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT` 查询 `MyRank` 时，使用玩家当前联盟ID查询。

### 4.2 更新分数（Rank 节点）

入口：`OnUpdateRankInfoOnRankBoardNode`

- 更新前校验：配置存在、阈值满足、PNMax 限制。
  - `RankThreshold > 0` 时才做阈值拦截；`RankThreshold = 0` 或空配置不拦截。
- 联盟榜映射：
  - 普通个人榜使用 `NotifyUpdateRankInfo.playerId` 作为榜实体 ID。
  - `ALLIANCE_ARENA`：
    - 若 `allianceId > 0`，直接按联盟 ID 更新，并校验联盟基础信息存在、联盟所属服与 `rankId` 中解析出的服务器一致。
    - 若 `allianceId == 0`，必须传 `playerId`，Rank 节点用玩家 Redis 中的 `PlayerAllianceInfo` 映射到联盟 ID。
    - 玩家路径下若玩家无有效联盟信息，直接拒绝更新；不再回退使用玩家 ID 写入联盟榜。
    - 玩家首次以当前联盟参与联盟竞技场榜时，按玩家 Redis 中的 `ArenaScore` 做一次增量补分，并标记 `ArenaJoined=true`。
  - `ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT`：
    - 若 `allianceId > 0`，直接按联盟 ID 更新。
    - 若 `allianceId == 0`，必须传 `playerId`，Rank 节点用玩家 Redis 中的 `PlayerAllianceInfo` 映射到联盟 ID。
    - 玩家路径下若玩家无有效联盟信息，直接拒绝更新；不再回退使用玩家 ID 写入联盟榜。
- 更新后调用：
  - `gloryArenaService.TryAppendByBattlePowerRankUpdate`
  - `gloryArenaService.TryAppendByArenaRankUpdate`
  - 用于把新进 TopN 的玩家补进荣耀擂台池（只追加，不做淘汰）。

### 4.3 点赞

- 仅活动榜支持。
- 当前链路：
  1. 写 `player_rank_thumb_up_log`
  2. 发点赞奖励（`LikeDropId`）
  3. 异步 RPC 更新榜点赞数（`ThumbUp`）

### 4.4 满榜宝箱

- 当前链路：
  1. 写 `player_rank_claim_chest_log`
  2. RPC 到 Rank 校验“是否满榜（人数 >= PN）”
  3. 发全员奖励（`AllDropId`）

### 4.5 进榜实时发奖

- 代码中常驻榜/活动榜“进榜发奖”逻辑均保留注释，当前停用。

---

## 5. 内存榜与持久化

实现：`server/logic/rankboardService/rankBoardInfo.go`

### 5.1 排序规则

- `Score DESC`
- 同分 `EnterTime ASC`

说明：`ThumbUpCount` 不参与排序。

### 5.2 更新行为

- 已在榜：
  - `incrementalUpdate=true`：加分
  - 否则：覆盖分
- 未在榜：
  - 榜未满：直接入榜
  - 榜已满：
    - `resort=false`：不挤榜
    - `resort=true`：仅当新数据优于末位才替换

### 5.3 落库与卸载

- 每榜持久化 loop：默认每 1 分钟。
- 持久化跳过窗口：`00:00~00:05` 与 `23:55~24:00`。
- 空闲卸载：榜 7 天无变更且不脏，停止持久化并从内存 map 移除。

---

## 6. 启动加载策略

实现：`server/logic/rankboardService/rankBoardService.go`

- 启动时按常驻榜配置扫描 `common_{rankId}%` 表并尝试加载。
- 活动榜当前不做启动预加载（按访问懒加载）。
- 懒加载路径：
  - 表存在：读表创建内存榜
  - 表不存在：自动 `CreateRankTable` 后创建空榜

注意（当前代码行为）：
- `ARENA/ALLIANCE_ARENA` 使用 `ParseArenaRankVersionDateInt` 解析 `s{sid}:t{yyyyMMdd}` 周版本。
- 荣耀擂台相关榜使用 `ParseGloryArenaRankVersionDateInt` 解析轮次/赛季版本。
- 版本解析失败会回退为“加载”。
- 已过结算窗口的 `ARENA/ALLIANCE_ARENA`、荣耀擂台相关历史榜，当前代码会计算该榜理论应结算日期；只要任一应结算任务已是 `reward_done`，启动时仍会加载该历史榜。
  - 这会导致部分历史榜在 Rank 重启后继续进入结算检测，但已完成任务会被幂等跳过。
  - 如果理论应结算任务都没有 `reward_done`，当前启动加载判断会返回不加载。

---

## 7. 结算与容灾

主入口：`RankBoardService.tryRecoverAndSettleRanks`

### 7.1 触发节奏

- 心跳每 30 秒。
- 结算扫描至少每 1 分钟一次。
- 结算扫描在 `00:15` 之后执行；`DAY/WEEK` 只生成已完成日期的任务，不生成当天任务。
- `00:00~00:14:59` 内如果算出的结算日期等于当天，会被 `before_daily_settle_window` 跳过。

### 7.2 扫描范围

- 当前实现按**内存已加载榜**遍历处理（`rankBoardInfoMap`），不再每轮扫描 DB 表名。
- `RankBoardService.tryRecoverAndSettleRanks` 会先拷贝内存榜列表，再逐个调用 `RankBoardInfo.tryRecoverAndSettleRanks`。
- 单榜结算参数（`PointType/SettlementType/Reward/Mail/version/serverId`）由 `rankId` 反解析配置得到。
- `SendRewardType == ENTER` 的榜跳过结算链路。

说明：
- 结算覆盖范围取决于该榜是否已加载进内存（启动加载或运行时懒加载）。

### 7.3 结算日期计算

函数：`logicCommon.GetRankSettleTaskSettleDates`

- `DAY`：起始日到“当前日前一日”逐日。
- `WEEK`：起始日到“当前日前一日”逐周日。
- 同时配置 `DAY + WEEK`：周日只算周结算（日结算跳过周日）。
- `ARENA / ALLIANCE_ARENA`：从 `s{sid}:t{yyyyMMdd}` 解析周开始/结束时间，结算日期最多截断到该周周日，不会跨周继续生成旧周榜日结任务。
- `ARENA / ALLIANCE_ARENA` 的起始日还会和服务器开服日期取较大值；当前实现允许从开服当天开始结算，不再强制跳过开服当天。
- 例：当前时间为 `2026-06-06 00:15` 后，日结算最多只会生成到 `20260605`；不会生成 `20260606`。
- 历史说明：`aefd06c8 排行榜问题解决` 已把旧逻辑的“按当天结算”改为“按前一天结算”。旧版本曾可能在 `2026-06-05 00:15` 提前创建 `version=20260605` 的任务；修复后这些老任务仍会因唯一键和 `INSERT IGNORE` 保留并被跳过。
- `GLORY_ARENA_ROUND_OVER`：
  - 从 version 的 `rsYYYYMMDD` 取 round start，结算日 = `start + 3天`
- `GLORY_ARENA_SEASON_OVER`：
  - 从 version 的 `stYYYYMMDD` 取赛季 start
  - 季前赛按规则动态计算结束日
  - 非季前赛固定按 4 轮（`st + 13天`）

### 7.4 任务状态机与幂等

表实体：`rank_settle_task` / `rank_snapshot_info` / `rank_reward_record`

- 任务状态：`pending -> running -> snapshot_done -> reward_done`，失败 `failed`
- 执行流程：
  1. `INSERT IGNORE rank_settle_task`
  2. 非 `snapshot_done` 任务：重建快照（先删旧快照）
     - 优先使用内存榜构建快照批量写入 `rank_snapshot_info`
     - 若内存中不存在该榜，当前直接返回 `rank snapshot not in memory` 并将任务置为 `failed`
     - 若内存榜存在但为空，会写入 0 条快照；后续无 pending 快照时任务会置为 `reward_done`
  3. 按快照逐条发奖
  4. `rank_reward_record` 用 `INSERT IGNORE` + `RowsAffected` 做幂等
  5. 发奖成功后更新 `rank_reward_record` 和 `rank_snapshot_info` 状态
  6. 无 pending 快照时任务置 `reward_done`

幂等与老数据行为：
- `rank_settle_task` 唯一键为 `rank_id + settle_type + version`。
- 已存在任务时，`INSERT IGNORE` 不会更新 `created_at/updated_at/status`。
- 如果读取到任务 `status=3 reward_done`，当前扫描只打印 `reason:task_reward_done` 并跳过，不会重建快照或重复发奖。
- `created_at/updated_at` 是毫秒时间戳 bigint，由 Rank 进程传入的 `currentTime` 写入，不是数据库自动 `DATETIME`。

发奖通道：
- 个人榜：`SendRankBoardRewardMail`
- 联盟榜：`SendRankBoardAllianceRewardMail`

### 7.5 结算日志

当前新增 `[rankSettle]` 关键日志，主要用于确认榜单是否进入检测、为什么不结算、以及任务是否执行：

- `check rankId:... pointType:... settleTypes:... currentDate:... allowToday:...`：榜单进入结算检测。
- `rankId:... settleType:... settleDates:[...]`：本轮计算出的应结算日期。
- `reason:empty_settle_dates`：该结算类型当前没有到期日期，例如周结算未到周日结束后。
- `reason:before_daily_settle_window`：当前仍在 `00:15` 前保护窗口。
- `process rankId:... taskVersion:...`：准备处理某个结算任务。
- `reason:task_reward_done taskId:...`：任务已经发奖完成，本轮幂等跳过。
- `rebuild snapshot ... oldStatus:...`：开始重建结算快照。
- `send rewards ... taskId:...`：开始按快照发奖。

---

## 8. 荣耀擂台池职责分工（Gateway / Rank / Game）

### 8.1 Gateway 侧

实现：`server/logic/gloryArenaService/gatewayGloryArenaService.go`

- 每分钟重算并写入 `REDIS_GLORY_ARENA_OPS_STATE`。
- 按 `groupVersion` 预创建池 key（若不存在）：
  - 对手池（ZSet）写入哨兵 `__pool_init__`
  - 资格池（Set）写入哨兵 `__pool_init__`
- 只做 key 初始化/续 TTL，不灌完整成员。

### 8.2 Rank 侧

实现：`server/logic/gloryArenaService/rankBoardGloryArenaPoolService.go`

- 每分钟巡检 cross state。
- 满足以下任一条件会重建池：
  - key 不存在
  - key 为空
  - key 仅有 `__pool_init__`
- 从排行榜拉取 TopN 构建：
  - 对手池：战力榜
  - 资格池：竞技场榜（当前周 version）

### 8.3 Game 侧

- 读 `ops_state` 与池 key 做资格判定、匹配对手、回包展示。
- 荣耀擂台报名/挑战流程直接依赖池数据。

---

## 9. 关键文件

1. `server/main/rankBoardMain.go`
2. `server/logic/gameController/rankBoardController.go`
3. `server/logic/rankboardService/rankBoardService.go`
4. `server/logic/rankboardService/rankBoardInfo.go`
5. `server/logic/logicCommon/rankBoardInterface.go`
6. `server/logic/logicCommon/arenaCommon.go`
7. `server/logic/logicCommon/gloryArenaLogic.go`
8. `server/logic/gameConfig/rank.go`
9. `server/logic/platform/easyDB/rankEasyDB.go`
10. `server/logic/platform/rankBoardPlatform/rankBoardPlatform.go`
11. `server/logic/gloryArenaService/gatewayGloryArenaService.go`
12. `server/logic/gloryArenaService/rankBoardGloryArenaPoolService.go`
13. `rpcProto/rank.proto`
14. `server/logic/rpcController/gameRpc.go`
15. `server/logic/platform/logicSessionManager/rankBoardSession.go`

---

## 10. 风险与优化点

### 10.1 一致性风险

1. 点赞与宝箱链路均为“先落业务日志/发奖，再异步更新榜或后置校验”，不是原子事务。  
   - 影响：可能出现“奖励已发但榜未更新”或“日志已写但后续校验失败”的不一致。

2. `OnRankLike` 先回成功再发 RPC 更新点赞数，RPC 失败路径仍会回错误。  
   - 影响：同一请求可能出现双回包语义冲突（成功后又错误）。

3. 满榜宝箱是“先写领取日志，再校验满榜”。  
   - 影响：若校验失败，玩家可能已写领取记录而无法再次领取（取决于唯一约束和重试路径）。

4. 结算快照重建为“删旧快照 + 从实时榜重建”，期间榜数据仍可能被更新。  
   - 影响：结算快照与某一时刻严格一致性不足（最终一致，不是强一致快照）。

### 10.2 可用性与性能风险

1. 落库采用“事务内 `DELETE + 批量 INSERT`”全量覆盖。  
   - 影响：已消除空表窗口，但全量写放大依然存在，榜大时 DB 压力较高。

2. `RankBoardService` 仍使用全局互斥锁保护 `rankBoardInfoMap` 的读写。  
   - 影响：虽然结算已改为“拷贝后解锁执行”，但在高并发更新/查询场景仍可能有锁竞争。

3. 持久化在 `00:00~00:05`、`23:55~24:00` 跳过。  
   - 影响：该窗口内脏数据完全依赖后续周期落库，若进程异常退出会放大数据回退。

4. 荣耀擂台池依赖分钟级巡检重建。  
   - 影响：Gateway 重启或池异常后，最多约 1 分钟收敛延迟。

5. 结算只遍历内存榜。  
   - 影响：若某些应结算榜长期未加载进内存，会出现结算延后或遗漏风险（取决于加载策略是否覆盖）。

6. 联盟竞技场榜历史脏数据需要单独清理。  
   - 背景：旧实现中 `NotifyUpdateRankInfo.id` 同时承载玩家 ID/联盟 ID，联盟榜映射失败时可能把玩家 ID 当作联盟 ID 写入 `ALLIANCE_ARENA` 榜。
   - 当前修复只阻止后续污染；已存在于 rank 表或内存榜中的玩家 ID 记录，需要按具体 `rankId` 清理或重建。

### 10.3 优化建议（按优先级）

1. **高优先级：收敛非原子链路**
   - 点赞/宝箱改为“先校验 + 幂等锁 + 发奖 + 最终状态提交”的单事务或 Saga 补偿流程。
   - 明确客户端只返回一次终态结果，避免双回包。

2. **高优先级：强化结算幂等与可恢复性**
   - 为 `rank_snapshot_info` 增加版本化重建标识或批次号，避免“删除后重建”中间态暴露。
   - 发奖失败场景补充可观测字段（失败原因、重试次数、下次重试时间）。

3. **中优先级：优化落库策略**
   - 从“全量覆盖”改为“增量 upsert + 周期全量校正”或“双表切换”方案，降低写放大。
   - 当前已是事务写入，可继续补充分榜并发限流与慢写保护。

4. **中优先级：降低锁竞争**
   - 将全局锁拆为“map 级读写锁 + 单榜锁”，查询尽量走 `RLock`。
   - 热榜与冷榜分层管理，减少同一把锁上的热点争用。

5. **中优先级：提升池收敛速度**
   - 在 Gateway 更新 `ops_state` 后增加轻量事件通知 Rank 主动触发一次重建检查。  
   - 保留分钟巡检作为兜底，形成“事件驱动 + 定时修复”双通道。

6. **低优先级：增强可观测性**
  - 增加核心指标：榜更新 QPS、落库耗时、结算任务状态分布、发奖失败率、池重建耗时与次数。
   - 增加关键审计日志（requestId、rankId、taskId、sourceId）便于追查跨服务链路问题。
