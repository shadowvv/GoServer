# GoServer

Go + Lua 游戏服务器框架

---

## 📂 项目结构

    GoServer/ # 项目根目录
    ├── config/ # 配置目录
    ├── logs/ # 日志目录
    |── proto/ # Protobuf 协议目录
    |── server/ # 服务器代码根目录
    ├── logic/ # 游戏逻辑相关代码
    ├── main/ # 服务器启动代码
    ├── module/ # 可独立抽离的功能模块
    └── service/ # 核心服务模块
        ├── db/ # 数据库服务
        ├── etcd/ # Etcd 服务
        ├── fileLoader/ # 文件加载服务
        ├── logger/ # 日志服务
        ├── netService/ # 网络服务
        ├── rpc/ # RPC 服务
        └── serviceInterface/ # 服务接口定义