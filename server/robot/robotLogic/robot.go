package robotLogic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/webProto"
	"github.com/drop/GoServer/server/robot/robotCommon"
	"github.com/drop/GoServer/server/robot/robotConfig"
	"github.com/drop/GoServer/server/robot/robotLogger"
	"github.com/drop/GoServer/server/robot/robotMonitor"
	"github.com/drop/GoServer/server/robot/robotRouter"
	"github.com/drop/GoServer/server/robot/robotUtils"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type Robot struct {
	// 登录信息
	Name       string
	Account    string
	ServerID   int32
	LoginURL   string
	Channel    int32
	Version    string
	Language   uint16
	DeviceID   string
	AppID      string
	Sign       string
	LoginToken string

	// 配置
	cfg            *robotConfig.RobotConfig
	mode           string // supported models: random, custom
	interval       int64
	duration       int
	autoOperations atomic.Bool

	// 当前运行状态
	startTime          time.Time
	currentModule      string
	currentMessageIdx  int
	waitingMessageID   int32
	waitingReqMsgID    atomic.Uint32
	waitingStartedAt   atomic.Int64
	loginReqSentAt     atomic.Int64
	loadSceneReqSentAt atomic.Int64
	state              atomic.Int32

	// 网络连接消息队列
	conn           *websocket.Conn
	messageChannel chan *robotCommon.MessageStruct
	writeChannel   chan *outboundMessage

	// 机器人启动关闭控制
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once

	// 监控相关
	monitor *robotMonitor.PlatformMonitor
}

const (
	StateConnected = iota
	StateLoggedIn
	StateSceneLoaded
	StateReady
	StateWaitingResp
	StateStopped
)

func NewRobot(name, account string, serverID int32, loginURL string, channel int32, version string, language uint16, deviceID, appID, sign string, cfg *robotConfig.RobotConfig, mode string, interval int64, duration int, monitor *robotMonitor.PlatformMonitor) *Robot {
	ctx, cancel := context.WithCancel(context.Background())

	if channel <= 0 {
		channel = 1
	}
	if version == "" {
		version = "1.0.0"
	}
	if deviceID == "" {
		deviceID = account
	}

	r := &Robot{
		Name:           name,
		Account:        account,
		ServerID:       serverID,
		LoginURL:       loginURL,
		Channel:        channel,
		Version:        version,
		Language:       language,
		DeviceID:       deviceID,
		AppID:          appID,
		Sign:           sign,
		cfg:            cfg,
		mode:           mode,
		interval:       interval,
		duration:       duration,
		ctx:            ctx,
		cancel:         cancel,
		messageChannel: make(chan *robotCommon.MessageStruct, 100),
		writeChannel:   make(chan *outboundMessage, 100),
		startTime:      time.Now(),
		monitor:        monitor,
	}
	r.autoOperations.Store(true)
	return r
}

