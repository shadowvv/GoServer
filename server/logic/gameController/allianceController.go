package gameController

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/socialPlatform"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/logic/socialService"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/wordFilter"
	"github.com/drop/GoServer/server/tool"
	"github.com/go-redis/redis/v8"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("social", &SocialController{})
}

type SocialController struct {
}

var _ LogicControllerInterface = (*SocialController)(nil)

// TODO:玩家离线时间处理,联盟聊天消息同步
func (s *SocialController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CREATE_ALLIANCE_REQ, &pb.CreateAllianceReq{}, CreateAllianceHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_REQ, &pb.ChangeAllianceBasicInfoReq{}, ChangeAllianceBasicInfoHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_SERVER_ALLIANCE_BASIC_INFO_REQ, &pb.GetServerAllianceBasicInfoReq{}, GetServerAllianceBasicInfoHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_APPLY_ALLIANCE_REQ, &pb.ApplyAllianceReq{}, ApplyAllianceHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_APPLY_LIST_REQ, &pb.GetApplyListReq{}, GetApplyListHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_REQ, &pb.GetPlayerAllianceInfoReq{}, GetPlayerAllianceInfoHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_REQ, &pb.ApproveAllianceApplyReq{}, ApproveAllianceApplyHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_REQ, &pb.KickMemberOutAllianceReq{}, KickMemberOutAllianceHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_REQ, &pb.ChangeMemberPositionReq{}, ChangeMemberPositionHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_LEAVE_ALLIANCE_REQ, &pb.LeaveAllianceReq{}, LeaveAllianceHandler, enum.FUNCTION_ID_LOCK_SYSTEM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ALLIANCE_SIGN_IN_REQ, &pb.AllianceSignInReq{}, AllianceSignInHandler, enum.FUNCTION_ID_LOCK_SYSTEM)

	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CREATE_ALLIANCE_RESP, &rpcPb.CreateAllianceResp{}, BackCreateAllianceFromSocialNode)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_ALLIANCE_BASIC_INFO_RESP, &rpcPb.ChangeAllianceBasicInfoResp{}, BackChangeAllianceBasicInfoFromSocialNode)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_INFO_RESP, &rpcPb.GetAllianceInfoResp{}, BackGetPlayerAllianceInfoFromSocialNode)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPLY_ALLIANCE_RESP, &rpcPb.ApplyAllianceResp{}, BackApplyAllianceFromSocialNode)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPROVE_ALLIANCE_APPLY_BATCH_RESP, &rpcPb.ApproveAllianceApplyResp{}, BackApproveAllianceApplyFromSocialNode)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_KICK_ALLIANCE_MEMBER_RESP, &rpcPb.KickAllianceMemberResp{}, BackKickMemberOutAllianceFromSocialNode)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_MEMBER_POSITION_RESP, &rpcPb.ChangeMemberPositionResp{}, BackChangeMemberPositionFromSocialNode)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_QUIT_ALLIANCE_RESP, &rpcPb.QuitAllianceResp{}, BackLeaveAllianceFromSocialNode)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_PUSH_ALLIANCE_CHANGED, &rpcPb.PushAllianceChanged{}, BackAllianceChangedFromSocialNode)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_ACTIVITY_ITEM_NUM_RESP, &rpcPb.GetAllianceActivityItemNumResp{}, BackGetAllianceActivityItemNumFromSocialNode)
}

func RegisterAllianceMessage() {
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CREATE_ALLIANCE_REQ, &rpcPb.CreateAllianceReq{}, CreateAllianceOnSocialNode)
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_ALLIANCE_BASIC_INFO_REQ, &rpcPb.ChangeAllianceBasicInfoReq{}, ChangeAllianceBasicInfoOnSocialNode)
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_INFO_REQ, &rpcPb.GetAllianceInfoReq{}, GetAllianceInfoOnSocialNode)
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPLY_ALLIANCE_REQ, &rpcPb.ApplyAllianceReq{}, ApplyAllianceOnSocialNode)
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPROVE_ALLIANCE_APPLY_BATCH_REQ, &rpcPb.ApproveAllianceApplyReq{}, ApproveAllianceApplyOnSocialNode)
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_KICK_ALLIANCE_MEMBER_REQ, &rpcPb.KickAllianceMemberReq{}, KickAllianceMemberOnSocialNode)
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_MEMBER_POSITION_REQ, &rpcPb.ChangeMemberPositionReq{}, ChangeMemberPositionOnSocialNode)
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_QUIT_ALLIANCE_REQ, &rpcPb.QuitAllianceReq{}, LeaveAllianceOnSocialNode)
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_ALLIANCE_ITEMS_OPERATION_REQ, &rpcPb.AllianceItemsOperationReq{}, AllianceItemsOperationOnSocialNode)
	RegisterAllianceMessageHandler(enum.MSG_TYPE_Alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_ACTIVITY_ITEM_NUM_REQ, &rpcPb.GetAllianceActivityItemNumReq{}, GetAllianceActivityItemNumOnSocialNode)
}

func ChangeAllianceBasicInfoHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ChangeAllianceBasicInfoReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if allianceInfo.AllianceId <= 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_ALLIANCE_NOT_IN_ALLIANCE)
		return
	}

	name := strings.TrimSpace(req.GetName())
	announce := strings.TrimSpace(req.GetAnnounce())
	notice := strings.TrimSpace(req.GetNotice())

	updateName := name != ""
	updateAnnounce := announce != ""
	updateNotice := notice != ""
	updateBadge := req.GetBadgeId() > 0

	if updateName {
		errorCode := CheckAllianceNameLegal(name)
		if errorCode != pb.ERROR_CODE_SUCCESS {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, errorCode)
			return
		}
	}
	if updateAnnounce {
		errorCode := CheckAllianceAnnounceLegal(announce)
		if errorCode != pb.ERROR_CODE_SUCCESS {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, errorCode)
			return
		}
	}
	if updateNotice {
		errorCode := CheckAllianceNoticeLegal(notice)
		if errorCode != pb.ERROR_CODE_SUCCESS {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, errorCode)
			return
		}
	}
	if updateName {
		result, err := itemService.CheckItemCount(player, gameConfig.GetChangeAllianceNameItem())
		if err != nil || !result {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		err = itemService.RemoveItem(player, gameConfig.GetChangeAllianceNameItem(), enum.ITEM_CHANGE_REASON_CREATE_ALLIANCE_NAME)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
	}

	err := rpcController.SendMessageToSocial(
		player.GetUserId(),
		allianceInfo.AllianceId,
		int32(pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP),
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_ALLIANCE_BASIC_INFO_REQ,
		&rpcPb.ChangeAllianceBasicInfoReq{
			OperatorUserId:      player.GetUserId(),
			AllianceId:          allianceInfo.AllianceId,
			UpdateName:          updateName,
			Name:                name,
			UpdateAnnounce:      updateAnnounce,
			Announce:            announce,
			UpdateBadgeId:       updateBadge,
			BadgeId:             req.GetBadgeId(),
			ApplyType:           req.GetApplyType(),
			PowerApplyCondition: req.GetPowerApplyCondition(),
			CityLevel:           req.GetCityLevel(),
			Notice:              notice,
			UpdateNotice:        updateNotice,
		},
	)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
}

func resetPlayerAllianceInfoWhenNotInAlliance(userID int64) {
	if userID <= 0 {
		return
	}
	logicCommon.UpdatePlayerAllianceInfo(&logicCommon.PlayerAllianceInfo{
		ArenaJoined:  false,
		RoundBestWin: 0,
		UserId:       userID,
		AllianceId:   0,
		AllianceName: "",
		JoinTime:     0,
	})
}

func ensureAllianceMembershipOnSocialNode(
	session *logicSessionManager.AllianceSession,
	alliance *socialService.AllianceModel,
	respMsgID rpcPb.RPC_MESSAGE_ID,
) bool {
	if session == nil {
		return false
	}
	if alliance != nil && alliance.HasMember(session.UserId) {
		return true
	}
	resetPlayerAllianceInfoWhenNotInAlliance(session.UserId)
	socialPlatform.SendErrorMessageBySession(session.GetSession(), respMsgID, pb.ERROR_CODE_ALLIANCE_NOT_IN_ALLIANCE)
	return false
}

func ChangeAllianceBasicInfoOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	req, ok := message.(*rpcPb.ChangeAllianceBasicInfoReq)
	if !ok {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if !ensureAllianceMembershipOnSocialNode(session, alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_ALLIANCE_BASIC_INFO_RESP) {
		return
	}
	code := alliance.ChangeAllianceBasicInfo(req)
	socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_ALLIANCE_BASIC_INFO_RESP, &rpcPb.ChangeAllianceBasicInfoResp{
		ErrorCode: int32(code),
	})
}

func BackChangeAllianceBasicInfoFromSocialNode(message proto.Message, player *model.PlayerModel) {
	resp, ok := message.(*rpcPb.ChangeAllianceBasicInfoResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if pb.ERROR_CODE(resp.ErrorCode) != pb.ERROR_CODE_SUCCESS {
		_ = itemService.AddItem(player, gameConfig.GetChangeAllianceNameItem(), enum.ITEM_CHANGE_REASON_CREATE_ALLIANCE_NAME_FILED)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE(resp.ErrorCode))
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_CHANGE_ALLIANCE_BASIC_INFO_RESP, &pb.ChangeAllianceBasicInfoResp{})
}

func GetServerAllianceBasicInfoHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.GetServerAllianceBasicInfoReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SERVER_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SERVER_ALLIANCE_BASIC_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	allianceName := strings.TrimSpace(req.GetAllianceName())
	alliances, code := loadServerAllianceBasicInfoFromRedis(player, allianceName)
	if code != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SERVER_ALLIANCE_BASIC_INFO_RESP, code)
		return
	}
	for _, alliance := range alliances {
		cfg := gameConfig.GetAllianceLevelCfg(alliance.Level)
		if cfg == nil {
			continue
		}
		alliance.MaxPlayerNum = cfg.Num
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_SERVER_ALLIANCE_BASIC_INFO_RESP, &pb.GetServerAllianceBasicInfoResp{
		BasicInfo: alliances,
	})
}

