# Alliance 模块开发文档（代码核对版）

本文档按当前代码实现核对联盟（alliance）模块行为、约定、风险与优化建议。  
核对日期：`2026-05-13`。

核对范围（核心文件）：
- `server/logic/gameController/allianceController.go`
- `server/logic/socialService/allianceService.go`
- `server/logic/socialService/allianceManager.go`
- `server/logic/socialService/socialAllianceModel.go`
- `server/logic/platform/dispatcherService/allianceMessageProcessor.go`
- `server/logic/platform/logicSessionManager/allianceSession.go`
- `server/logic/rpcController/{gameRpc.go,socialRpc.go,gatewayRpc.go}`
- `server/enum/{allianceEnum.go,redisKeyEnum.go}`
- `server/logic/model/allianceModel.go`

## 1. 当前实现概览

### 1.1 节点与链路

- Game 接收客户端联盟消息并转发到 Social：
  - 入口与回包：`server/logic/gameController/allianceController.go`
  - 发往 Social：`server/logic/rpcController/gameRpc.go` (`SendMessageToSocial`)
- Social 按 `allianceId` 分片串行处理（同联盟同 processor）：
  - `server/logic/platform/dispatcherService/allianceMessageProcessor.go`
  - `server/logic/platform/logicSessionManager/allianceSession.go`
- Social 业务核心：
  - `AllianceService`：`server/logic/socialService/allianceService.go`
  - `AllianceManager`：`server/logic/socialService/allianceManager.go`
  - `AllianceModel`：`server/logic/socialService/socialAllianceModel.go`
- 联盟变更通知链路（对客户端）：
  - Social 调 `NotifyAllianceOperationToGateway`
  - Gateway 下发 `PUSH_ALLIANCE_CHANGE`
  - 当前 `oper`：`ENTER`、`KICKOUT`、`NEW_APPLY`
  - 代码：`server/logic/rpcController/socialRpc.go`、`server/logic/rpcController/gatewayRpc.go`
- 另有 Social -> Game 内部推送消息 `RPC_MESSAGE_PUSH_ALLIANCE_CHANGED`，但 Game 回调 `BackAllianceChangedFromSocialNode` 目前为空实现。

### 1.2 数据模型

- MySQL：
  - `alliance`
  - `alliance_member`（`user_id` 主键，单角色仅在一个联盟）
- 代码实体：
  - `AllianceEntity.CityLevelCondition`（列：`city_level_condition`）
  - `AllianceMemberEntity`
  - 代码：`server/logic/model/allianceModel.go`
- Redis：
  - `alliance:basic:{allianceId}`
  - `alliance:server:{serverId}`（ZSet，`score=allianceTotalPower`）
  - `alliance:name:index:{serverId}`（Hash，`name -> allianceId`）
  - `alliance:apply:list:{allianceId}`（ZSet，按申请时间）
  - `player:AllianceInfo:{userId}`（玩家联盟归属缓存）

### 1.3 客户端消息（已接入）

1. `CREATE_ALLIANCE_REQ/RESP`
2. `CHANGE_ALLIANCE_BASIC_INFO_REQ/RESP`
3. `GET_SERVER_ALLIANCE_BASIC_INFO_REQ/RESP`
4. `APPLY_ALLIANCE_REQ/RESP`
5. `GET_APPLY_LIST_REQ/RESP`
6. `GET_PLAYER_ALLIANCE_INFO_REQ/RESP`
7. `APPROVE_ALLIANCE_APPLY_REQ/RESP`
8. `KICK_MEMBER_OUT_ALLIANCE_REQ/RESP`
9. `CHANGE_MEMBER_POSITION_REQ/RESP`
10. `LEAVE_ALLIANCE_REQ/RESP`
11. `PUSH_ALLIANCE_CHANGE`（Gateway 主动推送）

## 2. 设计约定（按代码现状）

1. 申请列表由 Redis 管理，不落 Social DB。  
   - 申请制联盟：Game 先写 `alliance:apply:list:{allianceId}`，再发 Social 触发管理层 `NEW_APPLY` 推送。  
   - 自由加入联盟：Game 发 Social，由 Social 执行入盟并写 `alliance_member`。  

2. 玩家联盟归属以 `player:AllianceInfo:{userId}` 为主判定。  
   - Game 侧入口校验统一走 `logicCommon.GetPlayerAllianceInfoFromRedis`。  
   - `BackCreateAllianceFromSocialNode` / `BackLeaveAllianceFromSocialNode` 不写 `AllianceId` 本地字段。  
   - `BackAllianceChangedFromSocialNode` 为空实现。  

3. Social 侧成员归属校验已接入。  
   - 需要“玩家在盟内”的 RPC，先做 `ensureAllianceMembershipOnSocialNode`。  
   - 校验失败返回 `ERROR_CODE_ALLIANCE_NOT_IN_ALLIANCE`，并重置 `player:AllianceInfo` 为退出态。  

4. 联盟名搜索为 Redis 索引“最佳努力”模型。  
   - `alliance:name:index:{serverId}` 命中后，会回读 `alliance:basic:{allianceId}` 做 server/name 二次校验。  
   - 重名判定仍由 Social 内存索引 + DB 查询兜底。  

