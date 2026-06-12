# webService 遗留问题（2026-05-27）

## 遗留问题

- CORS 配置当前仍为“回写任意 Origin + Allow-Credentials=true”。
- 该项暂不在应用层修改，依赖 Nginx 侧防护。

## 保留前提（必须满足）

- Nginx 必须做 Origin 白名单拦截（非白名单直接拒绝）。
- 后端服务端口不得对外网直连暴露。

## netService 遗留问题（2026-05-27）

- [ ] 限制 WebSocket `CheckOrigin`（当前全放行）。
  - 文件：`server/service/netService/netServer.go`
  - 建议：按环境配置白名单域名，区分内网/公网。

- [ ] `OnSessionClose` 后的日志语义统一。
  - 文件：
    - `server/logic/platform/logicSessionManager/gameSessionManager.go`
    - `server/logic/platform/logicSessionManager/gatewaySessionManager.go`
  - 现状：日志仍为 `session timeout`。
  - 建议：改为 `session closed` 并记录关闭原因（timeout/read_error/write_error/manual_kick）。

## 仓库现存构建问题（非本次 netService 改动引入）

- [ ] `server/main/*.go`
  - `go test ./server/...` 下多 `main` 函数冲突。
