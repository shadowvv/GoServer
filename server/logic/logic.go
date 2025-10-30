package logic

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/logic/platform"
	"google.golang.org/protobuf/proto"
)

var loginMsgProcessor = &LoginMessageProcessor{}
var sceneMsgProcessor = &SceneMessageProcessor{}
var serviceMsgProcessor = &ServiceMessageProcessor{}
var innerMsgProcessor = &InnerMessageProcessor{}

type MessageHandleFunc func(msgId uint32, message proto.Message, user logicInterface.UserBaseInterface)

func RegisterProcessor() {
	platform.RegisterProcessor(enum.MSG_TYPE_LOGIN, loginMsgProcessor)
	platform.RegisterProcessor(enum.MSG_TYPE_PLAYER, sceneMsgProcessor)
	platform.RegisterProcessor(enum.MSG_TYPE_SERVICE, serviceMsgProcessor)
	platform.RegisterProcessor(enum.MSG_TYPE_INNER, innerMsgProcessor)
}

func RegisterProcess(msgType, msgID uint32, msg proto.Message, h MessageHandleFunc) {

	platform.RegisterProcess(msgType, msgID, msg)
	switch msgType {
	case enum.MSG_TYPE_LOGIN:
		loginMsgProcessor.RegisterProcess(msgID, msg, h)
	case enum.MSG_TYPE_PLAYER:
		loginMsgProcessor.RegisterProcess(msgID, msg, h)
	case enum.MSG_TYPE_SERVICE:
		loginMsgProcessor.RegisterProcess(msgID, msg, h)
	case enum.MSG_TYPE_INNER:
		loginMsgProcessor.RegisterProcess(msgID, msg, h)
	}

}