5. Social 启动会重建联盟缓存。  
   - 重建 `alliance:*` 索引。  
   - 清理并重建全部 `player:AllianceInfo:*`。  

6. 心跳逻辑在 Social tick 执行。  
   - 检查间隔：`30m`（`AllianceHeartbeatCheckInterval`）  
   - 自动解散阈值：全员离线超过 `7d`（`AllianceAutoDissolveOfflineThreshold`）  
   - 会长自动转移：副会长优先（离线 <= `3d`），否则普通成员（离线 < `7d`）  

## 3. 需求核对结果（本轮重点）

### 3.1 创建联盟宣言规则

目标需求：
- 玩家填了宣言：校验合法性。
- 玩家不填宣言：直接创建。

核对结果：**已实现**。  
代码：`CreateAllianceHandler`（`server/logic/gameController/allianceController.go`）。

## 4. 风险清单（当前）

### P1（高优先级）

1. 修改联盟信息时，数值门槛字段仍“无更新位直接覆盖”。  
   - 位置：`ChangeAllianceBasicInfoHandler` + `AllianceModel.ChangeAllianceBasicInfo`  
   - 问题：`applyType/powerApplyCondition/cityLevel` 没有 `Update*` 标志，可能被默认 `0` 覆盖。  

2. 改盟名回包失败时无条件返还改名道具，存在道具异常增发风险。  
   - 位置：`BackChangeAllianceBasicInfoFromSocialNode`  
   - 问题：只要 Social 回错就执行 `AddItem(changeAllianceNameItem)`，不区分本次是否真的消耗了改名道具。  

3. 创建联盟 / 改盟名均先扣道具再发 RPC；若发往 Social 失败，道具不返还。  
   - 位置：`CreateAllianceHandler`、`ChangeAllianceBasicInfoHandler`  
   - 问题：`SendMessageToSocial` 失败路径直接回错，没有补偿。  

4. 申请制入盟存在“同请求双响应”风险。  
   - 位置：`ApplyAllianceHandler`  
   - 问题：申请制先回 `APPLY_ALLIANCE_RESP` 成功，再调用 `sendApplyAllianceToSocial`；若 RPC 发送失败，会再次回错误。  

5. `BackAllianceChangedFromSocialNode` 空实现，缺少 Game 侧兜底同步。  
   - 位置：`BackAllianceChangedFromSocialNode`  
   - 影响：当前主要依赖 Gateway 推送；当推送异常时，Game 本地无补偿逻辑。  

### P2（中优先级）

1. `APPLY_ALLIANCE_REQ` 部分失败路径错误码语义不准确。  
   - 位置：`ApplyAllianceHandler`  
   - 现状：读取联盟基础信息失败时返回 `ERROR_CODE_ALLIANCE_ALREADY_IN_ALLIANCE`。  

2. `ChangeAllianceBasicInfo` 未校验 `applyType` 枚举合法性。  
   - 位置：`AllianceModel.ChangeAllianceBasicInfo`  
   - 影响：异常客户端可能写入非法入盟类型值。  

3. `*_BATCH_*` RPC 命名与当前“单人审批”语义不一致。  
   - 位置：`RPC_MESSAGE_APPROVE_ALLIANCE_APPLY_BATCH_REQ/RESP`  
   - 影响：接口可读性和维护性下降。  

## 5. 优化建议

1. 为 `applyType/powerApplyCondition/cityLevel` 增加显式更新位（或改为 pointer 字段）。  
2. 改盟名失败返还道具前，增加“是否在本次已扣除”的判定。  
3. 对“先扣道具后 RPC”的路径增加发送失败补偿（本地回滚或异步补偿）。  
4. 申请制入盟统一为单响应语义（成功/失败只回一次）。  
5. 实现 `BackAllianceChangedFromSocialNode` 或移除该链路并统一走 Gateway 推送。  
6. 修正 `APPLY_ALLIANCE_REQ` 错误码映射，并补充 `applyType` 合法性校验。  

## 6. 验证记录

- 代码核对方式：静态审阅核心链路（Game -> Social -> Gateway）与关键状态写入点（MySQL/Redis）。  
- 执行命令：`go test ./...`（`2026-05-13`）  
- 结果：**失败**（存在仓库级构建问题，非 alliance 单模块独有），示例：
  - `easyDB/backendDB.go`: `fmt.Errorf` 格式化参数问题
  - `gameController/shopController.go`: logger 格式化参数问题
  - `main/*`: 多个 `main` 重定义（多入口一起构建）

## 7. 变更记录

### 2026-05-13

1. 依据当前代码重写文档结构与链路说明。  
2. 同步确认 `NotifyAllianceOperationToGateway` 操作类型仍为 `ENTER/KICKOUT/NEW_APPLY`。  
3. 保留并更新心跳参数（`30m` 检查、`7d` 解散、`3d/7d` 传位）。  
4. 风险清单新增“改盟名道具异常返还”和“RPC 发送失败无道具补偿”两项 P1。  
5. 更新验证记录，补充本次 `go test ./...` 实际失败结论。  