func loadServerAllianceBasicInfoFromRedis(player *model.PlayerModel, allianceName string) ([]*pb.AllianceBasicInfo, pb.ERROR_CODE) {
	if player == nil || player.User == nil || dbService.RDB == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	if allianceName != "" {
		return loadAllianceByNameFromRedis(player, allianceName)
	}

	serverID := player.User.GetServerId()
	indexKey := enum.GetServerAllianceSetKey(serverID)
	ctx := context.Background()
	const randomFloatCount = 5
	fetchLimit := int64(enum.GetServerAllianceInfoMaxCount + randomFloatCount)

	allianceIDs, err := dbService.RDB.ZRevRange(ctx, indexKey, 0, fetchLimit-1).Result()
	if err != nil {
		logger.ErrorBySprintf("[alliance] get server alliance set from redis failed serverId:%d err:%v", serverID, err)
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	if len(allianceIDs) == 0 {
		return []*pb.AllianceBasicInfo{}, pb.ERROR_CODE_SUCCESS
	}

	keys := make([]string, 0, len(allianceIDs))
	for _, id := range allianceIDs {
		allianceID, parseErr := strconv.ParseInt(id, 10, 64)
		if parseErr != nil || allianceID <= 0 {
			continue
		}
		keys = append(keys, enum.GetAllianceBasicInfoKey(allianceID))
	}
	if len(keys) == 0 {
		return []*pb.AllianceBasicInfo{}, pb.ERROR_CODE_SUCCESS
	}

	alliances, err := dbService.RDB.MGet(ctx, keys...).Result()
	if err != nil {
		logger.ErrorBySprintf("[alliance] get alliance basic info from redis failed serverId:%d err:%v", serverID, err)
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}

	playerPower := player.GetMainFormationPower()
	playerCityLevel := int32(1)
	if player.ArchitectureModel != nil {
		playerCityLevel = player.ArchitectureModel.GetMainLevel()
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	playerAllianceID := allianceInfo.AllianceId

	joinable := make([]*pb.AllianceBasicInfo, 0, fetchLimit)
	for _, value := range alliances {
		if value == nil {
			continue
		}
		var raw string
		switch v := value.(type) {
		case string:
			raw = v
		case []byte:
			raw = string(v)
		default:
			continue
		}
		if raw == "" {
			continue
		}
		alliance := &model.AllianceEntity{}
		if err = json.Unmarshal([]byte(raw), alliance); err != nil {
			continue
		}
		info := buildServerAllianceBasicInfo(alliance, playerAllianceID, playerPower, playerCityLevel, player.GetUserId())
		if info == nil {
			continue
		}
		joinable = append(joinable, info)
		if int64(len(joinable)) >= fetchLimit {
			break
		}
	}

	for i := len(joinable) - 1; i > 0; i-- {
		j := tool.RandInt(0, i)
		joinable[i], joinable[j] = joinable[j], joinable[i]
	}
	if len(joinable) > enum.GetServerAllianceInfoMaxCount {
		joinable = joinable[:enum.GetServerAllianceInfoMaxCount]
	}

	result := make([]*pb.AllianceBasicInfo, 0, enum.GetServerAllianceInfoMaxCount)
	result = append(result, joinable...)
	return result, pb.ERROR_CODE_SUCCESS
}

func loadAllianceByNameFromRedis(player *model.PlayerModel, allianceName string) ([]*pb.AllianceBasicInfo, pb.ERROR_CODE) {
	if player == nil || player.User == nil || dbService.RDB == nil || allianceName == "" {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	ctx := context.Background()
	serverID := player.User.GetServerId()
	nameIndexKey := enum.GetAllianceNameIndexKey(serverID)
	allianceIDRaw, err := dbService.RDB.HGet(ctx, nameIndexKey, allianceName).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return []*pb.AllianceBasicInfo{}, pb.ERROR_CODE_SUCCESS
		}
		logger.ErrorBySprintf("[alliance] get alliance name index failed serverId:%d name:%s err:%v", serverID, allianceName, err)
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	allianceID, parseErr := strconv.ParseInt(allianceIDRaw, 10, 64)
	if parseErr != nil || allianceID <= 0 {
		return []*pb.AllianceBasicInfo{}, pb.ERROR_CODE_SUCCESS
	}
	alliance, code := LoadAllianceBasicInfoFromRedis(allianceID)
	if code != pb.ERROR_CODE_SUCCESS || alliance == nil {
		// Name index is best-effort and may be stale.
		return []*pb.AllianceBasicInfo{}, pb.ERROR_CODE_SUCCESS
	}
	if alliance.ServerId != serverID || alliance.Name != allianceName {
		// Name index is best-effort and may be stale.
		return []*pb.AllianceBasicInfo{}, pb.ERROR_CODE_SUCCESS
	}

	playerPower := player.GetMainFormationPower()
	playerCityLevel := int32(1)
	if player.ArchitectureModel != nil {
		playerCityLevel = player.ArchitectureModel.GetMainLevel()
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	playerAllianceID := allianceInfo.AllianceId
	info := buildServerAllianceBasicInfo(alliance, playerAllianceID, playerPower, playerCityLevel, player.GetUserId())
	if info == nil {
		return []*pb.AllianceBasicInfo{}, pb.ERROR_CODE_SUCCESS
	}
	return []*pb.AllianceBasicInfo{info}, pb.ERROR_CODE_SUCCESS
}

func buildServerAllianceBasicInfo(alliance *model.AllianceEntity, playerAllianceID int64, playerPower int64, playerCityLevel int32, playerId int64) *pb.AllianceBasicInfo {
	if alliance == nil || alliance.AllianceId <= 0 {
		return nil
	}
	cfg := gameConfig.GetAllianceLevelCfg(alliance.Level)
	if cfg == nil {
		return nil
	}
	applyType := enum.AllianceEnterType_Free
	if playerAllianceID > 0 || alliance.MemberNum >= cfg.Num {
		applyType = enum.AllianceEnterType_Condition_NOT_MATCH
	} else {
		if alliance.ApplyType == enum.AllianceEnterType_Apply {
			applyType = enum.AllianceEnterType_Apply
			applyList, errorCode := loadAllianceApplyListUserIDsFromRedis(alliance.AllianceId)
			if errorCode != pb.ERROR_CODE_SUCCESS {
				applyType = enum.AllianceEnterType_Condition_NOT_MATCH
			} else {
				if slices.Contains(applyList, playerId) {
					applyType = enum.AllianceEnterType_AlreadyApply
				} else {
					if playerPower < alliance.PowerApplyCondition || playerCityLevel < alliance.CityLevelCondition {
						applyType = enum.AllianceEnterType_Condition_NOT_MATCH
					}
				}
			}
		}
	}

	return &pb.AllianceBasicInfo{
		AllianceId:          alliance.AllianceId,
		Name:                alliance.Name,
		Announce:            alliance.Announce,
		Notice:              alliance.Notice,
		BadgeId:             alliance.BadgeId,
		Level:               alliance.Level,
		ApplyType:           applyType,
		PowerApplyCondition: alliance.PowerApplyCondition,
		CityLevel:           alliance.CityLevelCondition,
		CurrentPlayerNum:    alliance.MemberNum,
		MaxPlayerNum:        cfg.Num,
		AllianceTotalPower:  alliance.AllianceTotalPower,
		LeaderName:          alliance.LeaderName,
	}
}

func ApplyAllianceHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ApplyAllianceReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if req.AllianceId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if allianceInfo.AllianceId != 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_ALLIANCE_ALREADY_IN_ALLIANCE)
		return
	}

	playerPower := player.GetMainFormationPower()
	playerCityLevel := int32(1)
	if player.ArchitectureModel != nil {
		playerCityLevel = player.ArchitectureModel.GetMainLevel()
	}
	alliance, code := LoadAllianceBasicInfoFromRedis(req.AllianceId)
	if code != pb.ERROR_CODE_SUCCESS || alliance == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_ALLIANCE_ALREADY_IN_ALLIANCE)
		return
	}
	cfg := gameConfig.GetAllianceLevelCfg(alliance.Level)
	if cfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if alliance.ServerId != player.User.GetServerId() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if alliance.MemberNum >= cfg.Num {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_ALLIANCE_IS_FULL)
		return
	}
	if playerPower < alliance.PowerApplyCondition || playerCityLevel < alliance.CityLevelCondition {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_ALLIANCE_ENTER_CONDITION_IS_NOT_MATCH)
		return
	}
	if alliance.ApplyType == enum.AllianceEnterType_Apply {
		addPlayerToApplyRedis(alliance.AllianceId, player.GetUserId())
		clientResp := &pb.ApplyAllianceResp{}
		messageSender.SendMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, clientResp)
	}

	if err := sendApplyAllianceToSocial(player, req.AllianceId); err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
}

func ApplyAllianceOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	resp := &rpcPb.ApplyAllianceResp{
		ErrorCode: int32(pb.ERROR_CODE_SUCCESS),
	}
	req, ok := message.(*rpcPb.ApplyAllianceReq)
	if !ok {
		resp.ErrorCode = int32(pb.ERROR_CODE_PB_CONV_ERROR)
		socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPLY_ALLIANCE_RESP, resp)
		return
	}
	if alliance == nil {
		resp.ErrorCode = int32(pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPLY_ALLIANCE_RESP, resp)
		return
	}
	resp = alliance.ApplyAlliance(req)
	if resp == nil {
		return
	}
	socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPLY_ALLIANCE_RESP, resp)
}

func BackApplyAllianceFromSocialNode(message proto.Message, player *model.PlayerModel) {
	resp, ok := message.(*rpcPb.ApplyAllianceResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		return
	}

	code := pb.ERROR_CODE(resp.GetErrorCode())
	if code != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, code)
		return
	}
	clientResp := &pb.ApplyAllianceResp{}
	if resp.Alliance != nil {
		clientResp.AllianceId = resp.Alliance.AllianceId
		eventBusService.SubmitAllianceJoinEvent(player.GetUserId())
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_APPLY_ALLIANCE_RESP, clientResp)
}

func sendApplyAllianceToSocial(player *model.PlayerModel, allianceID int64) error {
	if player == nil || player.User == nil || allianceID <= 0 {
		return errors.New("invalid apply alliance args")
	}
	return rpcController.SendMessageToSocial(
		player.GetUserId(),
		allianceID,
		int32(pb.MESSAGE_ID_APPLY_ALLIANCE_RESP),
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPLY_ALLIANCE_REQ,
		&rpcPb.ApplyAllianceReq{
			UserId:     player.GetUserId(),
			ServerId:   player.User.GetServerId(),
			AllianceId: allianceID,
		},
	)
}

func addPlayerToApplyRedis(allianceId int64, playerId int64) {
	now := tool.UnixNowMilli()
	ctx := context.Background()
	key := enum.GetAllianceApplyListKey(allianceId)
	expireBefore := now - enum.AllianceApplyExpireDurationMillis
	if err := dbService.RDB.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(expireBefore, 10)).Err(); err != nil {
		logger.ErrorBySprintf("[allianceManager] cleanup alliance apply expired failed allianceId:%d err:%v", allianceId, err)
		return
	}

	_, err := dbService.RDB.ZAddArgs(ctx, key, redis.ZAddArgs{
		NX: true,
		Members: []redis.Z{{
			Score:  float64(now),
			Member: strconv.FormatInt(playerId, 10),
		}},
	}).Result()
	if err != nil {
		logger.ErrorBySprintf("[allianceManager] add alliance apply redis failed allianceId:%d userId:%d err:%v", allianceId, playerId, err)
		return
	}

	maxLength := int64(enum.ApplyListMaxLength)
	if maxLength <= 0 {
		maxLength = 50
	}
	count, err := dbService.RDB.ZCard(ctx, key).Result()
	if err != nil {
		logger.ErrorBySprintf("[allianceManager] get alliance apply list size failed allianceId:%d err:%v", allianceId, err)
		return
	}
	overflow := count - maxLength
	if overflow > 0 {
		if err = dbService.RDB.ZRemRangeByRank(ctx, key, 0, overflow-1).Err(); err != nil {
			logger.ErrorBySprintf("[allianceManager] trim alliance apply list failed allianceId:%d overflow:%d err:%v", allianceId, overflow, err)
			return
		}
	}
	return
}

func GetApplyListHandler(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.GetApplyListReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_APPLY_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_APPLY_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_APPLY_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if allianceInfo.AllianceId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_APPLY_LIST_RESP, pb.ERROR_CODE_ALLIANCE_NOT_IN_ALLIANCE)
		return
	}
	applyUserIDs, code := loadAllianceApplyListUserIDsFromRedis(allianceInfo.AllianceId)
	if code != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_APPLY_LIST_RESP, code)
		return
	}
	infos := make([]*pb.AllianceApplyPlayerInfo, 0, len(applyUserIDs))
	for _, userID := range applyUserIDs {
		if userID <= 0 {
			continue
		}
		info := &pb.AllianceApplyPlayerInfo{
			PlayerId: userID,
		}
		basic := logicCommon.GetPlayerRedisInfo(userID)
		if basic != nil {
			info.NickName = basic.BasicInfo.Name
			info.Head = basic.BasicInfo.HeadId
			info.CityLevel = basic.BasicInfo.MainCityLevel
			info.Power = basic.BattleInfo.GetMainFormationPower()
			info.HeadFrame = basic.BasicInfo.FrameId
		}
		infos = append(infos, info)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_APPLY_LIST_RESP, &pb.GetApplyListResp{Infos: infos})
}

