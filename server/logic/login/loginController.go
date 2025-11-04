package login

import (
	"github.com/drop/GoServer/server/logic"
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/logicInterface"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
	"sync"
)

func RegisterControllerMessage() {
	logic.RegisterProcess(enum.MSG_TYPE_LOGIN, pb.MESSAGE_ID_LOGIN_REQ, &pb.LoginReq{}, LoginReqHandle)
}

var loginMutexMap = &LoginMutexMap{
	accountMap: make(map[string]bool),
	sessionMap: make(map[int64]bool),
}

type LoginMutexMap struct {
	sync.Mutex
	accountMap map[string]bool
	sessionMap map[int64]bool
}

func (l *LoginMutexMap) enterMutex(account string, sessionId int64) bool {
	loginMutexMap.Lock()
	defer loginMutexMap.Unlock()
	if _, ok := loginMutexMap.accountMap[account]; ok {
		return false
	}
	if _, ok := loginMutexMap.sessionMap[sessionId]; ok {
		return false
	}
	loginMutexMap.accountMap[account] = true
	loginMutexMap.sessionMap[sessionId] = true
	return true
}

func (l *LoginMutexMap) exitMutex(account string, sessionId int64) {
	loginMutexMap.Lock()
	defer loginMutexMap.Unlock()
	delete(loginMutexMap.accountMap, account)
	delete(loginMutexMap.sessionMap, sessionId)
}

func LoginReqHandle(message proto.Message, user logicInterface.UserBaseInterface) {

	platform.InfoWithFunction(enum.FUNC_LOGIN, "login req", user)

	req, ok := message.(*pb.LoginReq)
	if !ok {
		platform.ErrorWithFunction(enum.FUNC_LOGIN, "message error", user, nil)
		return
	}
	if req.Account == "" {
		platform.ErrorWithFunction(enum.FUNC_LOGIN, "account is empty", user, nil)
		return
	}

	if ok := loginMutexMap.enterMutex(req.Account, user.GetSession().GetID()); !ok {
		platform.ErrorWithFunction(enum.FUNC_LOGIN, "account already login", user, nil)
		return
	}

	platform.InfoWithFunction(enum.FUNC_LOGIN, "account login", user)
	//TODO:判断用户是否在内存
	userModel, err := platform.GetByStringID[model.UserModel]("account", req.Account)
	if err != nil {
		loginMutexMap.exitMutex(req.Account, user.GetSession().GetID())
		platform.ErrorWithFunction(enum.FUNC_LOGIN, "get user error", user, err)
		return
	}

	platform.AddUser(userModel.UserId, user)
	userModel.LastLoginTime = tool.GetCurrentTimeMillis()
	err = platform.Save[model.UserModel](userModel)
	if err != nil {
		loginMutexMap.exitMutex(req.Account, user.GetSession().GetID())
		platform.ErrorWithFunction(enum.FUNC_LOGIN, "save user error", user, err)
		return
	}
	//platform.enterScene()

	err = user.GetSession().Send(int32(pb.MESSAGE_ID_LOGIN_RESP), &pb.LoginResp{})
	if err != nil {
		loginMutexMap.exitMutex(req.Account, user.GetSession().GetID())
		platform.ErrorWithFunction(enum.FUNC_LOGIN, "send login resp error", user, err)
		return
	}

	loginMutexMap.exitMutex(req.Account, user.GetSession().GetID())

	platform.InfoWithFunction(enum.FUNC_LOGIN, "login success", user)
}
