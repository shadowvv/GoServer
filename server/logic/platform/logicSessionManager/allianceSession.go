package logicSessionManager

import (
	"fmt"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type AllianceSession struct {
	UserId        int64
	AllianceId    int64
	BackMessageId int32
	ErrorCode     int32

	sender logicCommon.GrpcSenderInterface[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage]
	codec  serviceInterface.CodecInterface
}

var _ serviceInterface.SessionInterface = (*AllianceSession)(nil)
var _ logicCommon.UserBaseInterface = (*AllianceSession)(nil)

func NewAllianceSession(
	codec serviceInterface.CodecInterface,
	userId int64,
	allianceId int64,
	backMessageId int32,
	sender logicCommon.GrpcSenderInterface[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage],
) *AllianceSession {
	return &AllianceSession{
		UserId:        userId,
		AllianceId:    allianceId,
		BackMessageId: backMessageId,
		sender:        sender,
		codec:         codec,
	}
}

func (s *AllianceSession) SendAndClose(msgId int32, msg proto.Message) {
	// social session is virtual and does not own a real transport connection.
}

func (s *AllianceSession) Send(msgId int32, message proto.Message) {
	frame, err := s.codec.Marshal(msgId, message)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] send messageId:%d userId:%d marshal error: %v", msgId, s.UserId, err))
		return
	}
	if int32(len(frame)) > 256*1024 {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] userId:%d send messageId:%d msg too large: %d", s.UserId, msgId, len(frame)))
		return
	}

	resp := &rpcPb.BackwardSocialMessage{
		UserId:        s.UserId,
		MsgId:         rpcPb.RPC_MESSAGE_ID(msgId),
		BackMessageId: s.BackMessageId,
		ErrorCode:     s.ErrorCode,
		Payload:       frame,
	}
	if err = s.sender.Send(resp); err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] userId:%d send messageId:%d error: %v", s.UserId, msgId, err))
	}
}

func (s *AllianceSession) SendPushToUser(userId int64, msgId int32, message proto.Message) {
	frame, err := s.codec.Marshal(msgId, message)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] push marshal failed userId:%d msgId:%d err:%v", userId, msgId, err))
		return
	}
	resp := &rpcPb.BackwardSocialMessage{
		UserId:        userId,
		MsgId:         rpcPb.RPC_MESSAGE_ID(msgId),
		BackMessageId: 0,
		ErrorCode:     0,
		Payload:       frame,
	}
	if err = s.sender.Send(resp); err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] push send failed userId:%d msgId:%d err:%v", userId, msgId, err))
	}
}

func (s *AllianceSession) Close() {
}

func (s *AllianceSession) GetID() int64 {
	return s.UserId
}

func (s *AllianceSession) IsActive() bool {
	return true
}

func (s *AllianceSession) GetUserId() int64 {
	return s.UserId
}

func (s *AllianceSession) GetUserAccount() string {
	return ""
}

func (s *AllianceSession) GetUserServerId() int32 {
	return 0
}

func (s *AllianceSession) GetSession() serviceInterface.SessionInterface {
	return s
}

func (s *AllianceSession) GetNodeId() int32 {
	return 0
}

func (s *AllianceSession) GetSceneId() int32 {
	return 0
}
