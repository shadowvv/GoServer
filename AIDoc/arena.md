# 竞技场实现说明（基于 `ArenaController` 真实代码）

更新时间：2026-05-16

## 1. 入口与职责边界

核心入口在：
- `server/logic/gameController/arenaController.go`

`ArenaController` 注册了 5 个协议处理：
- `GET_ARENA_INFO_REQ` -> `GetArenaInfo`
- `Get_CHALLENGE_LIST_REQ` -> `GetChallengeList`
- `REFRESH_CHALLENGE_LIST_REQ` -> `RefreshChallengeList`
- `Get_Arena_LOG_REQ` -> `GetArenaLog`
- `START_ARENA_BATTLE_REQ` -> `StartArenaBattle`

这 5 个接口都绑定 `FUNCTION_ID_ARENA`，功能开关由统一框架拦截。

---

## 2. 数据模型与存储

### 2.1 玩家竞技场主数据
文件：`server/logic/model/playerArenaModel.go`

`player_arena_data`（`PlayerArenaEntity`）：
- `user_id`
- `challenge_list`：缓存挑战列表（json）
- `challenge_num`：当日挑战次数
- `refresh_count`：当日刷新次数
- `score`：当前竞技场积分
- `version`：赛季版本（格式 `s{serverId}:t{YYYYMMDD}`，按“本周周一日期”编码）
- `last_reward_time`：最近一次发放周榜奖励时间

### 2.2 竞技场对战日志
`player_arena_log`（`PlayerArenaLogEntity`）：
- `battle_id`
- `attack_user_id` / `defend_user_id`
- `attack_score_change` / `defend_score_change`
- `defend_resolved`：防守积分是否已结算到被挑战者
- `challenge_time`
- `version`

---

## 3. ArenaController 接口行为

## 3.1 `GetArenaInfo`
- 返回 `score`
- 返回 `refreshTime = 日免费刷新上限 - 已刷新次数`（最低 0）
- 返回 `freeChallengeTime = 日免费挑战上限 - 已挑战次数`（最低 0）

说明：协议字段名是 `refreshTime/freeChallengeTime`，实际语义是“剩余次数”，不是时间戳。

## 3.2 `GetChallengeList`
- 先读 `PlayerArenaModel.GetChallengeList()`
- 若为空则 `RefreshChallengeList()`
- 输出 5 个 `OpponentBasicInfo`
  - 真人：从 `logicCommon.GetPlayerBasicInfo` 读取昵称、防守战力、积分、单位
  - 机器人：只标 `isRobot=1`（基础信息走机器人配置）

## 3.3 `RefreshChallengeList`
- 结算保护：当天 `00:00 ~ 00:30`（`ARENA_RESOLVE_TIME = 30min`）禁止刷新，返回 `ARENA_IS_SETTLING`
- 刷新消耗：
  - 先扣免费次数（`arenaDailyFreeRefreshAttempts`）
  - 超出后扣钻（`refreshDiamondConsumptionQuantity`）
- 调用 `PlayerArenaModel.RefreshChallengeList()` 重建列表并返回

## 3.4 `GetArenaLog`
- 查询当前玩家作为“防守方”的近 30 条日志
- 返回字段：
  - 对手ID/昵称
  - `changeScore`（这里取 `attack_score_change`）
  - 对战时间

注意：这里是“被挑战记录视角”，`changeScore` 当前展示的是进攻方变更值。

## 3.5 `StartArenaBattle`
- 结算保护：`00:00~00:30` 禁止开战，返回 `ARENA_IS_SETTLING`
- 参数校验：
  - pb 反序列化
  - 竞技场副本配置存在（`GetArenaInstanceCfg`）
  - 不能重复进入竞技场实例
  - 必须当前在主线实例中（非主线不允许直接切）
- 挑战消耗：
  - 当日免费挑战用尽后，扣竞技场门票（`instance.ticketID`）
- 目标校验：
  - 复仇：目标必须存在于日志中
  - 普通挑战：目标必须在当前挑战列表中，并识别是否机器人
