package gameController

import (
	"fmt"
	"strconv"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gloryArenaService"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/rankBoardPlatform"
	"github.com/drop/GoServer/server/logic/rankboardService"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("rankBoard", &RankBoardController{})
}

type RankBoardController struct {
}

var _ LogicControllerInterface = (*RankBoardController)(nil)

func (p *RankBoardController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_RANK_LIST_REQ, &pb.RankListReq{}, GetRankList, enum.FUNCTION_ID_RANKBOARD)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_RANK_INFO_RESP, &rpcPb.GetRankInfoResp{}, BackRankListFromRankBoardNode)

	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_RANK_LIKE_REQ, &pb.RankLikeReq{}, OnRankLike, enum.FUNCTION_ID_RANKBOARD)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_REQ, &pb.RankReceiveBoxRewardReq{}, OnReceiveRankBoxReward, enum.FUNCTION_ID_RANKBOARD)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHECK_RANK_FULL_RESP, &rpcPb.CheckRankIsFullResp{}, BackRankIsFullFromRankBoardNode)
}

func RegisterRankBoardMessage() {
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANK_BOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_RANK_INFO_REQ, &rpcPb.GetRankInfoReq{}, GetRankListFromRankBoardNode)
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANK_BOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_THUMB_UP_RANK_INFO, &rpcPb.ThumbUpRankInfo{}, OnThumbUpRankInfoOnRankBoardNode)
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANK_BOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHECK_RANK_FULL_REQ, &rpcPb.CheckRankIsFullReq{}, CheckRankIsFullOnRankBoardNode)
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANK_BOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_MY_RANK_REQ, &rpcPb.GetMyRankReq{}, GetMyRankFromRankBoardNode)
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANK_BOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, &rpcPb.NotifyUpdateRankInfo{}, OnUpdateRankInfoOnRankBoardNode)
}
func GetRankList(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.RankListReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if req.CommonRankId == 0 && req.ActId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if req.ActId != 0 && len(req.ActTargetId) == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if req.CommonRankId != 0 && len(req.ActTargetId) > 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	version := ""
	if req.ActId != 0 {
		if ok, version = player.PlayerActivityModel.CheckActivityOpen(req.ActId); !ok {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
			return
		}
	} else if req.CommonRankId != 0 {
		rankCfg := gameConfig.GetRankCfg(req.CommonRankId)
		if rankCfg != nil {
			if rankCfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_ARENA) || rankCfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA) {
				if player.PlayerArenaModel != nil && player.PlayerArenaModel.GetVersion() != "" {
					version = player.PlayerArenaModel.GetVersion()
				} else {
					version = logicCommon.GetArenaRankVersionByTime(player.User.GetServerId(), tool.UnixNowMilli())
				}
			} else if rankCfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_ROUND_WIN_COUNT) || rankCfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT) {
				if player.PlayerGloryArenaModel != nil {
					version = player.PlayerGloryArenaModel.GetPoolVersion()
				}
			} else if rankCfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_SEASON_WIN_COUNT) {
				if player.PlayerGloryArenaModel != nil {
					version = player.PlayerGloryArenaModel.GetSeasonVersion()
				}
			}
		}
	}

	actTargetIds := req.ActTargetId
	if req.CommonRankId != 0 {
		actTargetIds = []int32{0}
	}
	rankIds := make([]string, 0, len(actTargetIds))
	seenRankIds := make(map[string]struct{}, len(actTargetIds))
	for _, actTargetId := range actTargetIds {
		rankId, err := logicCommon.GetRankUniqueId(req.CommonRankId, req.ActId, actTargetId, player.User.GetServerId(), version)
		if err != nil {
			logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		if _, ok = seenRankIds[rankId]; ok {
			continue
		}
		seenRankIds[rankId] = struct{}{}
		rankIds = append(rankIds, rankId)
	}
	if len(rankIds) == 0 {
		messageSender.SendMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, &pb.RankListResp{})
		return
	}
	err := rpcController.SendMessageToRankBoardBatch(player.GetUserId(), rankIds, int32(pb.MESSAGE_ID_RANK_LIST_RESP), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_RANK_INFO_REQ, &rpcPb.GetRankInfoReq{})
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_RANK_BOARD_NOT_FOUND)
		return
	}
}

