# 荣耀擂台完整实现说明（gloryArena）

更新时间：2026-05-16  
文档定位：以当前代码实现为准，描述完整链路、时间规则、状态模型与已知风险。

## 1. 总体结论

1. 匹配池仅使用 Redis，不做 MySQL 池快照落库。  
2. 赛季推进采用链式规则：`PRE -> FIRST -> SECOND -> POST`。  
3. 若 `FIRST` 开不起来，继续 `PRE`；一旦 `FIRST` 开启，后续按轮次推进到 `SECOND` 和 `POST`。  
4. 轮次为“`00:30` 开启、`00:00` 结束”，`00:00-00:30` 为轮次交接休赛；每日挑战时段独立走配置 `12:00-22:00`。  

## 2. 代码入口与职责

- Controller：`server/logic/gameController/gloryArenaController.go`
- 玩家模型：`server/logic/model/playerGloryArenaModel.go`
- Game 匹配服务：`server/logic/gloryArenaService/gameGloryArenaService.go`
- Rank 构池服务：`server/logic/gloryArenaService/rankBoardGloryArenaPoolService.go`
- Gateway 状态同步：`server/logic/gloryArenaService/gatewayGloryArenaService.go`
- 赛季/轮次计算：`server/logic/logicCommon/gloryArenaLogic.go`
- 战斗结算：`server/logic/raid/gloryArenaInstanceOperator.go`
- 配置：`gameConfig/gloryArena.json`、`gameConfig/constant.json`

职责划分：

- Gateway：计算并写入跨服状态 `ops_state`
- Rank：构建资格池/对手池，响应榜单增量追加
- Game：读取池并做三选一匹配
- Model：玩家进度、状态重置、免费次数与候选缓存
- Raid：战斗胜负结算、发奖、战后重拉候选

## 3. 数据结构

### 3.1 Redis

1. 对手池（ZSET）  
`glory_arena:pool:opponent:{groupVersion}`  
member=`playerId`，score=`战力榜分数`

2. 资格池（SET）  
`glory_arena:pool:qualify:{groupVersion}`  
member=`playerId`

3. 跨服状态（Hash）  
`glory_arena:ops:state`  
field=`serverId`，value=`GloryArenaOpsServerState(JSON)`

4. TTL（常量）
- `OPPONENT`：8 天
- `QUALIFY`：8 天
- `OPS_STATE`：3 天

### 3.2 版本号

- `groupVersion`：`s{seasonType}:ss{groupStartServerID}:c{effectiveSize}:rs{roundStart}:ri{roundIndexInSeason}`
- `seasonVersion`：`s{seasonType}:ss{groupStartServerID}:c{effectiveSize}:st{seasonStart}`

其中 `seasonStart` 基于当前轮次回推到该赛季第 1 轮起点。

### 3.3 玩家表

主表：`player_glory_arena_data`

关键字段：
- `season_id / round_id / pool_version`
- `enroll_count / status`
- `win_count / life / round_best_win / round_got_box / season_total_win`
- `defeated_set / last_match_group / defeated_cache`

快照表：`player_glory_arena_selected_opponent`（3/6/9 选将快照）

## 4. 时间与赛季规则（当前实现）

### 4.1 Gateway 刷新频率

- `GatewayGloryArenaService` 每分钟 tick。
- 每次 tick 都重算并刷新全服 `ops_state`。

### 4.2 赛季推进规则

当前逻辑入口：`CalculateGloryArenaCrossServerResult`

1. 先计算 FIRST 理论开启时间：  
   - 开服第 2 天 `00:30` 为季前首轮开始；  
   - 首轮结束后取“下一个周二 `00:30`”作为 FIRST 开启锚点。  
2. 若当前时间早于该锚点：走开服初始 `PRE` 规则。  
3. 若达到/超过该锚点：进入公共赛季逻辑。

公共赛季逻辑：

1. FIRST 门槛：
   - 必须能组成 2 服配对（`1-2`、`3-4`...）
   - 配对内各服都达到 FIRST 可开启时间
2. FIRST 门槛不满足：继续 `PRE`（按周二/周五窗口滚动）。
3. FIRST 门槛满足后：
   - 以该配对内最大 FIRST 开始时间作为赛季锚点
   - 基于锚点与当前时间计算全局轮次 `globalRound`
   - 赛季映射：
     - `globalRound 1~4` => `FIRST`
     - `globalRound 5~8` => `SECOND`
     - `globalRound >=9` => `POST`

### 4.3 分组规则

- 非 `PRE` 分组要求奇偶相邻结构（`1-2`、`3-4`...）。
- 组规模候选：
  - `FIRST`：2 服
  - `SECOND`：4 -> 2
  - `POST`：8 -> 4 -> 2
