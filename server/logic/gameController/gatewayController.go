package gameController

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/drop/GoServer/server/logic/gameConfig"
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
	RegisterSidewayMessageHandler(enum.MSG_TYPE_SIDEWAY, pb.MESSAGE_ID_GET_OTHER_PLAYER_BASIC_INFO_REQ, &pb.GetOtherPlayerBasicInfoReq{}, GetOtherPlayerBasicInfoSidewayHandler)
	RegisterSidewayMessageHandler(enum.MSG_TYPE_SIDEWAY, pb.MESSAGE_ID_GET_HERO_DETAIL_INFO_REQ, &pb.GetHeroDetailInfoReq{}, GetHeroDetailInfoSidewayHandler)
	RegisterSidewayMessageHandler(enum.MSG_TYPE_SIDEWAY, pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_REQ, &pb.GetEquipmentDetailInfoReq{}, GetEquipmentDetailInfoSidewayHandler)
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
	nodeId, err := ServerNodeService.GetGameNodeIdByUserId(user.UserId)
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
		nodeId, err := ServerNodeService.GetGameNodeIdByUserId(user.UserId)
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

func decodeSidewayClientReq(message proto.Message, user *logicCommon.GatewayPlayerInfo, req proto.Message, respID pb.MESSAGE_ID) bool {
	clientReq, ok := message.(*rpcPb.ClientReq)
	if !ok || clientReq == nil {
		messageSender.SendErrorMessage(user, respID, pb.ERROR_CODE_PB_CONV_ERROR)
		logger.ErrorBySprintf("[gateway] sideway decode convert clientReq error respMsgId:%d", respID)
		return false
	}
	if len(clientReq.Data) < 4 {
		messageSender.SendErrorMessage(user, respID, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		logger.ErrorBySprintf("[gateway] sideway decode message length error respMsgId:%d", respID)
		return false
	}
	if err := proto.Unmarshal(clientReq.Data[4:], req); err != nil {
		messageSender.SendErrorMessage(user, respID, pb.ERROR_CODE_PB_CONV_ERROR)
		logger.ErrorBySprintf("[gateway] sideway decode unmarshal error respMsgId:%d err:%v", respID, err)
		return false
	}
	return true
}

func GetOtherPlayerBasicInfoSidewayHandler(message proto.Message, user *logicCommon.GatewayPlayerInfo) {
	req := &pb.GetOtherPlayerBasicInfoReq{}
	if !decodeSidewayClientReq(message, user, req, pb.MESSAGE_ID_GET_OTHER_PLAYER_BASIC_INFO_RESP) {
		return
	}
	GetOtherPlayerBasicInfoHandler(req, user)
}

func GetHeroDetailInfoSidewayHandler(message proto.Message, user *logicCommon.GatewayPlayerInfo) {
	req := &pb.GetHeroDetailInfoReq{}
	if !decodeSidewayClientReq(message, user, req, pb.MESSAGE_ID_GET_HERO_DETAIL_INFO_RESP) {
		return
	}
	GetHeroDetailInfoHandler(req, user)
}

func GetEquipmentDetailInfoSidewayHandler(message proto.Message, user *logicCommon.GatewayPlayerInfo) {
	req := &pb.GetEquipmentDetailInfoReq{}
	if !decodeSidewayClientReq(message, user, req, pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_RESP) {
		return
	}
	GetEquipmentDetailInfoHandler(req, user)
}

func GetOtherPlayerBasicInfoHandler(req *pb.GetOtherPlayerBasicInfoReq, user *logicCommon.GatewayPlayerInfo) {
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
					Id:      heroInfo.Uid,
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

func GetHeroDetailInfoHandler(req *pb.GetHeroDetailInfoReq, user *logicCommon.GatewayPlayerInfo) {
	if req == nil {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_HERO_DETAIL_INFO_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	playerRedisInfo := logicCommon.GetPlayerRedisInfo(req.UserId)
	if playerRedisInfo == nil || playerRedisInfo.BattleInfo == nil || playerRedisInfo.BattleInfo.FormationHeroes == nil {
		logger.ErrorBySprintf("[gateway] GetHeroDetailInfoHandler redis info not found userId:%d heroOwnId:%d", req.UserId, req.HeroId)
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_HERO_DETAIL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	heroInfo := playerRedisInfo.BattleInfo.FormationHeroes[req.GetHeroId()]
	if heroInfo == nil {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_HERO_DETAIL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	type equipmentBasicRow struct {
		EquipmentOwnID int64 `gorm:"column:equipment_own_id"`
		EquipmentID    int32 `gorm:"column:equipment_id"`
		Level          int32 `gorm:"column:level"`
		SlotType       int32 `gorm:"column:slot_type"`
		SlotIndex      int32 `gorm:"column:slot_index"`
		StarLevel      int32 `gorm:"column:star_level"`
		StrongLevel    int32 `gorm:"column:strong_level"`
	}
	var equipmentRows []*equipmentBasicRow
	err := easyDB.GetPlayerDB().
		Model(&model.EquipmentEntity{}).
		Select("equipment_own_id", "equipment_id", "level", "slot_type", "slot_index", "star_level", "strong_level").
		Where("hero_own_id = ? AND is_deleted = ?", req.HeroId, false).
		Find(&equipmentRows).Error
	if err != nil {
		logger.ErrorBySprintf("[gateway] GetHeroDetailInfoHandler load equipments error heroOwnId:%d err:%v", req.HeroId, err)
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_HERO_DETAIL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	equipments := make([]*pb.EquipmentBasicInfo, 0)
	for _, row := range equipmentRows {
		if row == nil {
			continue
		}
		equipments = append(equipments, &pb.EquipmentBasicInfo{
			EquipmentOwnId: row.EquipmentOwnID,
			EquipmentId:    row.EquipmentID,
			Level:          row.Level,
			SlotType:       row.SlotType,
			SlotIndex:      row.SlotIndex,
			StarLevel:      row.StarLevel,
			StrongLevel:    row.StrongLevel,
		})
	}

	var accessory *pb.AccessoryDetail
	type accessoryRow struct {
		AccessoryId    int32 `gorm:"column:accessory_id"`
		AccessoryLevel int32 `gorm:"column:accessory_level"`
		Num            int32 `gorm:"column:num"`
		HeroOwnId      int64 `gorm:"column:hero_own_id"`
	}
	accessoryData := &accessoryRow{}
	err = easyDB.GetPlayerDB().
		Model(&model.AccessoryEntity{}).
		Select("accessory_id", "accessory_level", "num", "hero_own_id").
		Where("user_id = ? AND hero_own_id = ?", req.UserId, req.HeroId).
		Limit(1).
		Take(accessoryData).Error
	if err == nil {
		type userLevelRow struct {
			Level int32 `gorm:"column:level"`
		}
		accessory = &pb.AccessoryDetail{
			AccessoryId:    accessoryData.AccessoryId,
			AccessoryLevel: accessoryData.AccessoryLevel,
			AccessoryNum:   accessoryData.Num,
			HeroOwnId:      accessoryData.HeroOwnId,
			Power:          gameConfig.GetAccessoryPower(accessoryData.AccessoryId, accessoryData.AccessoryLevel, playerRedisInfo.BasicInfo.MainCityLevel),
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.ErrorBySprintf("[gateway] GetHeroDetailInfoHandler load accessory error userId:%d heroOwnId:%d err:%v", req.UserId, req.HeroId, err)
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_HERO_DETAIL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	resp := &pb.GetHeroDetailInfoResp{
		BasicInfo: &pb.CharactorBasicInfo{
			Attrs:         heroInfo.Attr,
			SkillIds:      &pb.SkillsInfo{BasicSkill: heroInfo.NormalAtk, SkillList: heroInfo.Skill},
			UnitsId:       heroInfo.Units,
			AttackRange:   heroInfo.AttackRange,
			PetBattleInfo: heroInfo.PetInfo,
		},
		Equips:    equipments,
		Accessory: accessory,
	}
	data, err := proto.Marshal(resp)
	if err != nil {
		logger.ErrorBySprintf("[platform] Send error message error")
		return
	}
	backMessage := &rpcPb.BackwardClientMessage{
		SessionId: user.GetSession().GetID(),
		MsgId:     int32(pb.MESSAGE_ID_GET_HERO_DETAIL_INFO_RESP),
		Payload:   data,
	}
	messageSender.SendMessage(user, pb.MESSAGE_ID_GET_HERO_DETAIL_INFO_RESP, backMessage)
}

func GetEquipmentDetailInfoHandler(req *pb.GetEquipmentDetailInfoReq, user *logicCommon.GatewayPlayerInfo) {
	if req == nil || req.GetEquipmentOwnId() <= 0 {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	playerRedisInfo := logicCommon.GetPlayerBasicInfoFromRedis(req.UserId)
	if playerRedisInfo == nil {
		logger.ErrorBySprintf("[gateway] GetEquipmentDetailInfoHandler redis info not found userId:%d", req.UserId)
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	type equipmentDetailRow struct {
		EquipmentOwnID int64  `gorm:"column:equipment_own_id"`
		EquipmentID    int32  `gorm:"column:equipment_id"`
		HeroOwnID      int64  `gorm:"column:hero_own_id"`
		SlotType       int32  `gorm:"column:slot_type"`
		SlotIndex      int32  `gorm:"column:slot_index"`
		Level          int32  `gorm:"column:level"`
		StarLevel      int32  `gorm:"column:star_level"`
		ForgeLevel     int32  `gorm:"column:forge_level"`
		SetID          int32  `gorm:"column:set_id"`
		IsLocked       bool   `gorm:"column:is_locked"`
		StrongLevel    int32  `gorm:"column:strong_level"`
		AttributeAffix string `gorm:"column:attribute_affix"`
		SkillAffix     string `gorm:"column:skill_affix"`
	}
	equipmentData := &equipmentDetailRow{}
	err := easyDB.GetPlayerDB().
		Model(&model.EquipmentEntity{}).
		Select("equipment_own_id", "equipment_id", "hero_own_id", "slot_type", "slot_index", "level", "star_level", "forge_level", "set_id", "is_locked", "strong_level", "attribute_affix", "skill_affix").
		Where("equipment_own_id = ? AND is_deleted = ?", req.GetEquipmentOwnId(), false).
		Take(equipmentData).Error
	if err != nil {
		messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_RESP, pb.ERROR_CODE_EQUIPMENT_NOT_FOUND)
		return
	}

	baseStats := make(map[int32]int32)
	if levelAttrCfg := gameConfig.GetEquipmentLevelAttrCfg(equipmentData.EquipmentID, equipmentData.Level); levelAttrCfg != nil {
		for _, attr := range levelAttrCfg.Attributes {
			baseStats[attr.AttrID] += attr.Value
		}
	}
	if baseCfg := gameConfig.GetEquipmentBaseCfgByEquipmentID(equipmentData.EquipmentID); baseCfg != nil {
		if strongCfg := gameConfig.GetEquipEnhanceCfg(model.GetEquipmentStrongId(baseCfg)); strongCfg != nil {
			for i, attrID := range strongCfg.Attr {
				baseStats[attrID] += strongCfg.AttrNum[i] * equipmentData.StrongLevel
			}
		}
	}

	type equipmentAttributeAffix struct {
		AffixID   int32 `json:"affixId"`
		AttrID    int32 `json:"attrId"`
		StatValue int32 `json:"statValue"`
	}
	type equipmentSkillAffix struct {
		SkillAffixID int32 `json:"skillAffixId"`
		SkillID      int32 `json:"skillId"`
		SkillLevel   int32 `json:"skillLevel"`
	}

	attributeAffixes := make([]equipmentAttributeAffix, 0)
	if equipmentData.AttributeAffix != "" {
		if err = json.Unmarshal([]byte(equipmentData.AttributeAffix), &attributeAffixes); err != nil {
			logger.ErrorBySprintf("[gateway] GetEquipmentDetailInfoHandler parse attribute affix error equipmentOwnId:%d err:%v", req.GetEquipmentOwnId(), err)
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
	}
	attributeAffixPB := make([]*pb.EquipmentAffixInfo, 0, len(attributeAffixes))
	attrs := make(map[int32]int64)
	for _, affix := range attributeAffixes {
		attributeAffixPB = append(attributeAffixPB, &pb.EquipmentAffixInfo{
			AffixId:   affix.AffixID,
			StatType:  affix.AttrID,
			StatValue: affix.StatValue,
		})
		// 与 equipmentScore 保持一致：相同属性后写覆盖
		attrs[affix.AttrID] = int64(affix.StatValue)
	}
	for k, v := range baseStats {
		if _, ok := attrs[k]; ok {
			attrs[k] += int64(v)
		} else {
			attrs[k] = int64(v)
		}
	}

	skillAffixes := make([]equipmentSkillAffix, 0)
	if equipmentData.SkillAffix != "" {
		if err = json.Unmarshal([]byte(equipmentData.SkillAffix), &skillAffixes); err != nil {
			logger.ErrorBySprintf("[gateway] GetEquipmentDetailInfoHandler parse skill affix error equipmentOwnId:%d err:%v", req.GetEquipmentOwnId(), err)
			messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
	}
	skillAffixPB := make([]*pb.EquipmentSkillAffixInfo, 0, len(skillAffixes))
	for _, affix := range skillAffixes {
		if affix.SkillAffixID == 0 {
			continue
		}
		skillAffixPB = append(skillAffixPB, &pb.EquipmentSkillAffixInfo{
			SkillAffixId: affix.SkillAffixID,
			SkillId:      affix.SkillID,
			SkillLevel:   affix.SkillLevel,
		})
	}

	var setInfo *pb.EquipmentSetInfo
	if equipmentData.SetID > 0 && equipmentData.HeroOwnID > 0 {
		setCfg := gameConfig.GetEquipmentSetCfg(equipmentData.SetID)
		if setCfg != nil {
			var setCount int64
			countErr := easyDB.GetPlayerDB().
				Model(&model.EquipmentEntity{}).
				Where("hero_own_id = ? AND set_id = ? AND is_deleted = ?", equipmentData.HeroOwnID, equipmentData.SetID, false).
				Count(&setCount).Error
			if countErr != nil {
				logger.ErrorBySprintf("[gateway] GetEquipmentDetailInfoHandler calc set count error heroOwnId:%d setId:%d err:%v", equipmentData.HeroOwnID, equipmentData.SetID, countErr)
				messageSender.SendErrorMessage(user, pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
				return
			}
			if setCount >= 2 {
				setInfo = &pb.EquipmentSetInfo{
					SetId:      equipmentData.SetID,
					PieceCount: int32(setCount),
				}
				activeStats := make(map[int32]int32)
				for i, level := range setCfg.SetLevels {
					if int32(setCount) >= level {
						if i < len(setCfg.SkillIDs) && setCfg.SkillIDs[i] > 0 {
							activeStats[setCfg.SkillIDs[i]] = 1
							setInfo.ActivePieceCount = level
						}
					}
				}
				setInfo.ActiveStats = activeStats
			}
		}
	}

	resp := &pb.GetEquipmentDetailInfoResp{
		Info: &pb.EquipmentDetailInfo{
			EquipmentOwnId: equipmentData.EquipmentOwnID,
			EquipmentId:    equipmentData.EquipmentID,
			HeroOwnId:      equipmentData.HeroOwnID,
			SlotType:       equipmentData.SlotType,
			SlotIndex:      equipmentData.SlotIndex,
			Level:          equipmentData.Level,
			StarLevel:      equipmentData.StarLevel,
			ForgeLevel:     equipmentData.ForgeLevel,
			SetId:          equipmentData.SetID,
			IsLocked:       equipmentData.IsLocked,
			BaseStats:      baseStats,
			AttributeAffix: attributeAffixPB,
			SkillAffix:     skillAffixPB,
			SetInfo:        setInfo,
			Power:          int64(gameConfig.GetAttrMapPower(playerRedisInfo.MainCityLevel, attrs)),
			StrongLevel:    equipmentData.StrongLevel,
		},
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		logger.ErrorBySprintf("[platform] Send error message error")
		return
	}
	backMessage := &rpcPb.BackwardClientMessage{
		SessionId: user.GetSession().GetID(),
		MsgId:     int32(pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_RESP),
		Payload:   data,
	}
	messageSender.SendMessage(user, pb.MESSAGE_ID_GET_EQUIPMENT_DETAIL_INFO_RESP, backMessage)
}