func GetRankListFromRankBoardNode(message proto.Message, rankId string, session serviceInterface.SessionInterface) {
	rankSession, ok := session.(*logicSessionManager.RankBoardSession)
	if !ok {
		rankBoardPlatform.SendErrorMessageBySession(session, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_RANK_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	rankIds := rankSession.GetRankBoardInfoIds
	if len(rankIds) == 0 {
		rankIds = []string{rankId}
	}
	resp := &rpcPb.GetRankInfoResp{
		RankPageInfos: make([]*rpcPb.RankPageInfo, 0, len(rankIds)),
	}
	for _, currentRankId := range rankIds {
		page := buildRpcRankPageInfo(currentRankId, rankSession.UserId)
		resp.RankPageInfos = append(resp.RankPageInfos, page)
	}
	if len(resp.RankPageInfos) == 1 {
		page := resp.RankPageInfos[0]
		resp.RankInfos = page.RankInfos
		resp.CommonId = page.CommonId
		resp.ActivityId = page.ActivityId
		resp.ActRankId = page.ActRankId
		resp.MyRank = page.MyRank
		resp.RankId = page.RankId
	}
	rankBoardPlatform.SendMessageBySession(session, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_RANK_INFO_RESP, resp)
}

func buildRpcRankPageInfo(rankId string, userId int64) *rpcPb.RankPageInfo {
	commonId, activityId, actRankId, _ := logicCommon.GetRankRealIdFromUniqueId(rankId)
	maxNum := int32(100)
	rankPointType := int32(enum.RANK_BOARD_SCORE_TYPE_LEVEL)
	cfg := gameConfig.GetRankCfgByIds(activityId, actRankId)
	if commonId != 0 {
		cfg = gameConfig.GetRankCfgByIds(0, commonId)
	}
	if cfg != nil {
		maxNum = cfg.PN
		rankPointType = cfg.PointType
	}

	queryUserId := userId
	if rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA) ||
		rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT) {
		allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(userId)
		if allianceInfo != nil && allianceInfo.AllianceId > 0 {
			queryUserId = allianceInfo.AllianceId
		} else {
			queryUserId = 0
		}
	}

	rankDetailInfo, playerRank, _ := rankboardService.GetRankInfo(rankId, int(maxNum), queryUserId)
	page := &rpcPb.RankPageInfo{
		CommonId:   commonId,
		ActivityId: activityId,
		ActRankId:  actRankId,
		MyRank:     playerRank,
		RankId:     rankId,
	}
	detail := make([]*rpcPb.RankInfo, len(rankDetailInfo))
	for i, info := range rankDetailInfo {
		detail[i] = &rpcPb.RankInfo{
			Id:            info.Id,
			Rank:          info.Rank,
			ThumbUpCount:  info.ThumbUpCount,
			Score:         info.Score,
			EnterRankTime: info.EnterTime,
		}
	}
	page.RankInfos = detail
	return page
}

func BackRankListFromRankBoardNode(message proto.Message, player *model.PlayerModel) {
	rpcMessage, ok := message.(*rpcPb.GetRankInfoResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if len(rpcMessage.RankPageInfos) > 0 {
		rankInfos := make([]*pb.RankInfo, 0, len(rpcMessage.RankPageInfos))
		for _, page := range rpcMessage.RankPageInfos {
			rankInfos = append(rankInfos, buildRankInfoFromRankPageInfo(page, player))
		}
		messageSender.SendMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, &pb.RankListResp{
			RankInfo: rankInfos,
		})
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, &pb.RankListResp{
		RankInfo: []*pb.RankInfo{buildRankInfoFromRankBoardResp(rpcMessage, player)},
	})
}

