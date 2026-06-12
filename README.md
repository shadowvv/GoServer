# GoServer

Go 游戏服务器框架，采用单仓多进程部署方式，将客户端接入、主游戏逻辑、社交、排行榜、HTTP 接口、运营后台和机器人压测拆分为不同节点。

## 架构概览

- 外部接入：客户端通过 WebSocket 连接 `gateway`，HTTP 请求进入 `http` 或 `backend` 节点。
- 节点通信：内部节点通过 gRPC 双向流通信，通过 etcd 完成服务注册、发现和上下线监听。
- 数据存储：MySQL 承载账号、玩家、排行榜、日志、后台等业务数据；Redis 承载在线状态、缓存、通知和运行指标。
- 逻辑模型：玩家玩法进入场景后按场景和玩家任务队列串行执行，社交按联盟维度串行，降低并发写冲突。

核心节点：

| 节点 | 主要职责 |
|---|---|
| `gateway` | 客户端 WebSocket 接入、登录链路承接、消息转发、踢人、广播、在线会话管理。 |
| `game` | 主游戏逻辑节点，承载玩家模型、场景调度、任务、背包、邮件、战斗、活动等核心玩法。 |
| `social` | 社交/联盟节点，独立处理联盟数据、联盟消息、联盟维度并发控制。 |
| `rankBoard` | 排行榜节点，维护榜单、排行数据刷新、榜单持久化和跨服查询。 |
| `http` | 登录、服务器列表、公告、充值回调、GM 充值等 HTTP 接口。 |
| `backend` | GM/运营后台接口，处理管理端请求、数据导入导出、运营查询和控制。 |
| `robot` | 压测和自动化客户端，模拟登录、建连、发包、收包和统计。 |

## 目录结构

```text
GoServer/
├── AIDoc/              # 架构说明、AI 辅助文档和补充设计文档
├── config/             # 节点、平台、后台、机器人等 YAML 配置
├── gameConfig/         # 游戏策划配置 JSON 数据
├── logs/               # 默认日志输出目录
├── rpcProto/           # 内部 gRPC proto 定义和生成入口
├── script/             # 脚本目录
├── server/             # 服务端 Go 代码主体
├── serverKeys/         # 服务密钥或平台凭据目录
├── sql/                # 数据库建表、初始化和变更 SQL
├── go.mod              # Go module 定义
└── README.md           # 项目说明
```

## `server` 包说明

| 包/目录 | 功能 |
|---|---|
| `server/enum` | 全局枚举和常量，包括节点类型、消息分类、Redis Key、SQL 模板、功能解锁、活动、道具、排行榜、后台权限等。 |
| `server/main` | 各节点启动入口，每个文件对应一个可独立启动的进程。 |
| `server/logic` | 业务核心层，包含控制器、领域模型、玩法服务、平台编排、RPC 控制器和配置加载。 |
| `server/service` | 基础设施层，封装 MySQL、Redis、etcd、gRPC、WebSocket、HTTP、日志、支付、敏感词过滤等通用服务。 |
| `server/robot` | 机器人压测框架，负责批量登录、WebSocket 建连、协议发包、回包匹配和性能统计。 |
| `server/tool` | 通用工具和配置生成工具，包括 YAML/文件加载、时间、随机、JWT、ID 生成、切片处理、Excel 转配置等。 |

## 启动入口

`server/main` 下的入口文件用于启动不同节点：

| 文件 | 启动节点/工具 | 说明 |
|---|---|---|
| `gameMain.go` | `game` | 启动主游戏逻辑节点，初始化玩家模型、玩法服务、场景服务、事件总线、敏感词、排行榜事件处理等。 |
| `gatewayMain.go` | `gateway` | 启动网关节点，初始化 WebSocket 网络服务、会话管理、客户端协议编解码、网关消息转发。 |
| `httpMain.go` | `http` | 启动 HTTP 节点，注册登录、服务器信息、公告、充值等 Web 接口，并通过 RPC 与内部节点通信。 |
| `backendMain.go` | `backend` | 启动运营后台节点，注册 `/manage/*` 管理接口，连接后台库、日志库、游戏库、排行库等。 |
| `socialMain.go` | `social` | 启动社交节点，初始化联盟服务、社交消息处理和跨节点 RPC。 |
| `rankBoardMain.go` | `rankBoard` | 启动排行榜节点，初始化榜单服务、排行消息处理和持久化依赖。 |
| `robotMain.go` | `robot` | 启动机器人压测客户端。 |
| `robotApiMain.go` | `robotApi` | 启动机器人控制 API 服务。 |
| `allInOneMain.go` | 预留/聚合入口 | 用于本地或开发阶段聚合启动的预留入口。 |
| `xlsx2json.go` | 配置生成工具 | 将 Excel 配置导出为 JSON，并生成 Go loader 文件，然后执行配置数据检测。 |

