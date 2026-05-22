package gameController

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/tool"
	"gorm.io/gorm"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/gatewayPlatform"
	"github.com/drop/GoServer/server/logic/platform/loginMutexService"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func RegisterGatewayMessage() {
	dispatcher.RegisterGatewayMessageHandler(enum.MSG_TYPE_LOGIN, onLoginMessage)
	dispatcher.RegisterGatewayMessageHandler(enum.MSG_TYPE_PLAYER, onPlayerMessage)
}

var userIdGenerator *tool.IdGenerator

func InitGatewayIdGenerator() {
	userIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_USER))
}

func onLoginMessage(message proto.Message, user *logicCommon.GatewayPlayerInfo) {
	clientReq, ok := message.(*rpcPb.ClientReq)
	if !ok {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		logger.ErrorBySprintf("[gateway] login convert clientReq error")
		return
	}

	loginMessage := &pb.LoginReq{}
	if err := proto.Unmarshal(clientReq.Data[4:], loginMessage); err != nil {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		logger.ErrorBySprintf("[gateway] login decode loginMessage error")
		return
	}
	logger.InfoWithSprintf("[gateway] begin login account:%s,serverId:%d", loginMessage.Account, loginMessage.ServerId)
	if user.Account != "_login" {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		logger.ErrorBySprintf("[gateway] login error init user error: account:%s,serverId:%d", loginMessage.Account, loginMessage.ServerId)
		return
	}

	if loginMessage.Account == "" {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		logger.ErrorBySprintf("[gateway] login error account is empty: account:%s,serverId:%d", loginMessage.Account, loginMessage.ServerId)
		return
	}

	logger.InfoWithSprintf("[gateway] login check token account:%s,serverId:%d", loginMessage.Account, loginMessage.ServerId)
	// 非审核服需要确定token，审核服是正式服跳转过去的，所以无法验证token
	if nodeConfig.Env != enum.ENV_AUDIT {
		token := loginMessage.Token
		_, err := dbService.RDB.Get(context.Background(), enum.GetLoginTokenKey(loginMessage.Account, token, loginMessage.ServerId)).Result()
		if err != nil {
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			logger.ErrorBySprintf("[gateway] login error redis check error : account:%s,serverId:%d,err:%v", loginMessage.Account, loginMessage.ServerId, err)
			return
		}
	}

	logger.InfoWithSprintf("[gateway] login enter mutex account:%s,serverId:%d", loginMessage.Account, loginMessage.ServerId)
	if ok = loginMutexService.EnterMutex(loginMessage.Account, user.Session.GetID()); !ok {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_ALREADY_ERROR)
		return
	}
	user.ServerId = loginMessage.ServerId
	user.Account = loginMessage.Account

	logger.InfoWithSprintf("[gateway] login check is new player account:%s,serverId:%d", user.Account, user.ServerId)
	isNewPlayer := false
	userEntity, err := easyDB.GetPlayerEntityByWhere[model.UserEntity](map[string]interface{}{"account": user.Account, "server_id": user.ServerId})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			isNewPlayer = true
			user.UserId = userIdGenerator.NextId()
		} else {
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			logger.ErrorBySprintf("[gateway] load player from db error : account:%s,serverId:%d,err:%v", loginMessage.Account, loginMessage.ServerId, err)
			return
		}
	} else {
		user.UserId = userEntity.UserId
	}

	if isNewPlayer {
		loginNewPlayer(user, clientReq)
	} else {
		loginOldPlayer(user, clientReq)
	}
}