func buildRankInfoFromRankPageInfo(page *rpcPb.RankPageInfo, player *model.PlayerModel) *pb.RankInfo {
	return buildRankInfoFromRankBoardResp(&rpcPb.GetRankInfoResp{
		RankInfos:  page.RankInfos,
		CommonId:   page.CommonId,
		ActivityId: page.ActivityId,
		ActRankId:  page.ActRankId,
		MyRank:     page.MyRank,
		RankId:     page.RankId,
	}, player)
}

func buildRankInfoFromRankBoardResp(rpcMessage *rpcPb.GetRankInfoResp, player *model.PlayerModel) *pb.RankInfo {
	_, activityId, _, _ := logicCommon.GetRankRealIdFromUniqueId(rpcMessage.RankId)
	isLiked := false
	log := &model.PlayerRankThumbUpLog{
		RankId:       strconv.FormatInt(int64(activityId), 10),
		FromPlayerId: player.User.GetUserId(),
		ThumbUpTime:  tool.GetTodayZeroByTimeStamp(tool.UnixNowMilli()),
	}
	_, err := easyDB.GetPlayerEntityByWhere[model.PlayerRankThumbUpLog](map[string]interface{}{"rank_id": log.RankId, "from_player_id": log.FromPlayerId, "thumb_up_time": log.ThumbUpTime})
	if err == nil {
		isLiked = true
	}

	isClaim := false
	chestLog := &model.PlayerRankClaimChestsLog{
		RankId:       rpcMessage.RankId,
		FromPlayerId: player.User.GetUserId(),
	}
	_, err = easyDB.GetPlayerEntityByWhere[model.PlayerRankClaimChestsLog](map[string]interface{}{"rank_id": chestLog.RankId, "from_player_id": chestLog.FromPlayerId})
	if err == nil {
		isClaim = true
	}

	rankPointType := int32(enum.RANK_BOARD_SCORE_TYPE_LEVEL)
	rankMaxNum := int32(100)
	cfg := gameConfig.GetRankCfgByIds(activityId, rpcMessage.ActRankId)
	if rpcMessage.CommonId != 0 {
		cfg = gameConfig.GetRankCfgByIds(0, rpcMessage.CommonId)
	}
	if cfg != nil {
		rankPointType = cfg.PointType
		rankMaxNum = cfg.PN
	}

	rankLen := int32(len(rpcMessage.RankInfos))
	if rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ARENA) && rankLen < rankMaxNum {
		rankLen = rankMaxNum
	}
	isAllianceRank := rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA) ||
		rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_TOTAL_POWER) ||
		rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT)
	if isAllianceRank {
		allianceDetails := make([]*pb.RankAllianceDetailInfo, len(rpcMessage.RankInfos))
		for index, info := range rpcMessage.RankInfos {
			allianceDetails[index] = &pb.RankAllianceDetailInfo{
				AllianceId: info.Id,
				Rank:       info.Rank,
				LikeNum:    info.ThumbUpCount,
				RankScore:  info.Score,
				RankedOn:   info.EnterRankTime,
			}
			alliance, code := LoadAllianceBasicInfoFromRedis(info.Id)
			if code != pb.ERROR_CODE_SUCCESS || alliance == nil {
				continue
			}
			allianceDetails[index].AllianceName = alliance.Name
			allianceDetails[index].Icon = alliance.BadgeId
			allianceDetails[index].BattlePower = alliance.AllianceTotalPower
		}
		return &pb.RankInfo{
			CommonRankId:    rpcMessage.CommonId,
			ActId:           rpcMessage.ActivityId,
			ActTargetId:     rpcMessage.ActRankId,
			ChestClaimed:    isClaim,
			IsLiked:         isLiked,
			MyRank:          rpcMessage.MyRank,
			AllianceDetails: allianceDetails,
		}
	}
	detail := make([]*pb.RankDetailInfo, rankLen)

	index := int32(0)
	originalScore := int64(0)
	if rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ARENA) {
		originalScore = int64(gameConfig.GetArenaInitScore())
	}
	for ; index < int32(len(rpcMessage.RankInfos)); index++ {
		info := rpcMessage.RankInfos[index]
		detail[index] = &pb.RankDetailInfo{
			PlayerId:  info.Id,
			Rank:      info.Rank,
			LikeNum:   info.ThumbUpCount,
			RankScore: info.Score + originalScore,
			RankedOn:  info.EnterRankTime,
		}
		if info.Id != player.User.GetUserId() {
			playerRedisInfo := logicCommon.GetPlayerRedisInfo(info.Id)
			if playerRedisInfo == nil {
				continue
			}
			detail[index].HeroId = playerRedisInfo.BasicInfo.ShowHeroId
			detail[index].ClassId = playerRedisInfo.BasicInfo.ShowClassId
			detail[index].NickName = playerRedisInfo.BasicInfo.Name
			detail[index].Head = playerRedisInfo.BasicInfo.HeadId
			detail[index].FrameId = playerRedisInfo.BasicInfo.FrameId
			detail[index].BattlePower = make(map[int32]int64)
			detail[index].Title = playerRedisInfo.BasicInfo.Title
			detail[index].BubbleId = playerRedisInfo.BasicInfo.BubbleId
			detail[index].ImageId = playerRedisInfo.BasicInfo.ImageId
			for id, formation := range playerRedisInfo.BattleInfo.FormationInfo {
				detail[index].BattlePower[id] = formation.BattlePower
			}
		} else {
			detail[index].HeroId = 0
			detail[index].ClassId = 0
			detail[index].NickName = player.User.GetNickname()
			detail[index].Head = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeHead)
			detail[index].FrameId = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeHeadFrame)
			detail[index].Title = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeTitle)
			detail[index].BubbleId = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeBubble)
			detail[index].ImageId = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeImage)
			detail[index].BattlePower = make(map[int32]int64)
			for id, formation := range player.PlayerCacheInfo.BattleInfo.FormationInfo {
				detail[index].BattlePower[id] = formation.BattlePower
			}
		}
	}

	rankInfo := &pb.RankInfo{
		CommonRankId: rpcMessage.CommonId,
		ActId:        rpcMessage.ActivityId,
		ActTargetId:  rpcMessage.ActRankId,
		ChestClaimed: isClaim,
		IsLiked:      isLiked,
		MyRank:       rpcMessage.MyRank,
		Details:      detail,
	}
	return rankInfo
}

