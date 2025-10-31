package logic

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform"
	"google.golang.org/protobuf/proto"
)

var loginMsgProcessor = NewLoginMessageProcessor(2)
var sceneMsgProcessor = &SceneMessageProcessor{}
var serviceMsgProcessor = &ServiceMessageProcessor{}
var innerMsgProcessor = &InnerMessageProcessor{}

type MessageHandleFunc func(message proto.Message, user logicInterface.UserBaseInterface)

func RegisterProcessor() {
	platform.RegisterProcessor(enum.MSG_TYPE_LOGIN, loginMsgProcessor)
	platform.RegisterProcessor(enum.MSG_TYPE_PLAYER, sceneMsgProcessor)
	platform.RegisterProcessor(enum.MSG_TYPE_SERVICE, serviceMsgProcessor)
	platform.RegisterProcessor(enum.MSG_TYPE_INNER, innerMsgProcessor)
}

func RegisterProcess(msgType uint32, msgID pb.MESSAGE_ID, msg proto.Message, h MessageHandleFunc) {

	messageId := int32(msgID)
	platform.RegisterProcess(msgType, messageId, msg)
	switch msgType {
	case enum.MSG_TYPE_LOGIN:
		loginMsgProcessor.RegisterProcess(messageId, h)
	case enum.MSG_TYPE_PLAYER:
		loginMsgProcessor.RegisterProcess(messageId, h)
	case enum.MSG_TYPE_SERVICE:
		loginMsgProcessor.RegisterProcess(messageId, h)
	case enum.MSG_TYPE_INNER:
		loginMsgProcessor.RegisterProcess(messageId, h)
	}

}