func GetPlayerAllianceInfoHandler(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.GetPlayerAllianceInfoReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if allianceInfo.AllianceId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_RESP, pb.ERROR_CODE_ALLIANCE_NOT_IN_ALLIANCE)
		return
	}

	err := rpcController.SendMessageToSocial(
		player.GetUserId(),
		allianceInfo.AllianceId,
		int32(pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_RESP),
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_INFO_REQ,
		&rpcPb.GetAllianceInfoReq{
			UserId:       player.GetUserId(),
			AllianceId:   allianceInfo.AllianceId,
			IncludeApply: false,
		},
	)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
}

func GetAllianceInfoOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	req, ok := message.(*rpcPb.GetAllianceInfoReq)
	if !ok {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if !ensureAllianceMembershipOnSocialNode(session, alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_INFO_RESP) {
		return
	}
	resp, code := alliance.GetAllianceInfo(req)
	if code != pb.ERROR_CODE_SUCCESS {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_INFO_RESP, code)
		return
	}
	socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_INFO_RESP, resp)
}

func BackGetPlayerAllianceInfoFromSocialNode(message proto.Message, player *model.PlayerModel) {
	resp, ok := message.(*rpcPb.GetAllianceInfoResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if resp.GetAlliance() == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	player.User.Entity.AllianceName = resp.GetAlliance().GetName()

	members := make([]*pb.AllianceMemberInfo, 0, len(resp.GetMembers()))
	for _, member := range resp.GetMembers() {
		if member == nil || member.GetUserId() <= 0 {
			continue
		}
		info := &pb.AllianceMemberInfo{
			PlayerId:    member.GetUserId(),
			NickName:    member.NickName,
			Head:        member.Head,
			Power:       member.Power,
			CityLevel:   member.CityLevel,
			Contribute:  member.Contribute,
			Position:    pb.ALLIANCE_POSITION(member.GetRole()),
			OfflineTime: member.OfflineTime,
		}
		members = append(members, info)
	}

	wareHouseList := make([]*pb.AllianceWarehouseInfo, 0, len(resp.GetWarehouse()))
	for _, warehouse := range resp.GetWarehouse() {
		wareHouseList = append(wareHouseList, &pb.AllianceWarehouseInfo{
			ItemId:  warehouse.ItemId,
			ItemNum: warehouse.Count,
		})
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_PLAYER_ALLIANCE_INFO_RESP, &pb.GetPlayerAllianceInfoResp{
		BasicInfo: allianceBasicInfoToPb(resp.GetAlliance(), getAllianceLeaderName(resp.GetAlliance(), resp.GetMembers())),
		Members:   members,
		Warehouse: wareHouseList,
	})
}

func ApproveAllianceApplyHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ApproveAllianceApplyReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if allianceInfo.AllianceId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_RESP, pb.ERROR_CODE_ALLIANCE_NOT_IN_ALLIANCE)
		return
	}
	applyUserID := req.GetPlayerId()
	if applyUserID <= 0 || applyUserID == player.GetUserId() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	err := rpcController.SendMessageToSocial(
		player.GetUserId(),
		allianceInfo.AllianceId,
		int32(pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_RESP),
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPROVE_ALLIANCE_APPLY_BATCH_REQ,
		&rpcPb.ApproveAllianceApplyReq{
			OperatorUserId: player.GetUserId(),
			AllianceId:     allianceInfo.AllianceId,
			ApplyUserId:    applyUserID,
			ApproveApply:   req.GetApproveApply(),
		},
	)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
}

func ApproveAllianceApplyOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	req, ok := message.(*rpcPb.ApproveAllianceApplyReq)
	if !ok {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPROVE_ALLIANCE_APPLY_BATCH_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if !ensureAllianceMembershipOnSocialNode(session, alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPROVE_ALLIANCE_APPLY_BATCH_RESP) {
		return
	}
	resp, code := alliance.ApproveAllianceApply(req)
	if code != pb.ERROR_CODE_SUCCESS {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPROVE_ALLIANCE_APPLY_BATCH_RESP, code)
		return
	}
	socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_APPROVE_ALLIANCE_APPLY_BATCH_RESP, resp)
}

func BackApproveAllianceApplyFromSocialNode(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*rpcPb.ApproveAllianceApplyResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_APPROVE_ALLIANCE_APPLY_RESP, &pb.ApproveAllianceApplyResp{})
}

func KickMemberOutAllianceHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.KickMemberOutAllianceReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if allianceInfo.AllianceId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_RESP, pb.ERROR_CODE_ALLIANCE_NOT_IN_ALLIANCE)
		return
	}
	targetUserID := req.GetPlayerId()
	if targetUserID <= 0 || targetUserID == player.GetUserId() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	err := rpcController.SendMessageToSocial(
		player.GetUserId(),
		allianceInfo.AllianceId,
		int32(pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_RESP),
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_KICK_ALLIANCE_MEMBER_REQ,
		&rpcPb.KickAllianceMemberReq{
			OperatorUserId: player.GetUserId(),
			AllianceId:     allianceInfo.AllianceId,
			TargetUserId:   targetUserID,
		},
	)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
}

func KickAllianceMemberOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	req, ok := message.(*rpcPb.KickAllianceMemberReq)
	if !ok {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_KICK_ALLIANCE_MEMBER_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if !ensureAllianceMembershipOnSocialNode(session, alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_KICK_ALLIANCE_MEMBER_RESP) {
		return
	}
	code := alliance.KickAllianceMember(req)
	if code != pb.ERROR_CODE_SUCCESS {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_KICK_ALLIANCE_MEMBER_RESP, code)
		return
	}
	session.SendPushToUser(req.TargetUserId, int32(rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_PUSH_ALLIANCE_CHANGED), &rpcPb.PushAllianceChanged{
		UserId:       req.TargetUserId,
		AllianceId:   0,
		AllianceName: "",
	})
	socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_KICK_ALLIANCE_MEMBER_RESP, &rpcPb.KickAllianceMemberResp{})
}

func BackKickMemberOutAllianceFromSocialNode(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*rpcPb.KickAllianceMemberResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_KICK_MEMBER_OUT_ALLIANCE_RESP, &pb.KickMemberOutAllianceResp{})
}

func ChangeMemberPositionHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ChangeMemberPositionReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if allianceInfo.AllianceId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP, pb.ERROR_CODE_ALLIANCE_NOT_IN_ALLIANCE)
		return
	}

	targetUserID := req.GetPlayerId()
	if targetUserID <= 0 || targetUserID == player.GetUserId() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if !IsValidPosition(req.GetPosition()) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	err := rpcController.SendMessageToSocial(
		player.GetUserId(),
		allianceInfo.AllianceId,
		int32(pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP),
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_MEMBER_POSITION_REQ,
		&rpcPb.ChangeMemberPositionReq{
			OperatorUserId: player.GetUserId(),
			AllianceId:     allianceInfo.AllianceId,
			TargetUserId:   targetUserID,
			TargetRole:     int32(req.GetPosition()),
		},
	)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
}

func IsValidPosition(position pb.ALLIANCE_POSITION) bool {
	return position != pb.ALLIANCE_POSITION_ALLIANCE_NONE
}

func ChangeMemberPositionOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	req, ok := message.(*rpcPb.ChangeMemberPositionReq)
	if !ok {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_MEMBER_POSITION_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if !ensureAllianceMembershipOnSocialNode(session, alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_MEMBER_POSITION_RESP) {
		return
	}
	code := alliance.ChangeMemberPosition(req)
	if code != pb.ERROR_CODE_SUCCESS {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_MEMBER_POSITION_RESP, code)
		return
	}
	socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHANGE_MEMBER_POSITION_RESP, &rpcPb.ChangeMemberPositionResp{})
}

func BackChangeMemberPositionFromSocialNode(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*rpcPb.ChangeMemberPositionResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_CHANGE_MEMBER_POSITION_RESP, &pb.ChangeMemberPositionResp{})
}

// TODO:退出联盟扣除联盟贡献
func LeaveAllianceHandler(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.LeaveAllianceReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LEAVE_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LEAVE_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LEAVE_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if allianceInfo.AllianceId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LEAVE_ALLIANCE_RESP, pb.ERROR_CODE_ALLIANCE_NOT_IN_ALLIANCE)
		return
	}
	err := rpcController.SendMessageToSocial(
		player.GetUserId(),
		allianceInfo.AllianceId,
		int32(pb.MESSAGE_ID_LEAVE_ALLIANCE_RESP),
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_QUIT_ALLIANCE_REQ,
		&rpcPb.QuitAllianceReq{
			UserId:     player.GetUserId(),
			AllianceId: allianceInfo.AllianceId,
		},
	)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LEAVE_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
}

func AllianceSignInHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.AllianceSignInReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_SIGN_IN_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_SIGN_IN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if req.SignType == 1 {
		cost := gameConfig.GetAllianceCheckInDiamondCost()
		signEntity := player.SignInModel.Entity
		if signEntity.BasicCount >= int32(len(cost)) {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_SIGN_IN_RESP, pb.ERROR_CODE_ALLIANCE_SIGN_IN_COUNT_OVER)
			return
		}
		err := itemService.RemoveItem(player, &gameConfig.ItemConfig{ID: enum.DIAMOND_ITEM_ID, Num: int64(cost[signEntity.BasicCount])}, enum.ITEM_CHANGE_REASON_ALLIANCE_SIGN_IN)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_SIGN_IN_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
			return
		}
		player.SignInModel.UpdateBasicCount(signEntity.BasicCount + 1)
	} else if req.SignType == 2 {
		signEntity := player.SignInModel.Entity
		if player.SignInModel.Entity.AdvertisementCount > 0 {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_SIGN_IN_RESP, pb.ERROR_CODE_ALLIANCE_SIGN_IN_COUNT_OVER) //次数超了
			return
		}
		player.SignInModel.UpdateAdvertisementCount(signEntity.AdvertisementCount + 1)
	}
	if req.SignType != 0 {
		items := gameConfig.Drop(gameConfig.GetAllianceCheckInRewards())
		if items == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_SIGN_IN_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_ALLIANCE_SIGN_IN)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_SIGN_IN_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
			return
		}
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_ALLIANCE_SIGN_IN_RESP, &pb.AllianceSignInResp{
		BasicCount:         player.SignInModel.Entity.BasicCount,
		AdvertisementCount: player.SignInModel.Entity.AdvertisementCount,
	})
}

func LeaveAllianceOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	req, ok := message.(*rpcPb.QuitAllianceReq)
	if !ok {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_QUIT_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if !ensureAllianceMembershipOnSocialNode(session, alliance, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_QUIT_ALLIANCE_RESP) {
		return
	}
	code := alliance.QuitAlliance(req)
	if code != pb.ERROR_CODE_SUCCESS {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_QUIT_ALLIANCE_RESP, code)
		return
	}
	socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_QUIT_ALLIANCE_RESP, &rpcPb.QuitAllianceResp{})
}