func OnRankLike(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.RankLikeReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if req.ActId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	actRankCfg := gameConfig.GetRankCfgByIds(req.ActId, req.ActTargetId)
	if actRankCfg == nil || actRankCfg.ActId != req.ActId {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	fullActRankCfg := gameConfig.GetRankActCfg(req.ActTargetId)
	if fullActRankCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	version := ""
	if ok, version = player.PlayerActivityModel.CheckActivityOpen(req.ActId); !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
		return
	}
	rankId, err := logicCommon.GetRankUniqueId(req.CommonRankId, req.ActId, req.ActTargetId, player.User.GetServerId(), version)
	if err != nil {
		logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	log := &model.PlayerRankThumbUpLog{
		RankId:       strconv.FormatInt(int64(req.ActId), 10),
		FromPlayerId: player.User.GetUserId(),
		ThumbUpTime:  tool.GetTodayZeroByTimeStamp(tool.UnixNowMilli()),
	}
	err = easyDB.CreatePlayerEntity[model.PlayerRankThumbUpLog](log)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_RANK_BOARD_IS_LIKED)
		return
	}
	drops := gameConfig.Drop(fullActRankCfg.LikeDropId)
	if drops != nil {
		_ = itemService.AddItems(player, drops, enum.ITEM_CHANGE_REASON_RANK_LIKE)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, &pb.RankLikeResp{
		Account: req.Account,
	})
	rpcReq := &rpcPb.ThumbUpRankInfo{
		Id:      req.Account,
		ThumbUp: 1,
	}
	err = rpcController.SendMessageToRankBoard(player.GetUserId(), rankId, int32(pb.MESSAGE_ID_RANK_LIKE_RESP), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_THUMB_UP_RANK_INFO, rpcReq)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_RANK_BOARD_NOT_FOUND)
		return
	}
}

