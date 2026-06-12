# 活动服务实现说明

更新时间：2026-06-11

## 1. 职责与整体架构

活动服务分为网关侧和游戏服侧，两侧都实现 `logicCommon.GameActivityServiceInterface`：

- `GatewayActivityService`：活动状态的权威计算方。负责加载配置、判断各服务器活动是否应开启、持久化活动记录、同步 Redis，并通知游戏节点重载。
- `GameActivityService`：活动状态的消费方。从 Redis 加载配置和开启记录，为业务提供活动查询接口。

核心文件：

- `server/logic/activityService/gatewayActivityService.go`
- `server/logic/activityService/gameActivityService.go`
- `server/logic/activityService/activityService.go`
- `server/logic/model/serverActivityConfigModel.go`
- `server/logic/model/serverOpenActivityModel.go`
- `server/logic/gameConfig/activity.go`
- `server/logic/gameController/httpGmController.go`
- `server/logic/backend/backendController.go`

网关启动时在 `gatewayPlatform` 中创建服务并调用 `StartService()`。游戏服和后台服使用 `NewGameActivityService()` 从 Redis 初始化。
HTTP 节点启动时加载全部游戏配置，用于校验 GM 提交的活动配置。

---

## 2. 核心数据模型

### 2.1 活动配置 `ServerActivityConfigEntity`

数据库表：`server_activity_config`

主要原始字段：

- `id`：活动 ID。
- `server_type`：服务器范围类型，`1` 为单服活动，`2` 为跨服活动。
- `server_unit`：允许开启活动的服务器或服务器组。
- `unlock_id`：活动开启所需的服务器解锁条件。
- `attend_unlock_id`：玩家参与活动所需的解锁条件，仅通过接口提供给业务层，网关不开启判断不使用它。
- `event_open` / `event_end`：活动允许开启的绝对时间范围。
- `week_open` / `month_open`：允许开启的星期或每月日期。
- `duration`：每次活动持续时长，单位为小时；可按开启次数配置多个值。
- `settle_time`：从开启时间起算的结算时长，单位为小时。
- `if_first` / `next_id` / `cd`：活动链首节点、下一活动 ID、链路冷却时间。
- `open_loop_num`：最大开启次数。
- `if_block` / `if_block_server`：全局屏蔽或按服务器屏蔽。

`ServerActivityConfigEntity.buildData()` 会将字符串字段解析为运行时字段，例如：

- `ServerUnitInfo`
- `UnlockIds` / `AttendUnlockIds`
- `EventOpenTime` / `EventEndTime`
- `WeekOpenDays` / `MonthOpenDays`
- `DurationTimes`
- `IfBlockServers`

配置模型构建时还会扫描 `next_id` 关系。被其他活动指向且 `if_first != 1` 的配置会标记为 `HasPreActivity=true`，不会作为独立首节点检查。

### 2.2 活动开启记录 `ServerOpenActivityEntity`

数据库表：`server_open_activity`

联合主键：

- `activity_id`
- `version`
- `open_server_id`

状态字段：

- `open_time`：开启时间。
- `settle_time`：结算时间。
- `end_time`：结束时间。
- `open_count`：非数据库字段，由模型根据历史记录动态计算。

`ServerOpenActivityModel.GetAllFinalActivity()` 会按服务器和活动 ID 聚合历史记录，返回每个活动最后一次记录，并计算其累计开启次数。

---

## 3. 配置与状态加载

### 3.1 网关配置加载

`NewGatewayActivityService()` 调用 `loadActivityConfig(env)`：

- 非本地、开发、测试环境：从 EasyDB 加载全部 `server_activity_config`。
- 本地、开发、测试环境：从游戏原始配置 `gameConfig.GetAllOriginalActivityCfg()` 构建服务端配置，并写入 EasyDB。

加载完成后，网关把全部活动配置序列化到 Redis：

- Key：`activity:config:`
- Value：全部 `ServerActivityConfigEntity` 的 JSON 数组。

### 3.2 网关开启记录加载

`loadOpenActivity()` 从 EasyDB 加载全部 `server_open_activity`，再按 `open_server_id` 分组，初始化 `ServerOpenActivityModel`。

### 3.3 游戏服加载

游戏服从 Redis 加载：

- 配置：读取 `activity:config:`。
- 开启记录：扫描 `activity:open:*`，每个服务器对应一个 Hash。
- Hash field：活动 ID。
- Hash value：该活动最后一次开启记录的 JSON。

`GameActivityService.Reload()` 会分别加载配置和开启记录。任一部分加载失败时只记录错误并保留该部分原有的内存快照；成功加载的另一部分仍可独立更新，避免 Redis 临时故障清空游戏节点的活动数据。

### 3.4 HTTP 节点配置加载与活动配置校验

HTTP 节点在 `BootHttpPlatform()` 中调用 `gameConfig.LoadAllConfig()`，加载活动配置校验所依赖的游戏配置，包括解锁配置。

`gameConfig.CheckActivityConfig()` 是活动配置的公共校验方法，由以下流程复用：