func loginNewPlayer(user *logicCommon.GatewayPlayerInfo, clientReq *rpcPb.ClientReq) {
	logger.InfoWithSprintf("[gateway] login create new player account:%s,serverId:%d,playerId:%d", user.Account, user.ServerId, user.UserId)
	oldInfo := gatewayPlatform.GetSessionManager().GetPlayerByUserId(user.UserId)
	if oldInfo != nil {
		logger.ErrorBySprintf("new player generator repeated id playerId:%d,account:%s,serverId:%d", user.UserId, user.Account, user.ServerId)
		loginMutexService.ExitMutex(user.Account, user.Session.GetID())
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	logger.InfoWithSprintf("[gateway] login get game nodeId account:%s,serverId:%d,playerId:%d", user.Account, user.ServerId, user.UserId)
	nodeId, err := ServerNodeService.GetGameNodeId(user.UserId)
	if err != nil {
		loginMutexService.ExitMutex(user.Account, user.Session.GetID())
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		logger.ErrorBySprintf("[gateway] login error get game node error : account:%s,serverId:%d,nodeId:%d,err:%v", user.Account, user.ServerId, user.NodeId, err)
		return
	}
	user.NodeId = nodeId
	err = gatewayPlatform.GetSessionManager().BindPlayerWithNode(user)
	if err != nil {
		loginMutexService.ExitMutex(user.Account, user.Session.GetID())
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_REBIND_PLAYER_ERROR)
		logger.ErrorBySprintf("[gateway] login error rebind player error : account:%s,serverId:%d,nodeId:%d,err:%v", user.Account, user.ServerId, user.NodeId, err)
		return
	}

	logger.InfoWithSprintf("[gateway] login send login message to game account:%s,serverId:%d,playerId:%d", user.Account, user.ServerId, user.UserId)
	err = rpcController.SendMessageToGame(user.NodeId, &rpcPb.ForwardGameMessage{
		SessionId: user.Session.GetID(),
		PlayerId:  user.UserId,
		Payload:   clientReq.Data,
	})
	if err != nil {
		loginMutexService.ExitMutex(user.Account, user.Session.GetID())
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		logger.ErrorBySprintf("[gateway] login error get game client error : account:%s,serverId:%d,nodeId:%d,err:%v", user.Account, user.ServerId, user.NodeId, err)
		return
	}
	logger.InfoWithSprintf("[gateway] login success : account:%s,serverId:%d,nodeId:%d", user.Account, user.ServerId, user.NodeId)
	loginMutexService.ExitMutex(user.Account, user.Session.GetID())
}

func loginOldPlayer(user *logicCommon.GatewayPlayerInfo, clientReq *rpcPb.ClientReq) {
	logger.InfoWithSprintf("[gateway] login load old player account:%s,serverId:%d,playerId:%d", user.Account, user.ServerId, user.UserId)
	oldInfo := gatewayPlatform.GetSessionManager().GetPlayerByUserId(user.UserId)
	if oldInfo != nil {
		logger.InfoWithSprintf("[gateway] login player is online account:%s,serverId:%d,playerId:%d", user.Account, user.ServerId, user.UserId)
		if oldInfo.GetSession().GetID() == user.Session.GetID() {
			loginMutexService.ExitMutex(user.Account, user.Session.GetID())
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_ALREADY_ERROR)
			return
		}
		user.NodeId = oldInfo.GetNodeId()
		messageSender.CloseSessionWithError(oldInfo, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_REPLACE_PLAYER_ERROR)
		err := gatewayPlatform.GetSessionManager().ReplacePlayerWithNewInfo(user)
		if err != nil {
			loginMutexService.ExitMutex(user.Account, user.Session.GetID())
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_REPLACE_PLAYER_ERROR)
			return
		}
	} else {
		logger.InfoWithSprintf("[gateway] login player is not online account:%s,serverId:%d,playerId:%d", user.Account, user.ServerId, user.UserId)
		nodeId, err := ServerNodeService.GetGameNodeId(user.UserId)
		if err != nil {
			loginMutexService.ExitMutex(user.Account, user.Session.GetID())
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			logger.ErrorBySprintf("[gateway] login error get game node error : account:%s,serverId:%d,nodeId:%d,err:%v", user.Account, user.ServerId, user.NodeId, err)
			return
		}
		user.NodeId = nodeId
		err = gatewayPlatform.GetSessionManager().BindPlayerWithNode(user)
		if err != nil {
			loginMutexService.ExitMutex(user.Account, user.Session.GetID())
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_REBIND_PLAYER_ERROR)
			logger.ErrorBySprintf("[gateway] login error rebind player error : account:%s,serverId:%d,nodeId:%d,err:%v", user.Account, user.ServerId, user.NodeId, err)
			return
		}
	}

	err := rpcController.SendMessageToGame(user.NodeId, &rpcPb.ForwardGameMessage{
		SessionId: user.Session.GetID(),
		PlayerId:  user.UserId,
		Payload:   clientReq.Data,
	})
	if err != nil {
		loginMutexService.ExitMutex(user.Account, user.Session.GetID())
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		logger.ErrorBySprintf("[gateway] login error get game client error : account:%s,serverId:%d,nodeId:%d,err:%v", user.Account, user.ServerId, user.NodeId, err)
		return
	}
	logger.InfoWithSprintf("[gateway] login success : account:%s,serverId:%d,nodeId:%d", user.Account, user.ServerId, user.NodeId)
	loginMutexService.ExitMutex(user.Account, user.Session.GetID())
}