func OnThumbUpRankInfoOnRankBoardNode(message proto.Message, rankId string, session serviceInterface.SessionInterface) {
	req, ok := message.(*rpcPb.ThumbUpRankInfo)
	if !ok {
		return
	}
	rankboardService.UpdateRankInfoThumbUp(rankId, req.Id, req.ThumbUp)
}

func OnReceiveRankBoxReward(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.RankReceiveBoxRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if req.ActId == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	version := ""
	if req.ActId != 0 {
		if ok, version = player.PlayerActivityModel.CheckActivityOpen(req.ActId); !ok {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
			return
		}
	}
	actRankCfg := gameConfig.GetRankCfgByIds(req.ActId, req.ActTargetId)
	if actRankCfg == nil || actRankCfg.ActId != req.ActId {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	rankId, err := logicCommon.GetRankUniqueId(req.CommonRankId, req.ActId, req.ActTargetId, player.User.GetServerId(), version)
	if err != nil {
		logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	log := &model.PlayerRankClaimChestsLog{
		RankId:       rankId,
		FromPlayerId: player.User.GetUserId(),
		ClaimTime:    tool.UnixNowMilli(),
	}
	err = easyDB.CreatePlayerEntity[model.PlayerRankClaimChestsLog](log)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, pb.ERROR_CODE_RANK_BOARD_CHEST_IS_CLAIMED)
		return
	}
	err = rpcController.SendMessageToRankBoard(player.GetUserId(), rankId, int32(pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHECK_RANK_FULL_REQ, &rpcPb.CheckRankIsFullReq{UserId: player.GetUserId()})
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, pb.ERROR_CODE_RANK_BOARD_NOT_FOUND)
		return
	}
}

func CheckRankIsFullOnRankBoardNode(message proto.Message, rankId string, session serviceInterface.SessionInterface) {
	req, ok := message.(*rpcPb.CheckRankIsFullReq)
	if !ok {
		return
	}
	rankBoardSession := session.(*logicSessionManager.RankBoardSession)
	if rankBoardSession == nil {
		return
	}
	commonId, actId, actRankId, _ := logicCommon.GetRankRealIdFromUniqueId(rankId)
	maxNum := int32(100)
	cfg := gameConfig.GetRankCfgByIds(actId, actRankId)
	if commonId != 0 {
		cfg = gameConfig.GetRankCfgByIds(0, commonId)
	}
	if cfg != nil {
		maxNum = cfg.PN
	}

	rankDetailInfo, _, err := rankboardService.GetRankInfo(rankId, int(maxNum), 0)
	if err != nil {
		rankBoardSession.ErrorCode = int32(pb.ERROR_CODE_RANK_BOARD_NOT_FOUND)
		rankBoardPlatform.SendErrorMessageBySession(session, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHECK_RANK_FULL_RESP, pb.ERROR_CODE_RANK_BOARD_NOT_FOUND)
		return
	}
	if len(rankDetailInfo) < int(maxNum) {
		rankBoardSession.ErrorCode = int32(pb.ERROR_CODE_RANK_BOARD_CHEST_CONDITION_NOT_MET)
		rankBoardPlatform.SendErrorMessageBySession(session, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHECK_RANK_FULL_RESP, pb.ERROR_CODE_RANK_BOARD_CHEST_CONDITION_NOT_MET)
		return
	}
	resp := &rpcPb.CheckRankIsFullResp{
		UserId: req.UserId,
		RankId: rankId,
	}
	rankBoardPlatform.SendMessageBySession(session, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHECK_RANK_FULL_RESP, resp)
}