节点启动参数由 `server/logic/platform.ParseCmdArgs()` 支持：

```bash
go run ./server/main/gameMain.go -nodeId=1 -nodeType=game -env=local -configName=local -channelId=1
```

如果不传 `-nodeId`，会读取 `config/nodeConfig.yaml`。普通节点读取 `config/platformConfig.yaml`，后台节点读取 `config/backendConfig.yaml`。

## `server/logic` 业务包说明

### 控制器和协议入口

| 包/目录 | 功能 |
|---|---|
| `logic/gameController` | 客户端、HTTP、GM、网关、社交、排行榜等消息入口。负责协议注册、handler 组织、参数校验、功能解锁校验和错误回包。 |
| `logic/pb` | 客户端业务协议生成代码，对应登录、聊天、背包、英雄、邮件、任务、商城、活动、排行榜等消息。 |
| `logic/rpcPb` | 内部 gRPC 协议生成代码，对应 gateway/game/social/rank/http 节点间 RPC。 |
| `logic/webProto` | HTTP/Web 层请求响应结构定义。 |
| `logic/rpcController` | 跨节点 RPC 控制器，封装 gRPC 客户端、服务端注册、消息发送、回包处理、自动重连和节点代理。 |

### 模型和通用接口

| 包/目录 | 功能 |
|---|---|
| `logic/model` | 领域模型层，包含玩家聚合根 `PlayerModel` 以及账号、角色、背包、英雄、宠物、邮件、任务、活动、充值、公告、白名单、封禁、排行榜快照等数据模型。 |
| `logic/logicCommon` | 业务公共接口和通用结构，定义道具、邮件、排行榜、GM、平台、用户、调度器等跨包接口；也包含吞吐监控等公共逻辑。 |
| `logic/gameConfig` | 游戏配置加载层，负责 JSON 配置结构、loader 注册、全量加载、热重载和配置检查。 |

### 核心玩法服务

| 包/目录 | 功能 |
|---|---|
| `logic/activityService` | 活动服务，维护服务器活动、开放活动状态、活动信息刷新，区分 game/gateway 侧使用场景。 |
| `logic/adChest` | 广告宝箱服务，处理广告奖励、广告宝箱状态和相关配置。 |
| `logic/adventure` | 冒险玩法服务，处理冒险进度、关卡、奖励等逻辑。 |
| `logic/cityAge` | 城市时代服务，处理城市时代升级、阶段解锁和相关成长。 |
| `logic/equipment` | 装备服务，处理装备养成、穿戴、GM 修改等逻辑。 |
| `logic/furniture` | 家具服务，处理家具持有、布置、升级或属性相关逻辑。 |
| `logic/gameServerInfoService` | 服务器信息服务，读取和缓存服务器列表、开服信息、活动开放等服务端状态。 |
| `logic/gloryArenaService` | 荣耀竞技场服务，包含 game/gateway/rankBoard 侧状态同步、匹配池和排行相关逻辑。 |
| `logic/gm` | GM 内部服务和 GM 命令处理，提供运营、测试、调试入口。 |
| `logic/hero` | 英雄通用逻辑和 GM 处理，服务英雄养成、属性计算相关流程。 |
| `logic/idle` | 挂机收益服务，处理离线/在线收益、资源累积和领取。 |
| `logic/inventory` | 背包服务，管理道具增删、消耗、检查、日志、GM 操作和背包模型。 |
| `logic/itemService` | 道具服务，提供统一发奖、消耗、道具日志和用户物品记录。 |
| `logic/lumber` | 伐木/资源生产类玩法服务。 |
| `logic/mail` | 邮件服务，负责邮件构建、发送、读取、领取、刷新通知和 GM 邮件。 |
| `logic/operationLogService` | 运营日志服务，异步写入行为、资源、道具、充值等运营数据。 |
| `logic/pass` | 通行证服务，处理系统通行证、过期刷新、奖励领取和活动联动。 |
| `logic/pet` | 宠物服务，处理宠物养成、属性和 GM 处理。 |
| `logic/petRecruit` | 宠物招募服务，处理招募池、抽取和奖励。 |
| `logic/raid` | 战斗/副本服务，封装主线、塔、副本、冒险、竞技场、荣耀竞技场等战斗执行器和结算逻辑。 |
| `logic/rankboardService` | 排行榜服务，维护榜单数据、排行事件处理、榜单刷新、点赞或排行信息查询。 |
| `logic/socialService` | 社交/联盟服务，管理联盟模型、成员、联盟操作和社交节点内存状态。 |
| `logic/task` | 任务服务，处理任务类型、进度推进、完成、领奖和 GM 操作。 |
| `logic/trial` | 试炼玩法服务，处理试炼关卡、挑战和奖励。 |
| `logic/turnTable` | 转盘玩法服务，处理抽奖、奖励池和活动关联。 |
| `logic/unlockService` | 功能解锁服务，基于配置、玩家进度和每日数据缓存判断系统开放条件。 |
| `logic/vipCard` | VIP/月卡类服务，处理权益、奖励、状态查询和跨服务接口。 |
| `logic/backend` | 后台业务逻辑，处理运营接口、导入导出、协议结构、ID 重映射和管理端请求分发。 |