- 构造 `PlayerInstanceRaid`：
  - `InstanceID = ARENA_INSTANCE_ID`
  - `TargetTd = opponentId`
  - `IsRobot = opponent.IsRobot`
- 调用链：
  - `raid.CanEnterInstanceStage`
  - `raid.OnLeaveRaid`
  - `raid.BuildInstanceRaid`
  - `raid.EnterScene`（带登录互斥锁）
- 成功后：
  - 累计总挑战次数 `StaticData.AddArenaChallengeTimes(1)`（用于新手保护匹配）
  - 累计当日挑战次数 `PlayerArenaModel.AddChallengeTime(1)`
  - 返回 `StartArenaBattleResp{SceneBasicInfo}`

---

## 4. 挑战列表算法（`PlayerArenaModel.RefreshChallengeList`）

目标：产出 5 个对手，且尽量包含比自己低分对手，并兼顾真人与机器人。

## 4.1 新手保护（总挑战次数 < 3）
- 只匹配机器人：
  - 优先 `[score-30, score]` / `[score-50, score]`
  - 再补 `[score, score+50]`
  - 不足再从机器人总榜补到 5 个

## 4.2 普通匹配
- 先按积分段从 Redis ZSet 拉：
  - `[score-30, score]`，不足再 `[score-50, score]`
- 不足则按排名窗口拉真人（自己上下约 20 名）
- 如果凑满 5 个但没有低分对手，会替换掉一个，补 1 个低分机器人
- 最后不足 5 时再补机器人（区间机器人 -> 全榜机器人）

## 4.3 排名缓存键
- `arena_score_info:{serverId}:{version}`
- `version` = `s{serverId}:t{YYYYMMDD}`（其中日期为本周周一）

---

## 5. 对战构建与结算（`raid/arenaInstanceOperator.go`）

竞技场在 raid 层有专用 operator：`ArenaInstanceRaid`。

## 5.1 Build 阶段
- 机器人对手：`BuildRaidWithBot`
  - 从 `gameConfig/arena.json` 读取 `bot.arenaLineup`
- 真人对手：`BuildRaidWithPlayer`
  - 从 `logicCommon.GetPlayerBasicInfo(targetId)` 读取防守阵容快照
  - 使用防守阵容的英雄属性/技能构建怪物模板
  - 敌方合体技由 `GetEnemyComberSkillIds` 计算

## 5.2 进攻方胜利（`OnRaidEnd`）
- 根据进攻方当前分段配置（`pointsParameters`）计算加分：
  - `add = WinBase + Coeff1 * (opponentScore - selfScore) / 10000`
  - 最少 +1
- `PlayerArenaModel.AddScore(add)`：更新玩家积分 + Redis积分榜 + RankBoard积分榜
- 记录操作日志 `ARENA_OPER_CHALLENGE`
- 若对手是真人：
  - 计算对手扣分（`ChangeOpponentScore(..., win=false)`）
  - 写入 `player_arena_log`，`defend_resolved=0`
- 刷新挑战列表
- 发放胜利掉落（`arenaVictoryReward`）
- 推送 `PushStageBattleWin`，`arenaChangePoint=add`

## 5.3 进攻方失败（`OnBattlePlayerDead`）
- 计算扣分：
  - `lose = LoseBase - Coeff2 * (opponentScore - selfScore) / 10000`
  - 最少 -1（最终是 `pointChange = -lose`）
- 其余流程与胜利对称：
  - 更新自己积分
  - 真人对手加分（`ChangeOpponentScore(..., win=true)`）
  - 写 battle log（`defend_resolved=0`）
  - 刷新挑战列表
  - 发放失败掉落（`arenaDefeatReward`）
  - 推送 `arenaChangePoint`

## 5.4 被挑战者积分的“延迟结算”
- 实时对战结束时只记录 `player_arena_log.defend_score_change`
- 被挑战者在自己的 `PlayerArenaModel.Heartbeat` 中轮询未结算日志：
  - 汇总 `defend_score_change`
  - 批量置 `defend_resolved=1`
  - 一次性 `AddScore(total)`