- `ActivityCfgLoader.checkData()`：加载本地活动配置时校验。
- 网关 `loadActivityConfig()`：从 EasyDB 加载活动配置后校验。
- HTTP 节点 `handleGmEditServerActivityConfig()`：GM 编辑活动配置写库前校验。

公共校验包括活动 ID、服务器类型、跨服范围、开启与参与解锁条件、绝对时间、星期开启日和月份开启日。

---

## 4. 网关启动、心跳与重载

`StartService()` 的执行流程：

1. 调用 `initAllActivity()`，立即全量检查活动。
2. 保存最终活动状态到 Redis。
3. 广播 `RPC_OPERATION_RELOAD_ACTIVITY`，通知游戏节点重载。
4. 启动默认间隔为 5 秒的 ticker。

每次心跳：

1. 若 `activityConfigChanged=true`，重新加载活动配置并覆盖 Redis 配置。
2. 调用 `checkActivityChange(configChanged)` 检查活动状态变化。
3. 若存在变化，将变化记录写入 EasyDB。
4. 若活动状态存在变化，将各服务器最后一次活动状态写入 Redis。
5. 若配置已成功重载或活动状态存在变化，广播 `RPC_OPERATION_RELOAD_ACTIVITY`。

每次心跳由独立的 panic 恢复边界保护。单次心跳发生 panic 时会记录错误并终止当前心跳，ticker goroutine 会继续执行后续心跳。

GM 修改活动配置的流程：

1. Backend 节点校验 Token 和编辑活动权限。
2. Backend 使用同步 HTTP 请求转发到 HTTP 节点，并等待 HTTP 节点返回结果。
3. HTTP 节点使用 `gameConfig.CheckActivityConfig()` 校验活动配置。
4. 校验通过后，HTTP 节点写入 EasyDB。
5. 写入成功后，HTTP 节点向网关发送 `RPC_OPERATION_RELOAD_ACTIVITY_CONFIG`。
6. Backend 将 HTTP 节点返回的 `Code`、`Msg` 和 `Data` 透传给管理端。

网关的 `Reload()` 只设置原子标记，实际重载在下一次 ticker 中执行。网关收到 `RPC_OPERATION_RELOAD_ACTIVITY_CONFIG` 后不会立即转发给游戏节点；下一次心跳成功加载配置并写入 Redis 后，才广播 `RPC_OPERATION_RELOAD_ACTIVITY`。因此，即使配置修改没有引起活动状态变化，游戏节点也会在 Redis 配置更新完成后重新加载。校验或写库失败时，HTTP 节点不会通知网关重载。

---

## 5. 活动刷新算法

`refreshAllActivity()` 分两阶段执行。

### 5.1 检查已有活动与活动链

遍历各服务器最后一次活动记录：

- 配置刚重载时，调用 `checkConfigChange()` 尝试更新尚未生效的开启时间、尚未结束的结束时间和尚未结算的结算时间。
- 当前活动存在 `next_id`、已经结束且满足 `cd` 后，检查下一活动。
- 下一活动的检查不再验证普通开启条件，只验证屏蔽、循环次数和服务器范围。
- 历史活动记录所属服务器已不在当前开放服务器列表时，`checkActivityOpen()` 会直接跳过该服务器，避免活动链推进访问空服务器对象。

因此，活动链后继节点的开启主要由前置活动结束和冷却时间驱动，不重新校验 `unlock_id`、绝对时间、星期或月份条件。

### 5.2 检查所有首节点活动

跳过 `HasPreActivity=true` 的配置，再逐个服务器检查：

- 从未开启：以 `openCount=0` 尝试开启。
- 已开启但已结束：使用历史累计开启次数尝试下一轮。
- 仍未结束：不做处理。

---

## 6. 活动开启条件

`checkActivityOpen()` 按以下顺序判断。

### 6.1 屏蔽条件

- `if_block == 1`：活动全局不开放。
- 当前服务器位于 `if_block_server`：该服务器不开放。

### 6.2 普通开启条件

仅首节点检查或独立活动检查时执行：

- 所有 `unlock_id` 都必须通过 `UnlockService.CheckServerInfoUnlock()`。
- 当前时间不能早于 `event_open`。
- 当前时间不能晚于 `event_end`。
- 配置 `week_open` 时，当前星期必须命中。
- 未配置 `week_open` 但配置 `month_open` 时，当前日期必须命中。

星期条件优先于月份条件，两者同时配置时只检查星期。

### 6.3 开启次数

以下任一条件成立时，活动按循环活动处理：

- 配置星期开启日。
- 配置月份开启日。
- `open_loop_num != 0`。
- 活动存在前置活动。

循环活动要求 `openCount < open_loop_num`，通过后将开启次数加一。其他活动最多开启一次。

### 6.4 服务器范围与版本

`server_unit` 使用分号分隔多个服务器分组。当前服务器命中任意一个分组即可开启活动；未命中任何分组时不开启。

`server_type=Single` 时，版本中的索引使用服务器 ID。