## `logic/platform` 平台包说明

平台层负责把基础设施、节点配置、路由、调度器、会话、RPC、场景和业务服务装配成可运行节点。

| 包/目录 | 功能 |
|---|---|
| `logic/platform` | 通用启动骨架，解析命令行参数，读取节点配置，初始化日志，处理信号和 pprof 开关。 |
| `logic/platform/nodeConfig` | 节点配置结构、平台配置结构和当前节点全局信息。 |
| `logic/platform/gamePlatform` | game 节点装配入口，初始化 DB/Redis、配置、模型服务、场景、事件总线、RPC、敏感词、支付订单等。 |
| `logic/platform/gatewayPlatform` | gateway 节点装配入口，初始化网络服务、网关路由、会话管理、RPC、服务器信息和活动状态。 |
| `logic/platform/httpPlatform` | http 节点装配入口，初始化 Web 服务、DB/Redis、充值服务、服务器信息服务和内部 RPC。 |
| `logic/platform/backendPlatform` | backend 节点装配入口，初始化后台 Web 服务、多数据库连接、活动/解锁服务和后台路由。 |
| `logic/platform/socialPlatform` | social 节点装配入口，初始化社交服务、联盟维度调度和跨节点 RPC。 |
| `logic/platform/rankBoardPlatform` | rankBoard 节点装配入口，初始化排行库、排行榜服务、排行消息调度和跨节点 RPC。 |
| `logic/platform/dispatcherService` | 消息调度层，按节点类型加载 gateway/login/scene/social/rankBoard/sideway 等处理器，并按 session、scene、alliance 等维度分发。 |
| `logic/platform/logicRouter` | 消息路由层，维护 `msgID -> msgType`、反序列化类型和 handler 分发。 |
| `logic/platform/logicCodec` | 客户端和内部游戏协议编解码。 |
| `logic/platform/logicSessionManager` | 网关、游戏、联盟、排行榜会话模型和会话管理。 |
| `logic/platform/logicScene` | 场景执行模型，管理 `sceneId -> Scene`、`playerId -> sceneId`，按 tick 处理玩家外部消息、内部任务和回调。 |
| `logic/platform/dbPool` | 异步 DB 写入池，按玩家或业务 key 分 worker，保障同玩家写入顺序，并支持 barrier 等等待机制。 |
| `logic/platform/easyDB` | 数据库访问封装，区分 serverDB、gameDB、rankDB、logDB、backendDB 等角色。 |
| `logic/platform/eventService` | 事件总线和事件定义，用于跨玩法事件发布订阅。 |
| `logic/platform/loginMutexService` | 登录互斥控制，避免同账号/同玩家并发登录导致状态冲突。 |
| `logic/platform/payOrderService` | 支付订单服务，处理订单状态、派发和节点内支付流程协作。 |
| `logic/platform/platformLogger` | 平台层日志辅助封装。 |
| `logic/platform/ServerNodeService` | 节点服务治理，负责 etcd 注册、watch、节点缓存、上下线回调和 RPC 客户端初始化触发。 |

## `server/service` 基础服务包说明

| 包/目录 | 功能 |
|---|---|
| `service/dbService` | MySQL 和 Redis 初始化封装，提供连接池配置、GORM 日志适配和 Redis 客户端。 |
| `service/easyRpc` | gRPC 服务端和客户端基础封装，支持内部双向流通信。 |
| `service/etcd` | etcd 客户端、服务注册、续租、watcher 等封装。 |
| `service/logger` | 基于 zap 和 lumberjack 的日志服务，支持控制台输出、文件轮转、错误日志分离等。 |
| `service/netService` | WebSocket 网络服务，管理连接、session、收发包和接入层回调。 |
| `service/payService` | 支付服务抽象和 Google 支付校验实现。 |
| `service/serviceInterface` | 基础服务接口定义，如网络 acceptor、codec、router 等接口。 |
| `service/webService` | HTTP 服务封装，支持路由注册、超时、请求体大小限制等。 |
| `service/wordFilter` | 敏感词过滤服务，基于 trie matcher 构建词库，支持初始化、热重载、检测、查找和替换。 |

## `server/robot` 包说明