func (r *Robot) Start() error {
	if r.LoginURL == "" {
		return fmt.Errorf("loginURL is required")
	}

	httpLoginStartedAt := time.Now()
	robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=http_login status=start loginURL=%s", r.LoginURL))
	wsURL, sessionToken, serverID, err := r.httpLogin()
	httpLoginCost := time.Since(httpLoginStartedAt)
	if err != nil {
		robotLogger.ErrorWithRobot(r, fmt.Sprintf("phase=http_login status=failed costMs=%d err=%v", httpLoginCost.Milliseconds(), err))
		return fmt.Errorf("http login failed: %w", err)
	}
	robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=http_login status=success costMs=%d ws=%s returnedServerID=%d", httpLoginCost.Milliseconds(), wsURL, serverID))

	r.LoginToken = sessionToken
	if serverID > 0 {
		r.ServerID = serverID
	}

	wsConnectStartedAt := time.Now()
	robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=ws_connect status=start ws=%s", wsURL))
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	wsConnectCost := time.Since(wsConnectStartedAt)
	if err != nil {
		robotLogger.ErrorWithRobot(r, fmt.Sprintf("phase=ws_connect status=failed costMs=%d err=%v", wsConnectCost.Milliseconds(), err))
		return fmt.Errorf("connect failed: %w", err)
	}
	robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=ws_connect status=success costMs=%d", wsConnectCost.Milliseconds()))
	r.conn = conn
	r.state.Store(StateConnected)

	pongWait := 30 * time.Second
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(appData string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	go r.readLoop()
	go r.writeLoop()
	go r.actionLoop()

	r.loginReqSentAt.Store(time.Now().UnixNano())
	if err := r.sendLoginRequest(); err != nil {
		r.loginReqSentAt.Store(0)
		robotLogger.ErrorWithRobot(r, fmt.Sprintf("phase=send_login_req status=failed err=%v", err))
		r.Stop()
		return fmt.Errorf("send login request failed: %w", err)
	}
	robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=send_login_req status=success req=%d wait=%d", pb.MESSAGE_ID_LOGIN_REQ, pb.MESSAGE_ID_LOGIN_RESP))

	if r.monitor != nil {
		r.monitor.SystemStats.AddRobot()
	}
	robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=robot_connected status=success ws=%s", wsURL))
	return nil
}

func (r *Robot) sendLoginRequest() error {
	loginReq := &pb.LoginReq{
		Account:  r.Account,
		ServerId: r.ServerID,
		Token:    r.LoginToken,
	}
	err := r.Send(uint32(pb.MESSAGE_ID_LOGIN_REQ), loginReq)
	return err
}

func (r *Robot) httpLogin() (wsAddr string, sessionToken string, serverID int32, err error) {
	loginReq := &webProto.LoginReq{
		Account:  r.Account,
		Channel:  r.Channel,
		Version:  r.Version,
		ServerID: r.ServerID,
		Language: r.Language,
		DeviceID: r.DeviceID,
		AppID:    r.AppID,
		Sign:     r.Sign,
	}

	body, err := json.Marshal(loginReq)
	if err != nil {
		return "", "", 0, fmt.Errorf("编码HTTP登录请求失败: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.LoginURL, bytes.NewReader(body))
	if err != nil {
		return "", "", 0, fmt.Errorf("创建HTTP登录请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("调用HTTP登录失败: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, fmt.Errorf("读取HTTP登录响应失败: %v", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", "", 0, fmt.Errorf("HTTP登录状态码异常: %d, body=%s", resp.StatusCode, string(respBody))
	}

	var errResp webProto.WebErrorMessage
	if json.Unmarshal(respBody, &errResp) == nil && errResp.Code != 0 {
		return "", "", 0, fmt.Errorf("HTTP登录返回错误码: %d", errResp.Code)
	}

	var loginResp webProto.LoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return "", "", 0, fmt.Errorf("解析HTTP登录响应失败: %v, body=%s", err, string(respBody))
	}

	if loginResp.BanInfo != nil {
		return "", "", 0, fmt.Errorf("账号被封禁: reason=%d, endTime=%d", loginResp.BanInfo.Reason, loginResp.BanInfo.EndTime)
	}

	if loginResp.WsAddr == "" {
		if loginResp.Announce != nil {
			return "", "", 0, fmt.Errorf("HTTP登录被公告拦截")
		}
		return "", "", 0, fmt.Errorf("HTTP登录成功但未返回wsAddr")
	}

	return loginResp.WsAddr, loginResp.SessionToken, loginResp.ServerId, nil
}

// readLoop 读取消息循环
func (r *Robot) readLoop() {
	defer func() {
		if !r.isStopped() {
			robotLogger.InfoWithRobot(r, "readLoop退出，停止机器人")
			r.Stop()
		}
	}()

	consecutiveErrors := 0
	maxConsecutiveErrors := 3

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		if r.conn == nil {
			return
		}

		_ = r.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		msgType, data, err := r.conn.ReadMessage()
		if err != nil {
			consecutiveErrors++

			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				robotLogger.InfoWithRobot(r, fmt.Sprintf("连接正常关闭: %v", err))
				return
			}

			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				errStr := err.Error()
				if strings.Contains(errStr, "1006") || strings.Contains(errStr, "abnormal closure") {
					robotLogger.InfoWithRobot(r, fmt.Sprintf("连接异常关闭(1006): 可能服务器崩溃或网络中断: %v", err))
				} else {
					robotLogger.ErrorWithRobot(r, fmt.Sprintf("连接异常关闭: %v", err))
				}
				return
			}

			errStr := err.Error()
			if r.isStopped() || strings.Contains(errStr, "use of closed network connection") {
				robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=read_loop status=stopped reason=connection_closed err=%v", err))
				return
			}
			if strings.Contains(errStr, "unexpected EOF") {
				robotLogger.InfoWithRobot(r, fmt.Sprintf("连接意外断开(EOF): %v，可能是服务器异常关闭", err))
				return
			}
			if strings.Contains(errStr, "i/o timeout") {
				if consecutiveErrors >= maxConsecutiveErrors {
					robotLogger.ErrorWithRobot(r, fmt.Sprintf("连续%d次读取超时，关闭连接: %v", consecutiveErrors, err))
					return
				}
				robotLogger.InfoWithRobot(r, fmt.Sprintf("读取超时(%d/%d)，继续重试: %v", consecutiveErrors, maxConsecutiveErrors, err))
				time.Sleep(1 * time.Second)
				continue
			}

			if consecutiveErrors >= maxConsecutiveErrors {
				robotLogger.ErrorWithRobot(r, fmt.Sprintf("连续%d次读取错误，关闭连接: %v", consecutiveErrors, err))
				return
			}
			robotLogger.InfoWithRobot(r, fmt.Sprintf("读取错误(%d/%d): %v，继续重试", consecutiveErrors, maxConsecutiveErrors, err))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		consecutiveErrors = 0

		if msgType != websocket.BinaryMessage {
			continue
		}
		if len(data) < 4 {
			continue
		}

		msg, err := robotRouter.DecodeMessage(data)
		if err != nil {
			robotLogger.ErrorWithRobot(r, fmt.Sprintf("解码消息失败: %v", err))
			continue
		}
		if msg == nil {
			continue
		}

		r.messageChannel <- msg
	}
}

func (r *Robot) writeLoop() {
	defer func() {
		if !r.isStopped() {
			robotLogger.InfoWithRobot(r, "writeLoop exited, stopping robot")
			r.Stop()
		}
	}()

	for {
		select {
		case <-r.ctx.Done():
			return
		case out := <-r.writeChannel:
			if out == nil {
				continue
			}

			err := r.writeSocketMessage(out.msgType, out.payload, out.msgID)

			if out.done != nil {
				out.done <- err
				close(out.done)
			}

			if err != nil {
				return
			}
		}
	}
}

func (r *Robot) writeSocketMessage(msgType int, payload []byte, msgID uint32) error {
	if r.isStopped() {
		return fmt.Errorf("robot already stopped")
	}
	if r.conn == nil {
		return fmt.Errorf("connection already closed")
	}

	writeTimeout := 10 * time.Second
	if msgType == websocket.PingMessage {
		writeTimeout = 5 * time.Second
	}
	_ = r.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := r.conn.WriteMessage(msgType, payload); err != nil {
		if msgType == websocket.PingMessage {
			robotLogger.InfoWithRobot(r, fmt.Sprintf("send heartbeat failed: %v", err))
		} else if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
			robotLogger.ErrorWithRobot(r, fmt.Sprintf("write timeout(MsgID=%d): %v", msgID, err))
		} else {
			robotLogger.ErrorWithRobot(r, fmt.Sprintf("write failed(MsgID=%d): %v", msgID, err))
		}
		return err
	}
	return nil
}

// actionLoop drives heartbeat, incoming message dispatch, and configured operations.
//
// Supported models:
// - random: pick one message from configured modules each tick
// - custom: execute messages by module order and message order, loop from first again until duration ends
func (r *Robot) actionLoop() {
	heartbeatTicker := time.NewTicker(10 * time.Second)
	readyTicker := time.NewTicker(100 * time.Millisecond)
	defer heartbeatTicker.Stop()
	defer readyTicker.Stop()

	var opTicker *time.Ticker
	var opTick <-chan time.Time
	var durationTimer *time.Timer
	var durationTick <-chan time.Time
	ready := false

	for {
		select {
		case <-r.ctx.Done():
			if opTicker != nil {
				opTicker.Stop()
			}
			if durationTimer != nil {
				durationTimer.Stop()
			}
			return
		case msg := <-r.messageChannel:
			r.handleMessage(msg)
		case <-heartbeatTicker.C:
			r.sendHeartbeat()
		case <-readyTicker.C:
			if ready || r.state.Load() < StateReady {
				continue
			}
			if !r.AutoOperationsEnabled() {
				ready = true
				if r.duration > 0 {
					durationTimer = time.NewTimer(time.Duration(r.duration) * time.Second)
					durationTick = durationTimer.C
				}
				robotLogger.InfoWithRobot(r, "phase=manual_ready status=success")
				continue
			}
			if !r.hasExecutableMessages() {
				ready = true
				if r.duration > 0 {
					durationTimer = time.NewTimer(time.Duration(r.duration) * time.Second)
					durationTick = durationTimer.C
				}
				robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=operation_loop status=idle reason=no_executable_messages duration=%ds", r.duration))
				continue
			}
			ready = true
			if r.duration > 0 {
				durationTimer = time.NewTimer(time.Duration(r.duration) * time.Second)
				durationTick = durationTimer.C
			}
			moduleGroups, _ := r.runtimeModuleMessages()
			robotLogger.InfoWithRobot(
				r,
				fmt.Sprintf(
					"phase=operation_loop status=start mode=%s intervalMs=%d duration=%ds plan=%s",
					r.mode, r.interval, r.duration, r.describeRuntimePlan(moduleGroups),
				),
			)
			opTicker = time.NewTicker(time.Duration(r.interval) * time.Millisecond)
			opTick = opTicker.C
		case <-durationTick:
			robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=robot_stop status=start reason=duration_reached duration=%ds uptimeMs=%d", r.duration, time.Since(r.startTime).Milliseconds()))
			if opTicker != nil {
				opTicker.Stop()
			}
			r.Stop()
			return
		case <-opTick:
			if r.isStopped() {
				if opTicker != nil {
					opTicker.Stop()
				}
				return
			}
			if atomic.LoadInt32(&r.waitingMessageID) != 0 {
				continue
			}
			if r.duration > 0 && time.Since(r.startTime) >= time.Duration(r.duration)*time.Second {
				robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=robot_stop status=start reason=duration_reached duration=%ds uptimeMs=%d", r.duration, time.Since(r.startTime).Milliseconds()))
				if opTicker != nil {
					opTicker.Stop()
				}
				r.Stop()
				return
			}

			var (
				messageID uint32
				ok        bool
			)
			switch r.mode {
			case "random":
				messageID, ok = r.pickRandomMessageID()
				if !ok {
					robotLogger.ErrorWithRobot(r, "random model has no executable messages")
					if opTicker != nil {
						opTicker.Stop()
					}
					return
				}
			case "custom":
				messageID, ok = r.nextCustomMessageID()
				if !ok {
					robotLogger.ErrorWithRobot(r, "custom model has no executable messages")
					if opTicker != nil {
						opTicker.Stop()
					}
					return
				}
			default:
				robotLogger.ErrorWithRobot(r, "unsupported model, only random/custom are allowed")
				if opTicker != nil {
					opTicker.Stop()
				}
				r.Stop()
				return
			}
			r.executeOperation(messageID)
		}
	}
}

// handleMessage 处理接收到的消息
func (r *Robot) handleMessage(msg *robotCommon.MessageStruct) {
	r.HandleWaitingResponseByRespID(msg.MsgID)

	if callback := robotRouter.GetMessageCallback(msg.MsgID); callback != nil {
		callback(r, msg)
	} else {
		if msg.MsgID < 500 {
			robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=message_dispatch status=ignored type=system msgID=%d", msg.MsgID))
		} else {
			robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=message_dispatch status=ignored type=normal msgID=%d", msg.MsgID))
		}
	}
}

func (r *Robot) HandleWaitingResponseByRespID(respMsgID uint32) bool {
	waitingID := atomic.LoadInt32(&r.waitingMessageID)
	if waitingID == 0 || int32(respMsgID) != waitingID {
		return false
	}

	startedAtNs := r.waitingStartedAt.Load()
	reqMsgID := r.waitingReqMsgID.Load()
	atomic.StoreInt32(&r.waitingMessageID, 0)
	r.waitingReqMsgID.Store(0)
	r.waitingStartedAt.Store(0)

	if !r.isStopped() {
		r.state.Store(StateReady)
	}

	if startedAtNs > 0 {
		latency := time.Since(time.Unix(0, startedAtNs))
		if r.monitor != nil {
			r.monitor.SystemStats.RecordOperationLatency(latency, reqMsgID, r.Name)
		}
		robotLogger.InfoWithRobot(
			r,
			fmt.Sprintf(
				"phase=recv_response status=success req=%d(%s) resp=%d(%s) costMs=%d",
				reqMsgID, pb.MESSAGE_ID(reqMsgID).String(), respMsgID, pb.MESSAGE_ID(respMsgID).String(), latency.Milliseconds(),
			),
		)
	}
	if r.monitor != nil {
		r.monitor.SystemStats.RecordOperationCompleted()
	}
	return true
}

func (r *Robot) hasExecutableMessages() bool {
	moduleGroups, ok := r.runtimeModuleMessages()
	if !ok {
		return false
	}
	for _, group := range moduleGroups {
		if len(group.MessageIDs) > 0 {
			return true
		}
	}
	return false
}

func (r *Robot) runtimeModuleMessages() ([]robotConfig.RobotRunModule, bool) {
	if r.cfg == nil {
		robotLogger.ErrorWithRobot(r, "robot config is nil")
		return nil, false
	}
	if len(r.cfg.CurrentRunModule) == 0 {
		return nil, false
	}
	return r.cfg.CurrentRunModule, true
}

func (r *Robot) pickRandomMessageID() (uint32, bool) {
	moduleGroups, ok := r.runtimeModuleMessages()
	if !ok {
		return 0, false
	}

	executable := make([]robotConfig.RobotRunModule, 0, len(moduleGroups))
	for _, group := range moduleGroups {
		if len(group.MessageIDs) > 0 {
			executable = append(executable, group)
		}
	}
	if len(executable) == 0 {
		return 0, false
	}

	selected := executable[rand.Intn(len(executable))]
	return selected.MessageIDs[rand.Intn(len(selected.MessageIDs))], true
}

func (r *Robot) nextCustomMessageID() (uint32, bool) {
	moduleGroups, ok := r.runtimeModuleMessages()
	if !ok {
		return 0, false
	}

	ensureCurrentModule := func() bool {
		if r.currentModule == "" {
			module, ok := firstExecutableModule(moduleGroups)
			if !ok {
				return false
			}
			r.currentModule = module
			r.currentMessageIdx = 0
			return true
		}

		for _, group := range moduleGroups {
			if group.Module == r.currentModule && len(group.MessageIDs) > 0 {
				return true
			}
		}

		module, ok := firstExecutableModule(moduleGroups)
		if !ok {
			return false
		}
		r.currentModule = module
		r.currentMessageIdx = 0
		return true
	}

	if !ensureCurrentModule() {
		return 0, false
	}

	var ids []uint32
	for _, group := range moduleGroups {
		if group.Module == r.currentModule {
			ids = group.MessageIDs
			break
		}
	}
	if len(ids) == 0 {
		return 0, false
	}

	if r.currentMessageIdx >= len(ids) {
		r.currentMessageIdx = 0
	}
	messageID := ids[r.currentMessageIdx]

	r.currentMessageIdx++
	if r.currentMessageIdx >= len(ids) {
		r.currentMessageIdx = 0
		if nextModule, ok := nextExecutableModule(moduleGroups, r.currentModule); ok {
			r.currentModule = nextModule
		}
	}
	return messageID, true
}

func firstExecutableModule(groups []robotConfig.RobotRunModule) (string, bool) {
	for _, group := range groups {
		if len(group.MessageIDs) > 0 {
			return group.Module, true
		}
	}
	return "", false
}

func nextExecutableModule(groups []robotConfig.RobotRunModule, current string) (string, bool) {
	if len(groups) == 0 {
		return "", false
	}

	start := -1
	for i, group := range groups {
		if group.Module == current {
			start = i
			break
		}
	}
	if start == -1 {
		return firstExecutableModule(groups)
	}

	for offset := 1; offset <= len(groups); offset++ {
		idx := (start + offset + len(groups)) % len(groups)
		group := groups[idx]
		if len(group.MessageIDs) > 0 {
			return group.Module, true
		}
	}
	return "", false
}

func (r *Robot) executeOperation(messageID uint32) {
	if r.state.Load() < StateReady {
		return
	}
	if r.cfg == nil {
		robotLogger.ErrorWithRobot(r, "robot config is nil")
		return
	}

	configItem, ok := r.cfg.FindRealOperation(pb.MESSAGE_ID(messageID))
	if !ok {
		robotLogger.ErrorWithRobot(r, fmt.Sprintf("operation not found for messageId: %d", messageID))
		return
	}
	moduleName := r.moduleByMessageID(configItem.MessageID)
	messageName := pb.MESSAGE_ID(configItem.MessageID).String()
	opStartedAt := time.Now()
	robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=operation status=start module=%s req=%d(%s) wait=%d(%s)", moduleName, configItem.MessageID, messageName, configItem.MessageID+1, pb.MESSAGE_ID(configItem.MessageID+1).String()))

	var msg proto.Message
	if configItem.Proto != nil {
		msg = proto.Clone(configItem.Proto)
	} else {
		// 动态手工构造：每次发送都实时取，避免在配置构建阶段固化。
		manualProto := robotRouter.GetManualProtoByMessageID(configItem.MessageID)
		if manualProto != nil {
			msg = proto.Clone(manualProto)
		} else {
			var err error
			msg, err = robotRouter.BuildProtoMessageByMessageID(configItem.MessageID)
			if err != nil {
				robotLogger.ErrorWithRobot(r, fmt.Sprintf("build default proto failed [messageId=%d]: %v", messageID, err))
				robotLogger.ErrorWithRobot(r, fmt.Sprintf("phase=operation status=failed step=build module=%s req=%d(%s) costMs=%d", moduleName, configItem.MessageID, messageName, time.Since(opStartedAt).Milliseconds()))
				return
			}
		}
	}

	if err := r.Send(uint32(configItem.MessageID), msg); err != nil {
		if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
			robotLogger.InfoWithRobot(r, fmt.Sprintf("send timeout [messageId=%d], server may be overloaded", messageID))
		} else {
			robotLogger.ErrorWithRobot(r, fmt.Sprintf("send failed [messageId=%d]: %v", messageID, err))
		}
		robotLogger.ErrorWithRobot(r, fmt.Sprintf("phase=operation status=failed step=send module=%s req=%d(%s) costMs=%d", moduleName, configItem.MessageID, messageName, time.Since(opStartedAt).Milliseconds()))
		return
	}
	robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=operation status=sent module=%s req=%d(%s) costMs=%d", moduleName, configItem.MessageID, messageName, time.Since(opStartedAt).Milliseconds()))
}