func BackRankIsFullFromRankBoardNode(message proto.Message, player *model.PlayerModel) {
	resp, ok := message.(*rpcPb.CheckRankIsFullResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	_, activityId, actRankId, _ := logicCommon.GetRankRealIdFromUniqueId(resp.RankId)
	actRankCfg := gameConfig.GetRankCfgByIds(activityId, actRankId)
	if actRankCfg == nil || actRankCfg.ActId != activityId {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	fullActRankCfg := gameConfig.GetRankActCfg(actRankId)
	if fullActRankCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	drops := gameConfig.Drop(fullActRankCfg.AllDropId)
	if drops != nil {
		_ = itemService.AddItems(player, drops, enum.ITEM_CHANGE_REASON_RANK_CLAIM_BOX)
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_RANK_RECEIVE_BOX_REWARD_RESP, &pb.RankReceiveBoxRewardResp{})
}

func GetMyRankFromRankBoardNode(message proto.Message, rankId string, session serviceInterface.SessionInterface) {
	_, ok := message.(*rpcPb.GetMyRankReq)
	if !ok {
		logger.ErrorBySprintf("convert error, message is %v", message)
		return
	}
	rankSession, ok := session.(*logicSessionManager.RankBoardSession)
	if !ok {
		rankBoardPlatform.SendErrorMessageBySession(session, rankSession.RespMsgId, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	resp := &rpcPb.GetMyRankResp{
		RankId: rankId,
	}
	resp.Rank = &rpcPb.MyRankData{
		Rank: -1,
	}
	playerRank := rankboardService.GetPlayerRank(rankId, rankSession.UserId)
	if playerRank != nil {
		resp.Rank.Rank = playerRank.Rank
		resp.Rank.Score = playerRank.Score
	}
	rankBoardPlatform.SendMessageBySession(session, rankSession.RespMsgId, resp)
}

func OnUpdateRankInfoOnRankBoardNode(message proto.Message, rankId string, session serviceInterface.SessionInterface) {
	req, ok := message.(*rpcPb.NotifyUpdateRankInfo)
	if !ok {
		return
	}
	realRankId, actId, actRankId, _ := logicCommon.GetRankRealIdFromUniqueId(rankId)
	maxNum := int32(100)
	resort := true
	var commonRankCfg *gameConfig.CommonRankCfg
	if realRankId != 0 {
		commonRankCfg = gameConfig.GetRankCfgByIds(0, realRankId)
	} else {
		commonRankCfg = gameConfig.GetRankCfgByIds(actId, actRankId)
	}

	if commonRankCfg == nil {
		return
	}
	maxNum = commonRankCfg.PNMax
	if commonRankCfg.RankType == int32(enum.RANK_BOARD_RANK_RULE_ENTER_TIME) {
		resort = false
	}
	targetId, score, incrementalUpdate, err := checkIsAllianceRankBoard(rankId, commonRankCfg, req)
	if err != nil {
		return
	}
	if commonRankCfg.RankThreshold > 0 && commonRankCfg.RankThreshold > score {
		return
	}

	isEnter, rank := rankboardService.UpdatePlayerRankInfo(rankId, targetId, score, incrementalUpdate, maxNum, resort)
	if rank == -1 {
		logger.ErrorBySprintf("rankBoardNode update player rank info error, rankId: %s, userId: %d", rankId, targetId)
		return
	}
	// TODO: 之后移动到其它地方 Keep glory-arena pools up-to-date by appending users that newly satisfy topN on rank updates.
	if score > 0 && rank > 0 {
		now := tool.UnixNowMilli()
		if commonRankCfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_BATTLE_POWER) {
			if _, err = gloryArenaService.TryAppendByBattlePowerRankUpdate(rankId, targetId, rank, score, now); err != nil {
				logger.ErrorBySprintf("gloryArena TryAppendByBattlePowerRankUpdate error, rankId: %s, userId: %d, rank: %d, score: %d, err: %v", rankId, targetId, rank, score, err)
			}
		}
		if commonRankCfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_ARENA) {
			if _, err = gloryArenaService.TryAppendByArenaRankUpdate(rankId, targetId, rank, score, now); err != nil {
				logger.ErrorBySprintf("gloryArena TryAppendByArenaRankUpdate error, rankId: %s, userId: %d, rank: %d, score: %d, err: %v", rankId, targetId, rank, score, err)
			}
		}
	}

	if isEnter {
		if commonRankCfg.SendRewardType == int32(enum.RANK_BOARD_SEND_REWARD_TYPE_ENTER) {
			dropId := gameConfig.GetRankRewardCfgWithRank(commonRankCfg.RankRewardsId[0], rank)
			var dropItem []*gameConfig.ItemConfig
			if dropId != 0 {
				dropItem = gameConfig.Drop(dropId)
			}
			if len(dropItem) > 0 {
				rankBoardPlatform.SendRankBoardRewardMail(commonRankCfg.MailId[0], targetId, dropItem, rank)
			}
		}
	}
}

func checkIsAllianceRankBoard(rankId string, cfg *gameConfig.CommonRankCfg, req *rpcPb.NotifyUpdateRankInfo) (id int64, score int64, incrementalUpdate bool, err error) {
	if cfg == nil || req == nil {
		return 0, 0, false, fmt.Errorf("invalid rank update args")
	}
	finalId := req.PlayerId
	finalScore := req.Score
	finalIncrementalUpdate := req.IncrementalUpdate
	switch cfg.PointType {
	case int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA):
		if req.AllianceId > 0 {
			if err = checkAllianceRankServer(rankId, req.AllianceId); err != nil {
				return 0, 0, false, err
			}
			finalId = req.AllianceId
			break
		}
		if req.PlayerId <= 0 {
			return 0, 0, false, fmt.Errorf("alliance arena rank update missing playerId/allianceId")
		}
		allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(req.PlayerId)
		if allianceInfo == nil || allianceInfo.AllianceId <= 0 {
			return 0, 0, false, fmt.Errorf("player not in alliance, userId:%d", req.PlayerId)
		}
		if err = checkAllianceRankServer(rankId, allianceInfo.AllianceId); err != nil {
			return 0, 0, false, err
		}
		finalId = allianceInfo.AllianceId
		if !allianceInfo.ArenaJoined {
			playerBasicInfo := logicCommon.GetPlayerBasicInfoFromRedis(req.PlayerId)
			if playerBasicInfo == nil {
				return 0, 0, false, fmt.Errorf("player basic info is nil, userId:%d", req.PlayerId)
			}
			finalScore = int64(playerBasicInfo.ArenaScore)
			finalIncrementalUpdate = true
			allianceInfo.ArenaJoined = true
			logicCommon.UpdatePlayerAllianceInfo(allianceInfo)
		}
	case int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT):
		if req.AllianceId > 0 {
			finalId = req.AllianceId
			break
		}
		if req.PlayerId <= 0 {
			return 0, 0, false, fmt.Errorf("alliance glory arena rank update missing playerId/allianceId")
		}
		allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(req.PlayerId)
		if allianceInfo == nil || allianceInfo.AllianceId <= 0 {
			return 0, 0, false, fmt.Errorf("player not in alliance, userId:%d", req.PlayerId)
		}
		finalId = allianceInfo.AllianceId
	}
	if finalId <= 0 {
		return 0, 0, false, fmt.Errorf("invalid rank update target, playerId:%d allianceId:%d", req.PlayerId, req.AllianceId)
	}
	return finalId, finalScore, finalIncrementalUpdate, nil
}

func checkAllianceRankServer(rankId string, allianceId int64) error {
	alliance, code := LoadAllianceBasicInfoFromRedis(allianceId)
	if code != pb.ERROR_CODE_SUCCESS || alliance == nil || alliance.AllianceId <= 0 {
		return fmt.Errorf("alliance basic info not found, allianceId:%d", allianceId)
	}
	_, _, rankServerId, ok := logicCommon.ParseCommonArenaRankTableMeta(rankId)
	if ok && rankServerId > 0 && alliance.ServerId > 0 && alliance.ServerId != rankServerId {
		return fmt.Errorf("alliance server mismatch, rankId:%s allianceId:%d allianceServerId:%d rankServerId:%d",
			rankId, allianceId, alliance.ServerId, rankServerId)
	}
	return nil
}
