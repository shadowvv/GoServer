# 出征系统实现梳理

本文从 `server/logic/gameController/expeditionController.go` 出发，梳理当前出征系统在服务端的真实实现路径与行为。

## 1. 模块与调用链

- 协议入口：`ExpeditionController.RegisterLogicMessage()`
  - `3221/3222` 出征信息
  - `3223/3224` 开始出征
  - `3225/3226` 立刻完成
  - `3227/3228` 取消
  - `3229/3230` 免费体力
  - `3231/3232` 加速
  - `3233/3234` 切换战场等级
  - `3236/3237` 领奖
  - `3235` 变更推送
- 核心状态与数据：`server/logic/model/expeditionModel.go`
- 配置来源：
  - `gameConfig/cityDispatch.go`（战场、点位、刷新规则、基础掉落）
  - `gameConfig/cityMonster.go`（怪物战力、时长、体力消耗、胜利掉落、每日最大刷新数）
  - `gameConfig/drop.go`（掉落抽取）
  - `gameConfig/constant.go`（体力恢复/免费体力/CD/秒完单价）
- 驱动方式：
  - `scenePlayer.processHeartbeat()` 每帧调用 `playerModel.Heartbeat(now)`
  - `PlayerModel.Heartbeat` 以 `500ms` 为最小间隔调度各子模型 `Heartbeat`
  - 同一轮心跳后执行 `SavePlayerToDB()`，将各模型标记的变更落库

## 2. 数据模型与状态

### 2.1 表结构（逻辑对象）

- `player_expedition_data`（`ExpeditionEntity`）
  - `last_recovery_stamina_time`
  - `daily_free_stamina_times`
  - `last_daily_free_stamina_time`
  - `monster_refresh_count`（json，`monsterId -> 当日已战斗完成次数`）
- `player_expedition_battlefield_data`（`ExpeditionBattlefieldEntity`）
  - `battlefield_id`
  - `battlefield_level`
  - `battlefield_max_level`
  - `battle_point_infos`（json，`pointId -> PointInfo`）
- `player_expedition_slot_data`（`ExpeditionSlotEntity`）
  - `slot_id`
  - `battlefield_id / point_id / start_time / end_time`

### 2.2 点位状态机

`enum.ExpeditionPointStatus`：

- `0 Idle`：空闲，可被派遣
- `1 Busy`：正在派遣中
- `2 Reward`：派遣完成，可领奖

状态流转：

- 开始出征：`Idle -> Busy`
- 时间到自动完成/立刻完成：`Busy -> Reward`
- 取消：`Busy -> Idle`
- 领奖：`Reward -> 删除点位（等待刷新补点）`

## 3. 业务流程（按协议）

### 3.1 出征信息 `ExpeditionInfoReq`

1. 计算可用槽位数：`默认1 + 已购买派遣编队 + VIP特权(最多再+2)`。
2. `CheckExpeditionUnlock(slotNum)`：
   - 初始化已解锁战场数据（按 `cityDispatch(level=1)` 的 unlock 条件）
   - 计算战场当前已解锁最大等级 `BattlefieldMaxLevel`
   - 新战场按当前已解锁最大等级初始化 `BattlefieldLevel/BattlefieldMaxLevel` 与点位
   - 已有战场解锁更高等级时，更新 `battlefield_max_level`，并同步 `battlefield_level` 到新最大等级
   - 最大等级提升时，移除当前 `Idle` 点位，保留 `Busy/Reward` 点位，按新等级配置刷新空点位
   - 确保 `1..slotNum` 的槽位行存在
3. 返回：
   - 战场列表（含点位 monsterId/nextRefreshTime/isReward）
   - 槽位列表（是否进行中、结束时间）
   - 免费体力领取次数与CD结束时间
   - 上次体力恢复时间戳

### 3.2 开始出征 `ExpeditionStartReq`

核心校验与处理：

1. 槽位校验、槽位空闲校验、点位存在且 `Idle` 校验
2. 读取怪物配置 `cityMonster`、编队配置（出征编队类型）
3. 读取战场等级配置（按 `pointInfo.Level`）
4. 扣除体力（`cfg.Energy`）
5. 计算胜负：
   - `totalPower = 编队英雄战力和`
   - `winPercent = (totalPower - 0.8*monsterPower) / (0.2*monsterPower)`
   - `winPercent < 0` 直接失败；否则平方后按概率判赢，超过1则必胜
6. 生成奖励池：
   - 必有战场基础掉落 `Drop1`
   - 胜利额外加怪物掉落 `DropId`
7. 写入状态：
   - 槽位写入 `battlefieldId/pointId/start/end`
   - 点位置 `Busy`，记录 `IsWin` 与 `RewardItem`
8. 返回进行中的槽位信息

### 3.3 立刻完成 `ExpeditionFinishReq`

1. 校验槽位存在且进行中
2. 若 `currentTime > endTime` 直接报错（槽位已结束，不走秒完）
3. 计算剩余分钟向上取整，消耗钻石：
   - `cost = ceil((end-current)/60s) * dispatchAccelerationCost`
