# Robot 实现同步（2026-04-29）

本文档基于当前代码状态同步：`E:\dhsOtherServer\dhs_server\server\robot`。

## 1. 目录结构

- `server/robot/booter.go`：机器人入口 `robot.Boot()`
- `server/robot/robotPlatform`：平台生命周期、批量启动/停止、信号等待
- `server/robot/robotLogic`：单机器人核心逻辑（登录、连接、读写循环、动作循环）
- `server/robot/robotRouter`：消息注册中心 + 编解码
- `server/robot/robotConfig`：配置加载与 operation 构建
- `server/robot/robotModuleController`：模块 controller（system/equipment/mail）
- `server/robot/robotMonitor`：系统监控与吞吐/延迟观测
- `server/robot/robotCommon`：公共接口与结构

## 2. 启动链路

主入口：`server/main/robotMain.go`

顺序：
1. `robot.Boot()`
2. `robotPlatform.NewRobotPlatform()`
   - `robotRouter.RegisterAllRobotMessages()`
   - `robotConfig.LoadConfig("config/robot.yaml")`
   - `logger.InitLoggerByConfig(&cfg.Logger)`
   - `cfg.BuildRealOperations()`
3. `platform.StartAllRobots()`
4. `platform.WaitForExitSignal()`
5. `platform.StopAllRobots()`
6. `platform.PrintFinalSummary(startTime)`

## 3. Router / Controller 注册关系

- `robotRouter` 是注册中心，负责：
  - `RegisterRobotController`
  - `RegisterRobotMessageHandler`
  - `RegisterRobotReceiveMessageHandler`
  - `GetMessageCallback`
  - `BuildProtoMessageByMessageID`
  - `BuildReceiveProtoMessageByMessageID`
  - `ValidateRobotOperationBinding`
  - `EncodeMessage / DecodeMessage`
- `robotModuleController` 通过 `init()` 注册模块 controller。
- `robotPlatform` 使用空白导入 `_ "github.com/drop/GoServer/server/robot/robotModuleController"` 确保注册链路不丢失。

## 4. 登录与就绪流程

`robotLogic.Robot.Start()`：
1. HTTP 登录（`httpLogin`）
2. 建立 WS 连接
3. 启动 `readLoop/writeLoop/actionLoop`
4. 发送 `pb.LoginReq`

`system` 回调推进：
- `LoginResp`：日志 `login success...`，发送 `LoadSceneOverReq`
- `LoadSceneOverResp`：设置 `SetSceneLoaded(true)`、`SetAuthed(true)`，日志 `robot is ready...`

## 5. 机器人状态机与串行执行

当前状态：
- `StateConnected`
- `StateLoggedIn`
- `StateSceneLoaded`
- `StateReady`
- `StateWaitingResp`
- `StateStopped`

执行约束：
- 同一机器人严格串行。
- `Send()` 后进入 `StateWaitingResp`，并设置 `waitingMessageID = reqMsgID + 1`。
- 收到对应响应（或 `MESSAGE_ID_MESSAGE_ERROR` 携带的对应 `respMsgID`）后，解锁 waiting，恢复 `StateReady`。

## 6. 错误消息处理（MessageError）

`system` 已注册 `MESSAGE_ID_MESSAGE_ERROR`。

处理逻辑：
1. 打印错误日志：`respMsgID + errorCode`
2. 若 `respMsgID` 命中当前机器人 `waitingMessageID`，视作本次请求结束，解除等待，继续后续逻辑，避免卡死在 `StateWaitingResp`。

## 7. 心跳

`sendHeartbeat()` 已改为发送业务协议：
- `pb.HeartReq`
- `MESSAGE_ID_HEART_REQ`

不再发送 WebSocket Ping。

## 8. 运行日志（新增）

当前会打印：
- 登录成功日志
- 场景加载完成、进入可执行状态日志
- 机器人正式开始执行时的运行计划日志（mode/interval/duration/模块与消息列表）
- 每个操作的开始日志（模块 + messageID + messageName）
- 每个操作的结束日志（sent 或 failed，含耗时）

## 9. 监控现状（SystemStats）

`PerformanceStats` 已移除，监控收敛到 `SystemStats`。

`SystemStats` 周期自动上报（内部 reporter）包含：
- 机器人总数（运行态加减）
- 吞吐量：
  - period：本周期 sent msgs / completed ops / msg/s / ops/s
  - total：累计 sent msgs / completed ops
- 全局延迟统计：
  - avg / p50 / p95 / p99
  - min（附 msgID + robotID）
  - max（附 msgID + robotID）
- 系统资源：
  - CPU、goroutine、进程内存、系统内存、GC 次数及峰值

## 10. 代码评价

整体评价：这版重构方向是对的，边界已经比较清晰，运行链路可读性和可观测性都有明显提升。

优点：
- 分层明确（platform / logic / router / controller / monitor）
- 串行单飞模型稳定，便于控制和定位问题
- `MessageError` 已纳入状态机收敛，避免 waiting 卡死
- 监控数据从“静态配置推测”改为“运行态真实计数”

## 11. 可优化点

1. 并发安全细化  
`SystemStats` 中 reporter 启停字段已加锁，但所有控制字段建议统一在同一把锁下读写（目前是可用的，仍可进一步收敛）。

2. waiting 状态超时机制  
当前主要依赖响应或 `MessageError` 解锁。建议增加 `waiting` 超时兜底（按请求级超时自动恢复或重试），提升抗异常能力。

3. 延迟统计按消息拆分  
目前延迟统计是全局聚合。后续可选增加“按 messageID 聚合分桶”，便于定位具体协议慢点。

4. 日志结构化  
当前操作日志是文本拼接。建议未来切到结构化字段（robot/module/msgID/cost/status），更方便平台检索和告警。

## 12. 风险点

1. 协议约定风险  
当前“请求完成”依赖 `resp = req + 1` 约定。若服务端某些协议不满足该规律，需要额外映射策略，否则可能误判 waiting。

2. 高并发日志量  
操作开始/结束日志在大量机器人并发下会产生较大 IO 压力，需要按环境分级或采样。

3. 自动注册隐式依赖  
controller 注册依赖空白导入触发 `init()`，未来如果入口调整或包路径变动，容易出现“注册缺失”问题，建议保留启动自检。
