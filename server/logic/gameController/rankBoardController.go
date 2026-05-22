package gameController

import (
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
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANKBOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_RANK_INFO_REQ, &rpcPb.GetRankInfoReq{}, GetRankListFromRankBoardNode)
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANKBOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_THUMB_UP_RANK_INFO, &rpcPb.ThumbUpRankInfo{}, OnThumbUpRankInfoOnRankBoardNode)
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANKBOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_CHECK_RANK_FULL_REQ, &rpcPb.CheckRankIsFullReq{}, CheckRankIsFullOnRankBoardNode)
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANKBOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_MY_RANK_REQ, &rpcPb.GetMyRankReq{}, GetMyRankFromRankBoardNode)
	RegisterRankBoardMessageHandler(enum.MSG_TYPE_RANKBOARD, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, &rpcPb.NotifyUpdateRankInfo{}, OnUpdatePlayerRankInfoOnRankBoardNode)
}
func GetRankList(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.RankListReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	version := ""
	if req.ActId != 0 {
		if ok, version = player.PlayerActivityModel.CheckActivityOpen(req.ActId); !ok {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIKE_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
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

	rankId, err := logicCommon.GetRankUniqueId(req.CommonRankId, req.ActId, req.ActTargetId, player.User.GetServerId(), version)
	if err != nil {
		logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	rpcReq := &rpcPb.GetRankInfoReq{}
	err = rpcController.SendMessageToRankBoard(player.GetUserId(), rankId, int32(pb.MESSAGE_ID_RANK_LIST_RESP), rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_RANK_INFO_REQ, rpcReq)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_RANK_BOARD_NOT_FOUND)
		return
	}
}
func GetRankListFromRankBoardNode(message proto.Message, rankId string, session serviceInterface.SessionInterface) {
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

	rankSession, ok := session.(*logicSessionManager.RankBoardSession)
	if !ok {
		rankBoardPlatform.SendErrorMessageBySession(session, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_RANK_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	queryUserId := rankSession.UserId
	if rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA) ||
		rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT) {
		allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(rankSession.UserId)
		if allianceInfo != nil && allianceInfo.AllianceId > 0 {
			queryUserId = allianceInfo.AllianceId
		} else {
			queryUserId = 0
		}
	}

	rankDetailInfo, playerRank, _ := rankboardService.GetRankInfo(rankId, int(maxNum), queryUserId)
	resp := &rpcPb.GetRankInfoResp{
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
	resp.RankInfos = detail
	rankBoardPlatform.SendMessageBySession(session, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_RANK_INFO_RESP, resp)
}

func BackRankListFromRankBoardNode(message proto.Message, player *model.PlayerModel) {
	rpcMessage, ok := message.(*rpcPb.GetRankInfoResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	_, activityId, _, _ := logicCommon.GetRankRealIdFromUniqueId(rpcMessage.RankId)
	isLiked := false
	log := &model.PlayerRankThumbUpLog{
		RankId:       strconv.FormatInt(int64(activityId), 10),
		FromPlayerId: player.User.GetUserId(),
		ThumbUpTime:  tool.TodayZero().UnixMilli(),
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
	detail := make([]*pb.RankDetailInfo, rankLen)
	if rankLen == 0 {
		messageSender.SendMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, &pb.RankListResp{})
		return
	}

	index := int32(0)
	originalScore := int64(0)
	if rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ARENA) {
		originalScore = int64(gameConfig.GetArenaInitScore())
	}
	isAllianceRank := rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA) ||
		rankPointType == int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT)
	for ; index < int32(len(rpcMessage.RankInfos)); index++ {
		info := rpcMessage.RankInfos[index]
		detail[index] = &pb.RankDetailInfo{
			PlayerId:  info.Id,
			Rank:      info.Rank,
			LikeNum:   info.ThumbUpCount,
			RankScore: info.Score + originalScore,
			RankedOn:  info.EnterRankTime,
		}
		if isAllianceRank {
			alliance, code := LoadAllianceBasicInfoFromRedis(info.Id)
			if code != pb.ERROR_CODE_SUCCESS || alliance == nil {
				continue
			}
			detail[index].NickName = alliance.Name
			detail[index].Head = 0
			detail[index].FrameId = 0
			detail[index].BattlePower = make(map[int32]int64)
			detail[index].Title = 0
			detail[index].BubbleId = 0
			detail[index].ImageId = 0
		} else if info.Id != player.User.GetUserId() {
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
	resp := &pb.RankListResp{
		RankInfo: rankInfo,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_RANK_LIST_RESP, resp)
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
		ThumbUpTime:  tool.TodayZero().UnixMilli(),
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

func OnUpdatePlayerRankInfoOnRankBoardNode(message proto.Message, rankId string, session serviceInterface.SessionInterface) {
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
	if commonRankCfg.RankThreshold > req.Score {
		return
	}

	score := req.Score
	targetId, err := checkIsAllianceRankBoard(commonRankCfg, req)
	if err != nil {
		logger.ErrorBySprintf("checkIsAllianceRankBoard err: %v", err)
		return
	}

	isEnter, rank := rankboardService.UpdatePlayerRankInfo(rankId, targetId, score, req.IncrementalUpdate, maxNum, resort)
	if rank == -1 {
		logger.ErrorBySprintf("rankBoardNode update player rank info error, rankId: %s, userId: %d", rankId, targetId)
		return
	}
	// TODO: 之后移动到其它地方 Keep glory-arena pools up-to-date by appending users that newly satisfy topN on rank updates.
	if score > 0 && rank > 0 {
		now := tool.UnixNowMilli()
		if _, err = gloryArenaService.TryAppendByBattlePowerRankUpdate(rankId, targetId, rank, score, now); err != nil {
			logger.ErrorBySprintf("gloryArena TryAppendByBattlePowerRankUpdate error, rankId: %s, userId: %d, rank: %d, score: %d, err: %v", rankId, targetId, rank, score, err)
		}
		if _, err = gloryArenaService.TryAppendByArenaRankUpdate(rankId, targetId, rank, score, now); err != nil {
			logger.ErrorBySprintf("gloryArena TryAppendByArenaRankUpdate error, rankId: %s, userId: %d, rank: %d, score: %d, err: %v", rankId, targetId, rank, score, err)
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
				rankBoardPlatform.SendRankBoardRewardMail(commonRankCfg.MailId[0], req.Id, dropItem, rank)
			}
		}
	}
}

func checkIsAllianceRankBoard(cfg *gameConfig.CommonRankCfg, req *rpcPb.NotifyUpdateRankInfo) (id int64, err error) {
	finalId := req.Id
	switch cfg.PointType {
	case int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA):
		allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(req.Id)
		if allianceInfo != nil && allianceInfo.AllianceId > 0 {
			finalId = allianceInfo.AllianceId
			// TODO:联盟因为临时屏蔽，之后再处理这个情况
			//if !allianceInfo.ArenaJoined {
			//	playerBasicInfo := logicCommon.GetPlayerBasicInfoFromRedis(req.Id)
			//	if playerBasicInfo == nil {
			//		return
			//	}
			//	score = int64(playerBasicInfo.ArenaScore)
			//	incrementalUpdate = true
			//	allianceInfo.ArenaJoined = true
			//	logicCommon.UpdatePlayerAllianceInfo(allianceInfo)
			//}
		}
	case int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT):
		allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(req.Id)
		if allianceInfo != nil && allianceInfo.AllianceId > 0 {
			finalId = allianceInfo.AllianceId
		}
	}
	return finalId, nil
}