- 按目标服所在索引窗口裁剪，不跨窗口跳组。

### 4.4 轮次窗口规则

统一规则：轮次开启时间 `00:30`，轮次结束时间 `00:00`（对应边界日），`00:00-00:30` 为休赛窗口

1. 开服初始 PRE（开服锚点）
   - 第 1 轮：开服第 2 天 `00:30`，持续 3 天
   - 第 1 轮结束：第 4 天 `00:00`
   - 第 2 轮：第 4 天 `00:30` 开始，仅当 `round2End <= 下一个周二00:30` 才允许开启
   - 若第 2 轮不允许开启：返回下一次开启窗口（不返回空窗口）

2. FIRST/SECOND/POST 与“继续 PRE”场景（周滚动双轮）
   - 轮 1：周二 `00:30` ~ 周五 `00:00`
   - 轮 2：周五 `00:30` ~ 下周一 `00:00`
   - 周一等休赛时段：`IsRoundOpen=false`，但返回下一轮窗口
   - 周五 `00:00-00:30`：轮 1 结束到轮 2 开启的休赛窗口

### 4.5 每日挑战时段

- 配置项：`constant.gloryArenaChallengeTime`
- 当前值：`12|22`
- 解释：每日 `12:00`（含）到 `22:00`（不含）
- 判定方式：按当天绝对时间戳比较，不受 `00:30` 轮次边界影响

## 5. Controller 行为

### 5.1 GetHonorArenaInfo

流程：

1. `ForceSyncRoundState`（主动同步一次轮次状态）
2. 若玩家已报名：直接回 `isCompete=1`
3. 读 `ops_state`
4. 轮次未开：直接回包（不查竞技场排名）
5. 查资格池：
   - 有资格：直接回包
   - 无资格：向 Rank 请求竞技场排名后回包
   - 取竞技场榜 `version` 时，复用竞技场版本函数（`00:30` 切版本；`00:00~00:29:59` 仍使用上一版本）

响应字段 `IsFree` 来源：`CanFreeCompete()`

### 5.2 HonorArenaStart（报名）

流程：

1. `ForceSyncRoundState`
2. 校验 `ops_state` 轮次开启
3. 校验资格池成员
4. 校验报名状态与挑战时段
5. 扣票规则：
   - `CanFreeCompete()==1`：免费
   - 否则检查并扣除实例 108 的 `TicketID`
6. 设置报名状态并 `AddEnrollCount(1)`

### 5.3 HonorArenaChallengeList

流程：

1. 校验挑战时段、报名状态、未结束
2. `isRefresh=0` 且本地有 3 个候选时优先复用本地缓存
3. 否则调用 `GetChallengeList` 从池中匹配，并回写 `last_match_group`

### 5.4 StartHonorArenaBattle

流程：

1. 校验轮次开启、挑战时段、报名状态
2. 仅允许选择当前三选一候选里的 `opponentId`
3. 进入荣耀擂台副本场景

## 6. 匹配算法（Game 服务）

`GetChallengeList` 输入：`playerId/winCount/selfPower/defeatedSet/lastOpponents/isRefresh`

流程：

1. `ZCard` 校验池大小（至少 3）
2. 按配置 `Rank` 百分比切片，使用 `ZRevRangeWithScores`（高分在前）
3. 按配置 `Battle` 做战力倍率过滤（`selfPower<=0` 放开）
4. 过滤自己、已击败、上次候选，随机取 3
5. fallback：
   - 放宽 `lastOpponents`
   - `winCount 5~8` 回退到高位区间（当前用配置 `id=9`）
   - `winCount 9~11` 同区间按名次补齐（去战力限制）
   - 最后兜底全池
6. `isRefresh=1` 时优先避免与当前列表重复，样本不足再放开

## 7. Rank 构池与增量

构池触发：

1. Rank 服务启动后每分钟巡检
2. 轮次 key 不存在时构池

构池来源：

- `QUALIFY`：竞技场榜 TopN（`GetArenaRankId + GetGloryArenaEntryRequirement`）
- `OPPONENT`：战力榜 TopN（`GLORY_ARENA_BATTLE_POWER_RANK_ID + GetGloryArenaOpponentRank`）

写入策略（当前实现）：

- `DEL key -> 批量 ZADD/SADD -> EXPIRE`
- 按 `playerId` 去重，重复时保留高分
- 无分布式锁

增量追加：

- 榜单更新时触发 `TryAppendByBattlePowerRankUpdate / TryAppendByArenaRankUpdate`
- 只追加，不删除（掉出 TopN 不移除）

## 8. 玩家状态重置与结算

### 8.1 状态同步与重置