type OperationSendResult struct {
	RequestMsgID   uint32 `json:"requestMsgId"`
	ResponseMsgID  uint32 `json:"responseMsgId"`
	MessageName    string `json:"messageName"`
	ExpectedStatus string `json:"status"`
}

func (r *Robot) DisableAutoOperations() {
	r.autoOperations.Store(false)
}

func (r *Robot) AutoOperationsEnabled() bool {
	return r.autoOperations.Load()
}

func (r *Robot) IsReady() bool {
	return r.state.Load() == StateReady && atomic.LoadInt32(&r.waitingMessageID) == 0
}

func (r *Robot) WaitReady(timeout time.Duration) bool {
	deadline := time.NewTimer(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()

	for {
		if r.IsReady() {
			return true
		}
		if r.isStopped() {
			return false
		}

		select {
		case <-deadline.C:
			return false
		case <-ticker.C:
		case <-r.ctx.Done():
			return false
		}
	}
}

func (r *Robot) SendOperation(messageID pb.MESSAGE_ID, params map[string]interface{}) (*OperationSendResult, error) {
	if !r.IsReady() {
		return nil, fmt.Errorf("robot is not ready")
	}

	var msg proto.Message
	manualProto := robotRouter.GetManualProtoByMessageID(messageID)
	if manualProto != nil {
		msg = proto.Clone(manualProto)
	} else {
		var err error
		msg, err = robotRouter.BuildProtoMessageByMessageID(messageID)
		if err != nil {
			return nil, err
		}
	}

	if len(params) > 0 {
		if err := robotUtils.BuildMessageWithParams(msg, params); err != nil {
			return nil, err
		}
	}

	if err := r.Send(uint32(messageID), msg); err != nil {
		return nil, err
	}

	return &OperationSendResult{
		RequestMsgID:   uint32(messageID),
		ResponseMsgID:  uint32(messageID) + 1,
		MessageName:    messageID.String(),
		ExpectedStatus: "sent",
	}, nil
}

// Send 发送消息
func (r *Robot) Send(msgID uint32, pbMsg proto.Message) error {
	if r.conn == nil {
		return fmt.Errorf("connection not established")
	}
	if r.isStopped() {
		return fmt.Errorf("robot already stopped")
	}

	data, err := robotRouter.EncodeMessage(msgID, pbMsg)
	if err != nil {
		return err
	}

	prevState := r.state.Load()
	atomic.StoreInt32(&r.waitingMessageID, int32(msgID+1))
	r.waitingReqMsgID.Store(msgID)
	r.waitingStartedAt.Store(time.Now().UnixNano())
	r.state.Store(StateWaitingResp)
	done := make(chan error, 1)
	if err = r.enqueueWrite(&outboundMessage{
		msgType: websocket.BinaryMessage,
		payload: data,
		msgID:   msgID,
		done:    done,
	}); err != nil {
		atomic.StoreInt32(&r.waitingMessageID, 0)
		r.waitingReqMsgID.Store(0)
		r.waitingStartedAt.Store(0)
		r.state.Store(prevState)
		return err
	}

	select {
	case err = <-done:
		if err != nil {
			atomic.StoreInt32(&r.waitingMessageID, 0)
			r.waitingReqMsgID.Store(0)
			r.waitingStartedAt.Store(0)
			r.state.Store(prevState)
			return err
		}
		if r.monitor != nil {
			r.monitor.SystemStats.RecordMessageSent()
		}
		robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=send_message status=success req=%d(%s) wait=%d(%s) bytes=%d", msgID, pb.MESSAGE_ID(msgID).String(), msgID+1, pb.MESSAGE_ID(msgID+1).String(), len(data)))
		return nil
	case <-r.ctx.Done():
		return fmt.Errorf("robot already stopped")
	}
}

func (r *Robot) Stop() {
	r.once.Do(func() {
		r.state.Store(StateStopped)
		r.cancel()
		if r.conn != nil {
			time.Sleep(100 * time.Millisecond)
			if r.conn != nil {
				_ = r.conn.Close()
			}
		}
		robotLogger.InfoWithRobot(r, fmt.Sprintf("phase=robot_stop status=success uptimeMs=%d", time.Since(r.startTime).Milliseconds()))
		if r.monitor != nil {
			r.monitor.SystemStats.RemoveRobot()
		}
	})
}

// sendHeartbeat 发送心跳
type outboundMessage struct {
	msgType int
	payload []byte
	msgID   uint32
	done    chan error
}

func (r *Robot) enqueueWrite(out *outboundMessage) error {
	select {
	case <-r.ctx.Done():
		return fmt.Errorf("robot already stopped")
	case r.writeChannel <- out:
		return nil
	}
}

func (r *Robot) sendHeartbeat() {
	if r.isStopped() {
		return
	}

	data, err := robotRouter.EncodeMessage(uint32(pb.MESSAGE_ID_HEART_REQ), &pb.HeartReq{})
	if err != nil {
		robotLogger.ErrorWithRobot(r, fmt.Sprintf("encode heartbeat failed: %v", err))
		return
	}

	select {
	case <-r.ctx.Done():
		return
	case r.writeChannel <- &outboundMessage{
		msgType: websocket.BinaryMessage,
		payload: data,
		msgID:   uint32(pb.MESSAGE_ID_HEART_REQ),
	}:
		if r.monitor != nil {
			r.monitor.SystemStats.RecordMessageSent()
		}
	default:
		robotLogger.InfoWithRobot(r, "heartbeat dropped: write queue is full")
	}
}

func (r *Robot) GetName() string {
	return r.Name
}

func (r *Robot) GetAccount() string {
	return r.Account
}

func (r *Robot) GetServerID() int32 {
	return r.ServerID
}

func (r *Robot) GetStateName() string {
	return stateName(r.state.Load())
}

func (r *Robot) CostSinceLoginRequest() time.Duration {
	startedAtNs := r.loginReqSentAt.Load()
	if startedAtNs <= 0 {
		return 0
	}
	return time.Since(time.Unix(0, startedAtNs))
}

func (r *Robot) CostSinceLoadSceneRequest() time.Duration {
	startedAtNs := r.loadSceneReqSentAt.Load()
	if startedAtNs <= 0 {
		return 0
	}
	return time.Since(time.Unix(0, startedAtNs))
}

func stateName(state int32) string {
	switch state {
	case StateConnected:
		return "connected"
	case StateLoggedIn:
		return "logged_in"
	case StateSceneLoaded:
		return "scene_loaded"
	case StateReady:
		return "ready"
	case StateWaitingResp:
		return "waiting_resp"
	case StateStopped:
		return "stopped"
	default:
		return fmt.Sprintf("unknown(%d)", state)
	}
}

func (r *Robot) SetAuthed(authed bool) {
	if !authed || r.isStopped() {
		return
	}
	for {
		current := r.state.Load()
		if current >= StateLoggedIn {
			return
		}
		if r.state.CompareAndSwap(current, StateLoggedIn) {
			return
		}
	}
}

func (r *Robot) SendLoadSceneOverRequest() {
	r.loadSceneReqSentAt.Store(time.Now().UnixNano())
	if err := r.sendLoadSceneOverRequest(); err != nil {
		r.loadSceneReqSentAt.Store(0)
		robotLogger.ErrorWithRobot(r, fmt.Sprintf("send load scene over request failed: %v", err))
	}
}

func (r *Robot) sendLoadSceneOverRequest() error {
	err := r.Send(uint32(pb.MESSAGE_ID_LOAD_SCENE_OVER_REQ), &pb.LoadSceneOverReq{})
	return err
}

func (r *Robot) SetSceneLoaded(loaded bool) {
	if !loaded || r.isStopped() {
		return
	}
	r.state.Store(StateSceneLoaded)
	r.state.Store(StateReady)
}

func (r *Robot) isStopped() bool {
	return r.state.Load() == StateStopped
}

func (r *Robot) moduleByMessageID(messageID pb.MESSAGE_ID) string {
	moduleGroups, ok := r.runtimeModuleMessages()
	if !ok {
		return "unknown"
	}
	for _, group := range moduleGroups {
		for _, id := range group.MessageIDs {
			if pb.MESSAGE_ID(id) == messageID {
				return group.Module
			}
		}
	}
	return "unknown"
}

func (r *Robot) describeRuntimePlan(groups []robotConfig.RobotRunModule) string {
	if len(groups) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(groups))
	for _, group := range groups {
		msgParts := make([]string, 0, len(group.MessageIDs))
		for _, msgID := range group.MessageIDs {
			msgParts = append(msgParts, fmt.Sprintf("%d(%s)", msgID, pb.MESSAGE_ID(msgID).String()))
		}
		parts = append(parts, fmt.Sprintf("%s:[%s]", group.Module, strings.Join(msgParts, ",")))
	}
	return strings.Join(parts, "; ")
}
