# GoServer

Go 游戏服务器框架

---

## 📂 项目结构

    GoServer/           # 项目根目录
    ├── config/         # 节点配置目录
    ├── docs/           # 文档目录
    ├── logs/           # 日志目录
    ├── examples/       # 功能示例目录
    ├── gameConfig/     # 游戏配置目录
    ├── robot/          # 机器人模块目录
    ├── rpcProto/       # RPC协议目录
    ├── sql/            # 数据库sql目录
    |── server/         # 服务器代码根目录
        |── enum/           # 服务器所有的枚举和常量
        ├── logic/          # 游戏逻辑相关代码
            |── gameConfig/         # 读取游戏配置对应的结构
            |── gameController/     # 玩家操作入口
            |—— gm/                 # GM相关代码
            |── platform/           # 平台相关代码
                    |── dbPool/               # 数据库连接池
        ├── main/           # 服务器启动代码
        └── service/        # 核心服务模块
            ├── db/                 # 数据库服务
            ├── easyRpc/            # RPC服务
            ├── etcd/               # Etcd 服务
            ├── logger/             # 日志服务
            ├── netService/         # 网络服务
            ├── payService/         # 支付服务
            └── serviceInterface/   # 服务接口定义
            |── webService/         # Web服务
            └── wordFilter/         # 敏感词过滤服务

## 运行环境
    go version 1.24
    etcd
    mysql 8.0
    redis v6.2