- `Heartbeat` 每分钟读取 `ops_state`
- 关键接口也会主动 `ForceSyncRoundState`
- `season/round/pool_version` 任一变化触发重置：
  - 赛季切换：`ResetSeasonProgress`
  - 轮次或版本切换：`ResetRoundProgress`
- 重置后回到未报名状态，清理旧候选缓存

### 8.2 战斗结算

- 胜利：`win+1` + 胜利奖励
- 失败：`life-1` + 失败掉落
- 结算后尝试重拉候选并刷新 `last_match_group`
- 荣耀擂台结算不改竞技场积分/排名

## 9. 榜单版本规则

- 轮次胜场榜（`pointType=7`）：
  - 分数：`round_best_win`
  - 版本：`pool_version`（`groupVersion`）
- 赛季胜场榜（`pointType=8`）：
  - 分数：`season_total_win`
  - 版本：`seasonVersion`
  - 若本地未同步到 `seasonVersion`，兼容回退为 `groupVersion` 去掉 `rs/ri`
- 季前赛不参与赛季胜场榜

## 10. 近期已修复项（2026-05）

1. 免费报名显示与实际扣票不一致：已统一同源判断（`CanFreeCompete`）。  
2. 挑战时间被错误偏移到 `12:30-22:30`：已修为 `12:00-22:00`。  
3. 周一 `00:00` 轮次状态异常跳变：已修复 PRE 回退计算口径。  
4. 第一轮结束后无“下一次开启时间”：已补回下一窗口。  
5. `ops_state` 仅日刷导致状态滞后：已改为每分钟刷新。  

## 11. 当前遗留风险

1. `GetHonorArenaInfo` 某些异常路径仍存在直接 `return` 未回包风险。  
2. 构池写入仍是 `DEL -> 批量写`，存在短暂读空窗口。  
3. 构池无分布式锁，多实例并发可能重复构池。  
4. 匹配 fallback 仍依赖固定配置 `id=9`。  

---

## 12.1 2026-05-16 重启职责与重建流程补充（Gateway / Rank）

1. `REDIS_GLORY_ARENA_OPS_STATE` 的写入职责在 **Gateway**，不在 Rank。  
   Gateway 启动后会先执行一次全量同步（`syncNow(true)`），并按分钟持续刷新 `ops_state`。

2. Gateway 在写完 `ops_state` 后，会按 `groupVersion` 检查并初始化池 key：  
   - `REDIS_GLORY_ARENA_POOL_OPPONENT`（zset）  
   - `REDIS_GLORY_ARENA_POOL_QUALIFY`（set）  
   若 key 不存在则创建并写入哨兵成员 `__pool_init__`，若存在则续 TTL。  
   该步骤仅保证 key 存在，不负责灌入完整业务成员。

3. Rank 重启后不会重建 `ops_state`。Rank 每分钟巡检时会重新计算跨服状态，并对每个 `groupVersion` 判定是否需要重建池数据。

4. Rank 的重建触发条件（当前实现）：  
   - key 不存在；或  
   - key 为空；或  
   - key 仅包含哨兵 `__pool_init__`（无真实成员）。  
   满足条件时重建对应的 `QUALIFY` / `OPPONENT` 池。

5. 因为 Rank 采用分钟级巡检，Gateway 重启后到 Rank 感知并重建池数据，存在最多约 1 分钟的收敛延迟。

## 12. 2026-05-16 实现同步更新（最新）

1. 竞技场排行榜 version 最终格式固定为：`s{sid}:t{date}`。  
   示例：`s1:t20260511`。

2. 代码严格只识别该格式，不再兼容其它格式。  
   - 生成：`BuildArenaRankVersion`  
   - 解析：`ParseArenaRankVersion`  
   代码位置：`server/logic/logicCommon/rankBoardInterface.go`

3. 荣耀擂台资格池 `glory_arena:pool:qualify:{groupVersion}` 的竞技场数据来源：  
   `common_{arenaRankId}_{version}`，其中 `arenaRankId=6`，`version=s{sid}:t{周一日期}`。  
   例如：serverId=1 在 `rs20260516` 轮次使用表 `common_6_s1:t20260511`。

4. Gateway 启动/同步 `ops_state` 后会检查并初始化每个 `groupVersion` 的池 key：  
   - `glory_arena:pool:opponent:{groupVersion}`（zset）  
   - `glory_arena:pool:qualify:{groupVersion}`（set）  
   代码位置：`server/logic/gloryArenaService/gatewayGloryArenaService.go`

5. 池 key 使用哨兵成员 `__pool_init__` 保活，读取和匹配逻辑会忽略该成员。

6. Rank 构池逻辑已改为按真实成员数（排除 `__pool_init__`）判断是否重建，  
   所以“仅有哨兵成员”的池会自动触发重建。  
   代码位置：`server/logic/gloryArenaService/rankBoardGloryArenaPoolService.go`