- 轮询间隔自适应：
  - 初始 10s，无变更逐步倍增，最大 10min
  - 有新日志后恢复 10s

---

## 6. 日/周周期逻辑

## 6.1 每日重置
`PlayerArenaModel.Heartbeat(passDay>0)`：
- 清零 `refresh_count`
- 清零 `challenge_num`
- 非竞技场战斗中时自动刷新挑战列表

## 6.2 每周赛季版本
当时间达到当天 00:30（结算窗后）：
- `version = s{serverId}:t{本周周一日期}`
- 若版本变化：
  - 积分重置为 `arenaSeasonInitialPoints`
  - 更新排名缓存键（新版本）
  - 记录 `ARENA_OPER_INIT`
  - 刷新挑战列表

补充边界：
- `00:00~00:30` 内主动玩法入口已封禁（不能刷新/不能开战），因此不会产生主动战斗写分。
- 被挑战方“延迟结算”链路仍可能在该窗口执行，但使用玩家当前 `version` 入账，不会提前切到新周版本。

---

## 7. 排行与奖励链路

## 7.1 两套榜单数据
- Redis ZSet：`arena_score_info:{server}:{version}`（用于匹配）
- RankBoard 服务榜：用于排行榜展示、备份、奖励发放

## 7.2 RankBoard 周备份
`rankboardService.checkNeedBackup` 对竞技场榜执行备份：
- 将当前榜快照插入 `backupRankInfo`
- 字段包含 `rank/score/backup_time/isReward`

## 7.3 奖励发放触发
`PlayerArenaModel.Heartbeat` 在跨天后触发 `RPC_MESSAGE_GET_MY_RANK_REQ` 查询历史榜。

回包 `BackMyRankFromRankBoardNode`：
- 遍历未领奖条目
- 按是否周一选择不同奖励档位与邮件模板
- 发送奖励邮件
- 更新 `last_reward_time`

---

## 8. 配置项总览（改造时优先关注）

## 8.1 `constant.json`
- `arenaSeasonInitialPoints`
- `arenaDailyFreeRefreshAttempts`
- `refreshDiamondConsumptionQuantity`
- `arenaDailyFreeChallengeAttempts`
- `arenaVictoryReward`
- `arenaDefeatReward`

## 8.2 `arena.json`
- `bot`：机器人阵容、机器人积分、机器人战力
- `pointsParameters`：分段积分公式参数（`winBase/loseBase/coeff1/coeff2/pointsRange`）

## 8.3 `instance.json`
- `ARENA_INSTANCE_ID(102)` 对应门票配置 `ticketID`

---

## 9. 改造建议（针对当前实现）

1. `GetArenaLog` 当前展示 `attack_score_change`，若要展示“我方积分变化”需改字段映射。  
2. `defend_score_change` 是延迟到账，UI 侧要明确“被挑战积分可能稍后刷新”。  
3. 结算窗是固定 `00:00~00:30`，若未来改为可配置，应将 `ARENA_RESOLVE_TIME` 外置。  
4. `StartArenaBattle` 只允许从主线实例进入竞技场，跨玩法切换需求需要改这层限制。  
5. 当前匹配依赖 `PlayerBasicInfo` Redis 快照，若快照延迟会影响展示/战力准确性。  

---

## 10. 关键代码定位

- 控制器入口：`server/logic/gameController/arenaController.go`
- 竞技场主模型：`server/logic/model/playerArenaModel.go`
- 竞技场副本结算：`server/logic/raid/arenaInstanceOperator.go`
- raid 分发入口：`server/logic/raid/raid.go`
- 排行封装：`server/logic/logicCommon/rankBoardInterface.go`
- 周榜备份：`server/logic/rankboardService/rankBoardService.go`
- 历史奖励发放：`server/logic/gameController/rankBoardController.go`