4. 扣费后调用 `FinishSlot`：
   - 点位 `Busy -> Reward`
   - 槽位清空
   - 统计 `AddExpeditionNum(1)`
5. 返回槽位清空 + 点位可领奖标记，并上报击杀事件

### 3.4 加速 `ExpeditionSpeedUpReq`

1. 校验槽位进行中
2. 遍历 `items(map[itemId]count)`：
   - 校验道具存在与数量足够
   - `lossTime += itemCfg.TargetId * 1000 * count`
   - 扣道具
3. `slot.EndTime -= lossTime`
4. 返回新的 `overTime`

### 3.5 取消 `ExpeditionCancelReq`

1. 校验槽位进行中
2. 点位 `Busy -> Idle`，清 `IsWin/RewardItem`
3. 槽位清空并返回

### 3.6 领奖 `ExpeditionClaimRewardReq`

1. 校验点位存在且状态为 `Reward`
2. 发奖（`point.RewardItem`），回包包含 `isWin + reward`
3. `ClaimPointReward` 删除点位（等待后续刷新补点）

### 3.7 切换战场等级 `ExpeditionChangeLevelReq`

1. 校验目标 `battlefieldId+level` 配置存在且解锁条件满足
2. 更新 `battlefield_level`
3. 手动切换等级不立即重置现有点位，后续普通刷怪周期按新等级配置生效
4. 最高等级自动解锁不由本协议触发，而是在进入出征信息时的 `CheckExpeditionUnlock` 中处理

### 3.8 免费体力 `ExpeditionClaimFreeStaminaReq`

1. 校验每日次数上限
2. 校验距离上次领取的CD
3. 递增领取计数、更新最后领取时间
4. 发放体力道具 `STAMINA_ITEM_ID`
5. 返回新的CD结束时间

## 4. 心跳驱动逻辑（ExpeditionModel.Heartbeat）

### 4.1 跨天重置

- `daily_free_stamina_times = 0`
- `last_daily_free_stamina_time = 0`
- `monster_refresh_count = {}`（清当日怪物战斗完成计数）

### 4.2 体力自然恢复

- 根据 `staminaRecoveryTime` 计算应恢复次数
- 以 `maximumStamina` 为上限补体力
- 更新 `last_recovery_stamina_time`

### 4.3 派遣与点位状态推进

- 槽位到时（`endTime <= now`）：
  - 点位 `Busy -> Reward`
  - 槽位清空
  - 推送 `ExpeditionChangePush` 槽位变化
- 点位刷新：
  - `Idle` 且过期（`now > nextRefreshTime`）的点位被移除
  - 根据配置补足 `AllMonsterNum`，新怪物写入空点位
  - 怪物选择受“当日已战斗完成次数 + 当前已刷出但未完成的 `Idle/Busy` 数量 + 本轮新刷出数量”与怪物 `Max` 限制
  - 推送 `ExpeditionChangePush` 点位变化

## 5. 怪物刷新与掉落规则

### 5.1 刷怪

- 每个战场等级配置定义：
  - 可刷新点位列表 `MonsterPoint`
  - 同时存在怪物数量 `AllMonsterNum`
  - 候选怪物 `CityMonsterID` 及权重 `Probability`
  - 点位生命周期CD `Cd`
- 每次补点时，`randomMonster`：
  - 使用临时限制计数过滤达到 `cityMonster.Max` 的怪物
  - 临时限制计数 = `monster_refresh_count` 中的当日已战斗完成次数 + 当前 `Idle/Busy` 点位中的同怪数量 + 本轮已抽中的同怪数量
  - `randomMonster` 只更新本轮临时计数，不直接写 `monster_refresh_count`
  - 按权重抽取怪物
  - `nextRefreshTime = now + Cd*1000`

### 5.2 怪物次数计数

- 玩家与怪物战斗完成后，`completeSlot` 将点位置为 `Reward`，并递增 `monster_refresh_count[monsterId]`
- 自动完成和立刻完成都会占用一次怪物 `Max` 次数
- 取消出征不会占用怪物 `Max` 次数
- `Reward` 点位不再作为“当前已刷出但未完成”计数，因为它已经在完成时计入 `monster_refresh_count`
- 例：`cityMonster.Max = 3`，当天已战斗完成 1 次，则同一时刻最多还能刷出 2 只该怪物

### 5.3 掉落

- `Drop(dropId)` = 固定掉落 + 每组权重掉落各抽1个
- 出征奖励：
  - 基础：战场 `Drop1`
  - 胜利追加：怪物 `DropId`

## 6. 当前实现中的注意点（按代码现状）

1. `ClaimPointReward` 仅从内存 map 删除点位，未立即写 `battle_point_infos` 变更；通常依赖后续心跳刷新触发落库。
2. 免费体力CD失败时发送错误消息ID使用了 `CHANGE_LEVEL_RESP`，与接口不一致（应为 `CLAIM_FREE_STAMINA_RESP`）。