func BackLeaveAllianceFromSocialNode(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*rpcPb.QuitAllianceResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LEAVE_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_LEAVE_ALLIANCE_RESP, &pb.LeaveAllianceResp{})
}

func LoadAllianceBasicInfoFromRedis(allianceID int64) (*model.AllianceEntity, pb.ERROR_CODE) {
	if allianceID <= 0 || dbService.RDB == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	key := enum.GetAllianceBasicInfoKey(allianceID)
	raw, err := dbService.RDB.Get(context.Background(), key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
		}
		logger.ErrorBySprintf("[alliance] get alliance basic info from redis failed allianceId:%d err:%v", allianceID, err)
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	alliance := &model.AllianceEntity{}
	if err = json.Unmarshal([]byte(raw), alliance); err != nil {
		logger.ErrorBySprintf("[alliance] unmarshal alliance basic info from redis failed allianceId:%d err:%v", allianceID, err)
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	return alliance, pb.ERROR_CODE_SUCCESS
}

func loadAllianceApplyListUserIDsFromRedis(allianceID int64) ([]int64, pb.ERROR_CODE) {
	if allianceID <= 0 || dbService.RDB == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	ctx := context.Background()
	key := enum.GetAllianceApplyListKey(allianceID)
	expireBefore := tool.UnixNowMilli() - enum.AllianceApplyExpireDurationMillis
	if err := dbService.RDB.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(expireBefore, 10)).Err(); err != nil {
		logger.ErrorBySprintf("[alliance] cleanup expired alliance apply list failed allianceId:%d err:%v", allianceID, err)
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}

	maxLength := int64(enum.ApplyListMaxLength)
	if maxLength <= 0 {
		maxLength = 50
	}
	values, err := dbService.RDB.ZRevRange(ctx, key, 0, maxLength-1).Result()
	if err != nil {
		logger.ErrorBySprintf("[alliance] get alliance apply list from redis failed allianceId:%d err:%v", allianceID, err)
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	if len(values) == 0 {
		return []int64{}, pb.ERROR_CODE_SUCCESS
	}

	userIDs := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, raw := range values {
		userID, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || userID <= 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	return userIDs, pb.ERROR_CODE_SUCCESS
}

func CreateAllianceHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.CreateAllianceReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.User == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())
	if allianceInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if allianceInfo.AllianceId != 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_ALLIANCE_ALREADY_IN_ALLIANCE)
		return
	}

	name := strings.TrimSpace(req.GetName())
	announce := strings.TrimSpace(req.GetAnnounce())
	errorCode := CheckAllianceNameLegal(name)
	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, errorCode)
		return
	}
	if announce != "" {
		errorCode = CheckAllianceAnnounceLegal(announce)
		if errorCode != pb.ERROR_CODE_SUCCESS {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, errorCode)
			return
		}
	}
	unlocks := gameConfig.GetCreateAllianceUnlock()
	for _, unlock := range unlocks {
		if !unlockService.CheckUnlock(unlock, player) {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_ALLIANCE_CREATE_CONDITION_IS_NOT_MATCH)
			return
		}
	}
	result, err := itemService.CheckItemCount(player, gameConfig.GetCreateAllianceItem())
	if err != nil || !result {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err = itemService.RemoveItem(player, gameConfig.GetCreateAllianceItem(), enum.ITEM_CHANGE_REASON_CREATE_ALLIANCE)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}

	err = rpcController.SendMessageToSocial(
		player.GetUserId(),
		0,
		int32(pb.MESSAGE_ID_CREATE_ALLIANCE_RESP),
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CREATE_ALLIANCE_REQ,
		&rpcPb.CreateAllianceReq{
			UserId:   player.GetUserId(),
			ServerId: player.User.GetServerId(),
			Name:     name,
			Announce: announce,
			BadgeId:  req.GetBadgeId(),
		},
	)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
}

func CheckAllianceNameLegal(name string) pb.ERROR_CODE {
	if name == "" || wordFilter.HasSensitive(name) || !tool.IsUnicodeLetterOrDigit(name) {
		return pb.ERROR_CODE_ALLIANCE_NAME_IS_ILLEGAL
	}
	if int32(utf8.RuneCountInString(name)) > gameConfig.GetAllianceNameMaxLength() {
		return pb.ERROR_CODE_ALLIANCE_NAME_TOO_LONG
	}
	return pb.ERROR_CODE_SUCCESS
}

func CheckAllianceAnnounceLegal(announce string) pb.ERROR_CODE {
	if announce == "" || wordFilter.HasSensitive(announce) || tool.HasSQLRisk(announce) {
		return pb.ERROR_CODE_ALLIANCE_NAME_IS_ILLEGAL
	}
	if int32(utf8.RuneCountInString(announce)) > gameConfig.GetAllianceAnnounceMaxLength() {
		return pb.ERROR_CODE_ALLIANCE_ANNOUNCE_TOO_LONG
	}
	return pb.ERROR_CODE_SUCCESS
}

func CheckAllianceNoticeLegal(notice string) pb.ERROR_CODE {
	if notice == "" || wordFilter.HasSensitive(notice) || tool.HasSQLRisk(notice) {
		return pb.ERROR_CODE_ALLIANCE_NOTICE_IS_ILLEGAL
	}
	if int32(utf8.RuneCountInString(notice)) > gameConfig.GetAllianceNoticeMaxLength() {
		return pb.ERROR_CODE_ALLIANCE_NOTICE_TOO_LONG
	}
	return pb.ERROR_CODE_SUCCESS
}

func CreateAllianceOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	req, ok := message.(*rpcPb.CreateAllianceReq)
	if !ok {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	resp := socialService.GetService().CreateAlliance(req)
	socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CREATE_ALLIANCE_RESP, resp)
}

