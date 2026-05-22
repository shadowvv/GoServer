package logicSessionManager

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

type GameSession struct {
	SessionId      int64
	UserId         int64
	acceptor       serviceInterface.AcceptorInterface
	codec          serviceInterface.CodecInterface
	sender         logicCommon.GrpcSenderInterface[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage]
	offlineTimeout time.Duration
	LastActiveTime atomic.Int64
}

var _ serviceInterface.SessionInterface = (*GameSession)(nil)

func NewGameSession(codec serviceInterface.CodecInterface, sessionId int64, userId int64, acceptor serviceInterface.AcceptorInterface, offlineTimeout time.Duration) *GameSession {
	s := &GameSession{
		SessionId:      sessionId,
		UserId:         userId,
		acceptor:       acceptor,
		codec:          codec,
		offlineTimeout: offlineTimeout,
	}
	return s
}

func (g *GameSession) BindSender(sender logicCommon.GrpcSenderInterface[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage]) {
	g.sender = sender
}

func (g *GameSession) Start() {
	go g.heartbeat()
}

func (g *GameSession) Send(msgId int32, message proto.Message) {
	frame, err := g.codec.Marshal(msgId, message)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] send messageId:%d sessionId:%d marshal error: %v", msgId, g.SessionId, err))
		return
	}
	if int32(len(frame)) > 1024*1024*2 {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] sessionId:%d send messageId:%d msg too large: %d", g.SessionId, msgId, len(frame)))
		return
	}
	resp := &rpcPb.BackwardClientMessage{
		SessionId: g.SessionId,
		MsgId:     msgId,
		Payload:   frame,
	}
	err = g.sender.Send(resp)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] sessionId:%d send messageId:%d error: %v", g.SessionId, msgId, err))
		return
	}
}

func (g *GameSession) SendAndClose(msgId int32, message proto.Message) {
	frame, err := g.codec.Marshal(msgId, message)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] send messageId:%d sessionId:%d marshal error: %v", msgId, g.SessionId, err))
		return
	}
	if int32(len(frame)) > 256*1024 {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] sessionId:%d send messageId:%d msg too large: %d", g.SessionId, msgId, len(frame)))
		return
	}

	resp := &rpcPb.BackwardClientMessage{
		SessionId:    g.SessionId,
		MsgId:        msgId,
		Payload:      frame,
		CloseSession: true,
	}
	err = g.sender.Send(resp)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] sessionId:%d send messageId:%d error: %v", g.SessionId, msgId, err))
		return
	}
}

func (g *GameSession) internalClose() {
	g.acceptor.OnConnectionTimeout(g)
}

// ------------------ heartbeat ------------------
func (g *GameSession) heartbeat() {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorWithZapFields(fmt.Sprintf("[net] heartbeat panic: %v, sessionId:%d", r, g.SessionId))
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if tool.UnixNowMilli()-g.LastActiveTime.Load() > g.offlineTimeout.Milliseconds() {
				logger.InfoWithSprintf(fmt.Sprintf("[net] sessionId:%d timeout", g.SessionId))
				g.internalClose()
				return
			}
		}
	}
}

func (g *GameSession) Close() {
}

func (g *GameSession) GetID() int64 {
	return g.SessionId
}

func (g *GameSession) IsActive() bool {
	return true
}