`server_type=Multi` 时，版本中的索引使用实际命中服务器组在 `ServerUnitVector` 中的下标，使同组服务器得到相同活动版本。

版本格式由 `getActivityVersion()` 生成：

```text
d{YYYYMMDD}s{服务器ID或服务器组下标}c{开启次数}
```

### 6.5 时间计算

- `settle_time == 0`：结算时间为 `math.MaxInt64`。
- 否则：`OpenTime + settle_time * 1小时`。
- 首次开启和配置重载均只根据 `settle_time` 计算结算时间，与 `duration` 是否为零无关。
- `duration` 可配置多个值，按开启次数选取；只有一个值时每次复用，开启次数超过配置数量时复用最后一个值。
- `duration == 0`：结束时间为 `math.MaxInt64`。
- 否则：`OpenTime + duration * 1小时`。

---

## 7. 持久化与节点同步

活动变化由 `ServerOpenActivityModel.OpenActivity()` 写入 EasyDB：

- 相同 `activity_id + version + open_server_id`：保存已有记录。
- 新版本：创建新的历史记录。

网关通过 Redis Pipeline 写入每个服务器最后一次活动状态：

```text
Key:   activity:open:{serverId}
Field: {activityId}
Value: ServerOpenActivityEntity JSON
```

活动状态写入后广播 `RPC_OPERATION_RELOAD_ACTIVITY`。配置成功重载并写入 Redis 后，即使活动状态没有变化，也会广播该操作。游戏节点收到广播后调用 `GameActivityService.Reload()`，分别重新加载 Redis 中的配置和活动记录；加载失败的部分保留原有内存快照。

---

## 8. 对外查询接口

`GameActivityServiceInterface` 提供：

- `IsActivitySettled(serverId, activityId, version)`：指定版本达到结算时间或结束时间时返回 `true`；记录不存在时也返回 `true`。
- `GetAllOpenActivityByServerId(serverId)`：返回 `open_time <= now < end_time` 的活动。
- `IsActivityOpen(serverId, activityId)`：返回当前有效的活动记录，并检查配置屏蔽状态。
- `GetActivityConfig(activityId)`：返回活动配置。
- `Reload()`：重新加载活动数据；配置或开启记录加载失败时保留对应的原有内存快照。

---

## 9. 当前实现的关键边界与风险

1. **Redis 活动状态采用只写入、不清理策略**
   - `saveActivityToRedis()` 使用 `HSet` 覆盖当前字段，不删除已有活动字段和服务器 Key。
   - 线上活动配置不会被直接删除；停止活动时必须通过 `if_block` 或 `if_block_server` 屏蔽。因此旧 Redis 字段不会造成活动重新开放或查询结果错误，当前无需增加清理逻辑。
   - 该结论依赖上述线上运维约束。如果未来允许删除活动配置或服务器，需要重新设计 Redis 快照清理与读取一致性方案，避免游戏节点重载时读到空或不完整状态。

2. **活动服务测试仍缺少外部依赖集成验证**
   - 当前已覆盖服务器分组匹配、配置重载结算时间、多段持续时长热更新、查询接口屏蔽一致性、心跳 panic 恢复、开启次数限制、活动链首次推进、活动链冷却时间、活动链最大开启次数、历史服务器缺失保护、配置重载广播条件，以及 Redis 加载失败时保留旧快照。
   - Redis 同步、EasyDB 持久化、RPC 广播，以及解锁与日期条件组合仍缺少自动化集成验证。

### 9.1 已实现的一致性保护

- 配置重载通知不会在 Redis 配置更新前转发给游戏节点；网关写入新配置后统一广播重载。
- 历史活动所属服务器已不存在时，活动链推进会跳过该服务器，不会因访问空服务器对象导致心跳 panic。
- 游戏节点从 Redis 重载配置或开启记录失败时，会保留对应的旧内存快照。

---

## 10. 关键代码定位

- 网关活动调度：`server/logic/activityService/gatewayActivityService.go`
- 服务器分组匹配测试：`server/logic/activityService/gatewayActivityService_test.go`
- 游戏服活动读取：`server/logic/activityService/gameActivityService.go`
- 活动版本生成：`server/logic/activityService/activityService.go`
- 活动配置公共校验：`server/logic/gameConfig/activity.go`
- GM 活动编辑入口与结果透传：`server/logic/backend/backendController.go`
- HTTP 节点活动校验、写库与重载通知：`server/logic/gameController/httpGmController.go`
- HTTP 节点配置加载：`server/logic/platform/httpPlatform/httpPlatform.go`
- 活动配置解析：`server/logic/model/serverActivityConfigModel.go`
- 活动历史与查询：`server/logic/model/serverOpenActivityModel.go`
- 服务接口：`server/logic/logicCommon/gameActivityInterface.go`
- 网关启动接线：`server/logic/platform/gatewayPlatform/gatewayPlatform.go`
- 游戏节点重载 RPC：`server/logic/rpcController/gameRpc.go`
- Redis Key：`server/enum/redisKeyEnum.go`
