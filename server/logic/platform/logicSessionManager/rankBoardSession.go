package logicSessionManager

import (
	"fmt"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type RankBoardSession struct {
	UserId              int64
	RankBoardId         string
	GetRankBoardInfoIds []string
	BackMessageId       int32
	RespMsgId           rpcPb.RPC_MESSAGE_ID
	ErrorCode           int32

	sender logicCommon.GrpcSenderInterface[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage]
	codec  serviceInterface.CodecInterface
}

var _ serviceInterface.SessionInterface = (*RankBoardSession)(nil)

func (g *RankBoardSession) SendAndClose(msgId int32, msg proto.Message) {

}

func NewRankBoardSession(codec serviceInterface.CodecInterface, userId int64, rankBoardIds []string, backMessageId int32, respMsgId rpcPb.RPC_MESSAGE_ID, sender logicCommon.GrpcSenderInterface[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage]) *RankBoardSession {
	rankBoardId := ""
	if len(rankBoardIds) > 0 {
		rankBoardId = rankBoardIds[0]
	}
	s := &RankBoardSession{
		UserId:              userId,
		RankBoardId:         rankBoardId,
		GetRankBoardInfoIds: append([]string(nil), rankBoardIds...),
		codec:               codec,
		BackMessageId:       backMessageId,
		RespMsgId:           respMsgId,
		sender:              sender,
	}
	return s
}

func (g *RankBoardSession) Send(msgId int32, message proto.Message) {
	frame, err := g.codec.Marshal(msgId, message)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] send messageId:%d userId:%d marshal error: %v", msgId, g.UserId, err))
		return
	}
	if int32(len(frame)) > 256*1024 {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] userId:%d send messageId:%d msg too large: %d", g.UserId, msgId, len(frame)))
		return
	}
	respMsgID := rpcPb.RPC_MESSAGE_ID(msgId)
	resp := &rpcPb.BackwardRankBoardMessage{
		UserId:        g.UserId,
		MsgId:         respMsgID,
		BackMessageId: g.BackMessageId,
		ErrorCode:     g.ErrorCode,
		Payload:       frame,
	}
	err = g.sender.Send(resp)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] userId:%d send messageId:%d error: %v", g.UserId, msgId, err))
		return
	}
}

func (g *RankBoardSession) internalClose() {
}

func (g *RankBoardSession) Close() {
}

func (g *RankBoardSession) GetID() int64 {
	return 0
}

func (g *RankBoardSession) IsActive() bool {
	return true
}