func onPlayerMessage(message proto.Message, user *logicCommon.GatewayPlayerInfo) {

	clientReq, ok := message.(*rpcPb.ClientReq)
	if !ok {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		logger.ErrorBySprintf("[gateway] player message convert clientReq error")
		return
	}
	if len(clientReq.Data) < 4 {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		logger.ErrorBySprintf("[gateway] player message length error : account:%s,serverId:%d,nodeId:%d", user.Account, user.ServerId, user.NodeId)
		return
	}
	msgID := binary.BigEndian.Uint32(clientReq.Data[:4])

	//TODO:后续优化到其它协程组
	if msgID == uint32(pb.MESSAGE_ID_GET_OTHER_PLAYER_BASIC_INFO_REQ) {
		if nodeConfig.Env == enum.ENV_AUDIT {
			// 审核服屏蔽请求他人信息的功能
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_OTHER_PLAYER_BASIC_INFO_RESP, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
			return
		}
		req := &pb.GetOtherPlayerBasicInfoReq{}
		err := proto.Unmarshal(clientReq.Data[4:], req)
		if err != nil {
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
			return
		}
		GetOtherPlayerBasicInfoHandle(req, user)
		return
	}

	err := rpcController.SendMessageToGame(user.NodeId, &rpcPb.ForwardGameMessage{
		SessionId: user.Session.GetID(),
		PlayerId:  user.UserId,
		Payload:   clientReq.Data,
	})
	if err != nil {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID(msgID+1), pb.ERROR_CODE_SYSTEM_ERROR)
		logger.ErrorBySprintf("[gateway] player message send message to game error : account:%s,serverId:%d,nodeId:%d,err:%v", user.Account, user.ServerId, user.NodeId, err)
		return
	}
}

func GetOtherPlayerBasicInfoHandle(req *pb.GetOtherPlayerBasicInfoReq, user *logicCommon.GatewayPlayerInfo) {
	other := logicCommon.GetPlayerRedisInfo(req.PlayerId)
	if other == nil {
		infos, err := easyDB.GetPlayerEntitiesByRaw[model.UserEntity](enum.SELECT_PLAYER_SQL_BY_ID_SQL, req.PlayerId)
		if err != nil || len(infos) == 0 {
			logger.ErrorWithZapFields("[arena] load player error", zap.Error(err))
			return
		}
		info := infos[0]
		player, err := LoadPlayer(info, false)
		if err != nil {
			logger.ErrorWithZapFields("[arena] load player error", zap.Error(err))
			return
		}
		player.RefreshPlayerBattleInfo()
		other = player.PlayerCacheInfo
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(req.PlayerId)
	if other != nil {
		resp := &pb.GetOtherPlayerBasicInfoResp{
			BasicInfo: &pb.PlayerBasicInfo{
				UserId:      other.BasicInfo.Id,
				ServerId:    other.BasicInfo.ServerId,
				NickName:    other.BasicInfo.Name,
				HeadId:      other.BasicInfo.HeadId,
				HeadFrameId: other.BasicInfo.FrameId,
				TitleId:     other.BasicInfo.Title,
				Level:       other.BasicInfo.MainCityLevel,
				BubbleId:    other.BasicInfo.BubbleId,
				ImageId:     other.BasicInfo.ImageId,
			},
			AllianceInfo: &pb.PlayerAllianceInfo{
				AllianceId:   allianceInfo.AllianceId,
				AllianceName: allianceInfo.AllianceName,
			},
		}
		resp.FormationDetails = make(map[int32]*pb.OtherPlayerFormationInfo)
		for formationId, formation := range other.BattleInfo.FormationInfo {
			formationPB := &pb.OtherPlayerFormationInfo{
				Heroes: make([]*pb.OtherPlayerHeroBasicInfo, 0),
			}
			formationPB.BattlePower = formation.BattlePower
			for _, heroId := range formation.Heroes {
				heroInfo := other.BattleInfo.FormationHeroes[heroId]
				if heroInfo == nil {
					continue
				}
				formationPB.Heroes = append(formationPB.Heroes, &pb.OtherPlayerHeroBasicInfo{
					HeroId:  int32(heroInfo.Id),
					Star:    heroInfo.Star,
					Level:   heroInfo.Level,
					ClassId: heroInfo.ClassId,
					Units:   heroInfo.Units,
				})
			}
			resp.FormationDetails[formationId] = formationPB
		}
		data, err := proto.Marshal(resp)
		if err != nil {
			logger.ErrorBySprintf("[platform] Send error message error")
			return
		}
		backMessage := &rpcPb.BackwardClientMessage{
			SessionId: user.GetSession().GetID(),
			MsgId:     int32(pb.MESSAGE_ID_GET_OTHER_PLAYER_BASIC_INFO_RESP),
			Payload:   data,
		}
		messageSender.SendMessage(user, pb.MESSAGE_ID_GET_OTHER_PLAYER_BASIC_INFO_RESP, backMessage)
		return
	}
	messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_OTHER_PLAYER_BASIC_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
}