func BackCreateAllianceFromSocialNode(message proto.Message, player *model.PlayerModel) {
	resp, ok := message.(*rpcPb.CreateAllianceResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if pb.ERROR_CODE(resp.ErrorCode) != pb.ERROR_CODE_SUCCESS {
		_ = itemService.AddItem(player, gameConfig.GetCreateAllianceItem(), enum.ITEM_CHANGE_REASON_CREATE_ALLIANCE_FILED)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE(resp.ErrorCode))
		return
	}
	alliance := resp.GetAlliance()
	if alliance == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_CREATE_ALLIANCE_RESP, &pb.CreateAllianceResp{
		BasicInfo: allianceBasicInfoToPb(alliance, getAllianceLeaderName(alliance, nil)),
	})
	eventBusService.SubmitAllianceJoinEvent(player.GetUserId())
}

func getAllianceLeaderName(alliance *rpcPb.AllianceInfo, members []*rpcPb.AllianceMember) string {
	if alliance == nil || alliance.GetLeaderUserId() <= 0 {
		return ""
	}
	leaderUserID := alliance.GetLeaderUserId()
	for _, member := range members {
		if member == nil {
			continue
		}
		if member.GetUserId() == leaderUserID && member.GetNickName() != "" {
			return member.GetNickName()
		}
	}
	leaderBasicInfo := logicCommon.GetPlayerBasicInfoFromRedis(leaderUserID)
	if leaderBasicInfo == nil {
		return ""
	}
	return leaderBasicInfo.Name
}

func allianceBasicInfoToPb(alliance *rpcPb.AllianceInfo, leaderName string) *pb.AllianceBasicInfo {
	if alliance == nil {
		return nil
	}
	return &pb.AllianceBasicInfo{
		AllianceId:          alliance.GetAllianceId(),
		Name:                alliance.GetName(),
		Announce:            alliance.GetAnnounce(),
		Notice:              alliance.GetNotice(),
		BadgeId:             alliance.GetBadgeId(),
		Level:               alliance.GetLevel(),
		ApplyType:           alliance.GetApplyType(),
		PowerApplyCondition: alliance.GetPowerApplyCondition(),
		CityLevel:           alliance.GetCityLevel(),
		CurrentPlayerNum:    alliance.GetMemberCount(),
		MaxPlayerNum:        alliance.GetMaxMember(),
		AllianceTotalPower:  alliance.GetAllianceTotalPower(),
		LeaderName:          leaderName,
	}
}

func BackAllianceChangedFromSocialNode(message proto.Message, player *model.PlayerModel) {
	//req, ok := message.(*rpcPb.PushAllianceChanged)
	//if !ok {
	//	return
	//}
	//if player == nil || player.User == nil {
	//	return
	//}
}

func AllianceItemsOperationOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	req, ok := message.(*rpcPb.AllianceItemsOperationReq)
	if !ok {
		socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_ALLIANCE_ITEMS_OPERATION_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	switch req.GetOperation() {
	case int32(enum.AddProp):
		err := alliance.AddItems(req.GetItems())
		if err != nil {
			socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_ALLIANCE_ITEMS_OPERATION_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
	case int32(enum.RemoveProp):
		err := alliance.RemoveItems(req.GetItems())
		if err != nil {
			socialPlatform.SendErrorMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_ALLIANCE_ITEMS_OPERATION_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		for _, v := range req.GetItems() {
			v.Count = -v.Count
		}
	}
	for _, v := range alliance.GetAllianceMember() {
		rpcController.NotifyAllianceOperationToGateway(v, 0, pb.ALLIANCE_CHANGE_OPER_ITEM_CHANGE, req.GetItems())
	}
}

func GetAllianceActivityItemNumOnSocialNode(message proto.Message, session *logicSessionManager.AllianceSession, alliance *socialService.AllianceModel) {
	_, ok := message.(*rpcPb.GetAllianceActivityItemNumReq)
	resp := &rpcPb.GetAllianceActivityItemNumResp{
		ItemNum:   0,
		ErrorCode: int32(pb.ERROR_CODE_SUCCESS),
	}
	if !ok {
		resp.ErrorCode = int32(pb.ERROR_CODE_PB_CONV_ERROR)
		socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_ACTIVITY_ITEM_NUM_RESP, resp)
		return
	}

	itemNum, err := alliance.GetItemCount(&rpcPb.ItemInfo{ItemId: enum.ALLIANCE_TASK_ACTIVE_VALUE_ITEM_ID, Count: 0})
	if err != nil {
		// 道具不存在
		resp.ErrorCode = int32(pb.ERROR_CODE_ITEM_NOT_EXIST)
		socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_ACTIVITY_ITEM_NUM_RESP, resp)
		return
	}
	socialPlatform.SendMessageBySession(session.GetSession(), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_ALLIANCE_ACTIVITY_ITEM_NUM_RESP, &rpcPb.GetAllianceActivityItemNumResp{
		ItemNum: itemNum,
	})
}

func BackGetAllianceActivityItemNumFromSocialNode(message proto.Message, player *model.PlayerModel) {
	resp, ok := message.(*rpcPb.GetAllianceActivityItemNumResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_DAILY_TASK_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if pb.ERROR_CODE(resp.ErrorCode) != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_DAILY_TASK_REWARD_RESP, pb.ERROR_CODE(resp.ErrorCode))
		return
	}
	itemNum := resp.GetItemNum()
	addItems, err := player.AllianceDailyActivityModel.RewardActivity(itemNum)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_DAILY_TASK_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	err = itemService.AddItems(player, addItems, enum.ITEM_CHANGE_REASON_ALLIANCE_DAILY_TASK)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ALLIANCE_DAILY_TASK_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_ALLIANCE_DAILY_TASK_REWARD_RESP, &pb.AllianceDailyTaskRewardResp{
		AllianceDailyDetail: &pb.AllianceDailyDetail{
			DailyBox: player.AllianceDailyActivityModel.Entity.DailyBox,
		},
	})
}
