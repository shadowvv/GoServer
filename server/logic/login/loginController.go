package login

import (
	"github.com/drop/GoServer/server/logic"
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func RegisterControllerMessage() {
	logic.RegisterProcess(enum.MSG_TYPE_LOGIN, pb.MESSAGE_ID_LOGIN_REQ, &pb.LoginReq{}, LoginReqHandle)
}

func LoginReqHandle(message proto.Message, user logicInterface.UserBaseInterface) {
	req := message.(*pb.LoginReq)
	if req.Account == "" {
		return
	}

	logger.Info("test login:", zap.String("account", req.Account))

	//userBasicInfo := db.Get()
	//if userBasicInfo == nil {
	//	userBasicInfo = createUser()
	//}
	//
	//loadBasicInfo()
	//
	//user.sendMessage(enum.MSG_TYPE_LOGIN, enum.MSG_ID_LOGIN_RSP, &pb.LoginRsp{
	//	UserId:    userBasicInfo.UserId,
	//	Account:   userBasicInfo.Account,
	//	SessionId: user.GetSessionId(),
	//})
}