| 包/目录 | 功能 |
|---|---|
| `robot` | 机器人启动入口，装配配置、日志、平台、模块和运行流程。 |
| `robot/robotApi` | 机器人控制 API 服务，提供路由、请求结构和机器人 session 管理。 |
| `robot/robotCommon` | 机器人通用接口定义。 |
| `robot/robotConfig` | 机器人配置加载，读取 `config/robot.yaml`。 |
| `robot/robotLogger` | 机器人日志封装。 |
| `robot/robotLogic` | 机器人核心行为，处理登录、连接、发包、业务循环和状态维护。 |
| `robot/robotModuleController` | 机器人模块化控制器，如系统、邮件、装备等自动化行为。 |
| `robot/robotMonitor` | 机器人平台指标和系统资源统计。 |
| `robot/robotPlatform` | 机器人平台装配，初始化路由、配置、监控和运行环境。 |
| `robot/robotRouter` | 机器人协议路由和编解码，处理动态 PB 请求/响应匹配。 |
| `robot/robotUtils` | 机器人工具函数，包含通用工具和 proto 反射工具。 |

## `server/tool` 工具包说明

| 包/目录 | 功能 |
|---|---|
| `tool` | 通用工具集合，包含配置加载、文件读取、时间转换、随机、JWT、ID 生成、类型转换、切片和 JSON slice 辅助处理。 |
| `tool/xlsx2json` | Excel 配置解析和 JSON 导出工具。 |
| `tool/xlsx2goloader` | 根据 Excel/JSON 配置生成 Go loader 代码。 |

## 配置与协议

| 目录/文件 | 说明 |
|---|---|
| `config/nodeConfig.yaml` | 本地默认节点信息：`nodeId`、`nodeType`、`environment`、`configName`、`channelId`。 |
| `config/platformConfig.yaml` | gateway/game/social/rankBoard/http 等普通节点的平台配置。 |
| `config/backendConfig.yaml` | backend 节点独立平台配置。 |
| `config/robot.yaml` | 机器人压测配置。 |
| `config/robotApiRoutes.yaml` | 机器人 API 路由配置。 |
| `config/dirtyWord.txt` | 敏感词词库，game 节点启动时由 `wordFilter` 加载。 |
| `gameConfig/` | 游戏配置 JSON，启动时由 `logic/gameConfig.LoadAllConfig()` 加载。 |
| `rpcProto/*.proto` | 内部 gRPC proto 源文件。 |
| `server/logic/rpcPb` | 由 `rpcProto` 生成的内部 RPC Go 代码。 |
| `server/logic/pb` | 客户端业务协议生成代码。 |
| `sql/` | 数据库结构和初始化 SQL。 |

生成内部 RPC 代码：

```bash
cd rpcProto
go generate
```

## 启动流程

普通节点统一遵循以下链路：

1. 读取命令行参数；未传参数时读取 `config/nodeConfig.yaml`。
2. 根据 `nodeType + configName` 从 `platformConfig.yaml` 或 `backendConfig.yaml` 读取节点配置。
3. 初始化日志、MySQL、Redis 等基础依赖。
4. 初始化 router、dispatcher、session manager、业务服务。
5. 启动 gRPC 服务并注册到 etcd，监听目标节点上下线。
6. 加载 `gameConfig`，注册协议 handler。
7. 进入节点主循环，等待网络、RPC、HTTP 或机器人任务。

## 信号处理

| 信号 | 行为 |
|---|---|
| `SIGHUP` | 热重载 `gameConfig`，并触发节点自己的 `AfterAllConfigReload()`。game 节点会同时重载敏感词。 |
| `SIGQUIT` | 触发 `KickAllPlayer()`，用于踢出在线玩家。 |
| `SIGTERM` / `SIGINT` | 当前实现直接退出进程。后续可扩展为优雅停服、排空队列和关闭连接。 |

## 运行环境

- Go 1.24
- etcd
- MySQL 8.0
- Redis 6.2

## 开发注意事项

- 新增客户端协议时，优先在 `gameController.Register*Message` 系列函数中完成消息注册，保持 router/dispatcher 链路一致。
- 新增玩法数据时，通常需要同时处理 `model`、`gameConfig`、`gameController`、对应玩法 service、DB/Redis 持久化和必要的 GM/后台入口。
- 涉及玩家状态写入时，应优先走场景/玩家任务队列或 DBPool，避免跨 goroutine 直接改玩家模型。
- 新增跨节点能力时，需要同时关注 `rpcProto`、`rpcPb`、`rpcController`、`ServerNodeService` 和对应 platform 的初始化链路。
- `config/dirtyWord.txt` 当前是相对路径，依赖进程工作目录；部署时需保证工作目录为项目根目录或调整为绝对路径配置。
