package socialService

import (
	"errors"
	"log"
	"slices"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/tool"
	"gorm.io/gorm"
)

type AllianceModel struct {
	alliance                *model.AllianceEntity
	members                 map[int64]*model.AllianceMemberEntity
	leaderName              string
	allianceWarehouseEntity map[int32]*model.AllianceWarehouseEntity

	dirty                  map[string]interface{}
	lastFlushTime          time.Time
	lastHeartbeatCheckTime time.Time
}

func (m *AllianceModel) GetAllianceId() int64 {
	return m.alliance.AllianceId
}

func (m *AllianceModel) HasMember(userID int64) bool {
	if m == nil || userID <= 0 {
		return false
	}
	_, ok := m.members[userID]
	return ok
}

func (m *AllianceModel) getLeaderUserID() int64 {
	if m == nil {
		return 0
	}
	for userID, member := range m.members {
		if member != nil && member.Role == int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER) {
			return userID
		}
	}
	return 0
}

func (m *AllianceModel) setLeaderNameCache(name string) {
	if m == nil {
		return
	}
	m.leaderName = name
	if m.alliance != nil {
		m.alliance.LeaderName = name
	}
}

func (m *AllianceModel) refreshLeaderNameCache() {
	leaderUserID := m.getLeaderUserID()
	if leaderUserID <= 0 {
		m.setLeaderNameCache("")
		return
	}
	leaderBasicInfo := logicCommon.GetPlayerBasicInfoFromRedis(leaderUserID)
	if leaderBasicInfo == nil {
		return
	}
	m.setLeaderNameCache(leaderBasicInfo.Name)
}

func (s *AllianceModel) GetAllianceInfo(req *rpcPb.GetAllianceInfoReq) (*rpcPb.GetAllianceInfoResp, pb.ERROR_CODE) {
	if req == nil || req.UserId <= 0 {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	leaderUserId := int64(0)
	memberIds := make([]int64, len(s.members))
	index := 0
	for _, member := range s.members {
		if member.Role == int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER) {
			leaderUserId = member.UserId
		}
		memberIds[index] = member.UserId
		index++
	}
	playerInfos := logicCommon.GetPlayerRedisInfos(memberIds)
	memberInfos := make([]*rpcPb.AllianceMember, len(s.members))
	warehouseInfos := make([]*rpcPb.AllianceWarehouseInfo, len(s.allianceWarehouseEntity))
	index = 0
	totalPower := int64(0)
	allianceArenaTotalScore := int64(0)
	allianceGloryRoundBestWinTotal := int64(0)
	for _, m := range s.members {
		playerInfo := playerInfos[m.UserId]
		if playerInfo != nil && playerInfo.BattleInfo != nil {
			battleInfo := playerInfo.BattleInfo
			totalPower += battleInfo.GetMainFormationPower()
		}
		if playerInfo != nil && playerInfo.BasicInfo != nil {
			allianceArenaTotalScore += int64(playerInfo.BasicInfo.ArenaScore)
			allianceGloryRoundBestWinTotal += int64(playerInfo.BasicInfo.GloryArenaBestWinCount)
		}
		memberInfos[index] = memberToPb(m, playerInfo)
		index++
	}
	index = 0
	for _, v := range s.allianceWarehouseEntity {
		warehouseInfos[index] = &rpcPb.AllianceWarehouseInfo{
			ItemId: v.ItemId,
			Count:  v.Count,
		}
		index++
	}
	if leaderUserId > 0 {
		leaderName := s.leaderName
		if leaderInfo := playerInfos[leaderUserId]; leaderInfo != nil && leaderInfo.BasicInfo != nil && leaderInfo.BasicInfo.Name != "" {
			leaderName = leaderInfo.BasicInfo.Name
		}
		s.setLeaderNameCache(leaderName)
	} else {
		s.setLeaderNameCache("")
	}
	s.alliance.AllianceTotalPower = totalPower
	syncAllianceBasicToRedis(s.alliance)
	syncAllianceRankFinalScores(req.UserId, s.alliance.ServerId, s.alliance.AllianceId, allianceArenaTotalScore, allianceGloryRoundBestWinTotal)
	resp := &rpcPb.GetAllianceInfoResp{
		Alliance:  allianceToPb(s.alliance, leaderUserId),
		Members:   memberInfos,
		Warehouse: warehouseInfos,
	}

	return resp, pb.ERROR_CODE_SUCCESS
}

func (m *AllianceModel) ChangeAllianceBasicInfo(req *rpcPb.ChangeAllianceBasicInfoReq) pb.ERROR_CODE {
	name := req.Name
	announce := req.Announce
	if req.UpdateName && name == "" {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	if req.UpdateBadgeId && req.BadgeId < 0 {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}

	member := m.members[req.OperatorUserId]
	if member == nil {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	cfg := gameConfig.GetAlliancePositionCfg(member.Role)
	if cfg == nil {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	if !slices.Contains(cfg.Permit, int32(enum.CHANGE_ALLIANCE_INFO)) {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}

	if m.dirty == nil {
		m.dirty = make(map[string]interface{})
	}

	if req.UpdateName {
		s := GetService()
		if s == nil || s.manager == nil {
			return pb.ERROR_CODE_SYSTEM_ERROR
		}
		exists, err := s.manager.nameExists(m.alliance.ServerId, name, req.AllianceId)
		if err != nil {
			return mapErrCode(err)
		}
		if exists {
			return pb.ERROR_CODE_ALLIANCE_NAME_ALREADY_EXISTS
		}
		if err := s.manager.claimRename(m.alliance.ServerId, m.alliance.Name, name, m.alliance.AllianceId); err != nil {
			return mapErrCode(err)
		}
		m.alliance.Name = name
		m.dirty["name"] = m.alliance.Name
		syncMemberAllianceInfoToRedis(m.alliance, m.members)
	}
	if req.UpdateAnnounce {
		m.alliance.Announce = announce
		m.dirty["announce"] = m.alliance.Announce
	}
	if req.UpdateNotice {
		m.alliance.Notice = req.Notice
		m.dirty["notice"] = m.alliance.Notice
	}
	if req.UpdateBadgeId {
		m.alliance.BadgeId = req.BadgeId
		m.dirty["badge_id"] = m.alliance.BadgeId
	}
	if m.alliance.ApplyType != req.ApplyType {
		m.alliance.ApplyType = req.ApplyType
		m.dirty["apply_type"] = m.alliance.ApplyType
	}
	if m.alliance.PowerApplyCondition != req.PowerApplyCondition {
		m.alliance.PowerApplyCondition = req.PowerApplyCondition
		m.dirty["power_apply_condition"] = m.alliance.PowerApplyCondition
	}
	if m.alliance.CityLevelCondition != req.CityLevel {
		m.alliance.CityLevelCondition = req.CityLevel
		m.dirty["city_level_condition"] = m.alliance.CityLevelCondition
	}
	syncAllianceBasicToRedis(m.alliance)
	return pb.ERROR_CODE_SUCCESS
}

func (s *AllianceModel) ApplyAlliance(req *rpcPb.ApplyAllianceReq) *rpcPb.ApplyAllianceResp {
	resp := &rpcPb.ApplyAllianceResp{
		ErrorCode: int32(pb.ERROR_CODE_SUCCESS),
	}
	if s.members[req.UserId] != nil {
		resp.ErrorCode = int32(pb.ERROR_CODE_ALLIANCE_ALREADY_IN_ALLIANCE)
		return resp
	}
	cfg := gameConfig.GetAllianceLevelCfg(s.alliance.Level)
	if cfg == nil {
		resp.ErrorCode = int32(pb.ERROR_CODE_SYSTEM_ERROR)
		return resp
	}
	if s.alliance.MemberNum >= cfg.Num {
		resp.ErrorCode = int32(pb.ERROR_CODE_ALLIANCE_IS_FULL)
		return resp
	}

	if s.alliance.ApplyType != enum.AllianceEnterType_Free {
		for _, m := range s.members {
			if m.Role == int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER) || m.Role == int32(pb.ALLIANCE_POSITION_ALLIANCE_COLEADER) {
				rpcController.NotifyAllianceOperationToGateway(m.UserId, s.alliance.AllianceId, pb.ALLIANCE_CHANGE_OPER_NEW_APPLY, nil)
			}
		}
		return nil
	}

	playerInfo := logicCommon.GetPlayerRedisInfo(req.UserId)
	if playerInfo == nil || playerInfo.BasicInfo.ServerId != req.ServerId {
		resp.ErrorCode = int32(pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return resp
	}
	mainPower := int64(0)
	if playerInfo.BattleInfo.FormationInfo != nil {
		mainFormation := playerInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)]
		if mainFormation != nil {
			mainPower = mainFormation.BattlePower
		}
	}
	if mainPower < s.alliance.PowerApplyCondition || playerInfo.BasicInfo.MainCityLevel < s.alliance.CityLevelCondition {
		resp.ErrorCode = int32(pb.ERROR_CODE_ALLIANCE_ENTER_CONDITION_IS_NOT_MATCH)
		return resp
	}

	now := tool.UnixNowMilli()
	var joinedMember *model.AllianceMemberEntity
	err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
		joinedMember = &model.AllianceMemberEntity{
			AllianceId: req.AllianceId,
			UserId:     req.UserId,
			Role:       int32(pb.ALLIANCE_POSITION_ALLIANCE_COMMON_MEMBER),
			JoinTime:   now,
		}
		if err := tx.Create(joinedMember).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return errAllianceAlreadyInAlliance
			}
			return err
		}
		return nil
	})
	if err != nil {
		resp.ErrorCode = int32(mapErrCode(err))
		return resp
	}
	s.members[req.UserId] = joinedMember
	s.alliance.MemberNum++
	syncAllianceMemberSetToRedis(s.alliance.AllianceId, s.members)
	roundBestWin := playerInfo.BasicInfo.GloryArenaBestWinCount
	s.alliance.AllianceTotalPower += playerInfo.BattleInfo.GetMainFormationPower()
	logicCommon.UpdatePlayerAllianceInfo(&logicCommon.PlayerAllianceInfo{
		ArenaJoined:  false,
		RoundBestWin: roundBestWin,
		UserId:       req.UserId,
		AllianceId:   s.alliance.AllianceId,
		AllianceName: s.alliance.Name,
		JoinTime:     joinedMember.JoinTime,
	})
	if roundBestWin > 0 {
		notifyAllianceGloryArenaRoundRankDelta(req.UserId, s.alliance.ServerId, s.alliance.AllianceId, int64(roundBestWin))
	}
	syncAllianceBasicToRedis(s.alliance)
	removeAllianceApplyFromRedis(req.AllianceId, req.UserId)
	resp.Alliance = allianceToPb(s.alliance, 0)
	resp.ErrorCode = int32(pb.ERROR_CODE_SUCCESS)
	return resp
}

func (s *AllianceModel) ApproveAllianceApply(req *rpcPb.ApproveAllianceApplyReq) (*rpcPb.ApproveAllianceApplyResp, pb.ERROR_CODE) {
	member := s.members[req.OperatorUserId]
	if member == nil {
		return nil, pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	cfg := gameConfig.GetAlliancePositionCfg(member.Role)
	if cfg == nil {
		return nil, pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	if !slices.Contains(cfg.Permit, int32(enum.APPROVE_ALLIANCE_APPLY)) {
		return nil, pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}

	levelCfg := gameConfig.GetAllianceLevelCfg(s.alliance.Level)
	if levelCfg == nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	if s.alliance.MemberNum >= levelCfg.Num {
		return nil, pb.ERROR_CODE_ALLIANCE_IS_FULL
	}

	applyUserID := req.ApplyUserId
	now := tool.UnixNowMilli()
	hasApply, applyErr := getAllianceApplyTimeFromRedis(req.AllianceId, applyUserID)
	if applyErr != nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR
	}
	if !hasApply {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}

	if !req.ApproveApply {
		removeAllianceApplyFromRedis(req.AllianceId, applyUserID)
		GetService().SendAllianceMail(gameConfig.GetAllianceRefuseMailId(), applyUserID, s.alliance.Name)
		return &rpcPb.ApproveAllianceApplyResp{}, pb.ERROR_CODE_SUCCESS
	}

	var approvedMember *model.AllianceMemberEntity
	err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
		approvedMember = &model.AllianceMemberEntity{
			AllianceId: req.AllianceId,
			UserId:     applyUserID,
			Role:       int32(pb.ALLIANCE_POSITION_ALLIANCE_COMMON_MEMBER),
			JoinTime:   now,
		}
		if err := tx.Create(approvedMember).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return errAllianceAlreadyInAlliance
			}
			return err
		}
		return nil
	})
	if err != nil {
		code := mapErrCode(err)
		if code == pb.ERROR_CODE_ALLIANCE_ALREADY_IN_ALLIANCE {
			removeAllianceApplyFromRedis(req.AllianceId, applyUserID)
		}
		return nil, code
	}

	s.members[applyUserID] = approvedMember
	s.alliance.MemberNum++
	syncAllianceMemberSetToRedis(s.alliance.AllianceId, s.members)
	roundBestWin := logicCommon.GetOtherPlayerGloryArenaRoundBestWin(applyUserID)
	playerBattleInfo := logicCommon.GetPlayerBattleInfoFromRedis(applyUserID)
	if playerBattleInfo != nil {
		s.alliance.AllianceTotalPower += playerBattleInfo.GetMainFormationPower()
	}
	logicCommon.UpdatePlayerAllianceInfo(&logicCommon.PlayerAllianceInfo{
		ArenaJoined:  false,
		RoundBestWin: roundBestWin,
		UserId:       applyUserID,
		AllianceId:   s.alliance.AllianceId,
		AllianceName: s.alliance.Name,
		JoinTime:     approvedMember.JoinTime,
	})
	if roundBestWin > 0 {
		notifyAllianceGloryArenaRoundRankDelta(applyUserID, s.alliance.ServerId, s.alliance.AllianceId, int64(roundBestWin))
	}
	syncAllianceBasicToRedis(s.alliance)
	removeAllianceApplyFromRedis(req.AllianceId, applyUserID)
	rpcController.NotifyAllianceOperationToGateway(applyUserID, s.alliance.AllianceId, pb.ALLIANCE_CHANGE_OPER_ENTER, nil)
	return &rpcPb.ApproveAllianceApplyResp{}, pb.ERROR_CODE_SUCCESS
}

func (s *AllianceModel) KickAllianceMember(req *rpcPb.KickAllianceMemberReq) pb.ERROR_CODE {
	if req.OperatorUserId == req.TargetUserId {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}

	operatorMember := s.members[req.OperatorUserId]
	if operatorMember == nil {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	cfg := gameConfig.GetAlliancePositionCfg(operatorMember.Role)
	if cfg == nil {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	if !slices.Contains(cfg.Permit, int32(enum.CHANGE_ALLIANCE_MEMBER_ROLE)) {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}

	targetMember := s.members[req.TargetUserId]
	if targetMember == nil || targetMember.Role == int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER) {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}

	err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("alliance_id = ? AND user_id = ?", req.AllianceId, req.TargetUserId).Delete(&model.AllianceMemberEntity{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return mapErrCode(err)
	}
	delete(s.members, req.TargetUserId)
	s.alliance.MemberNum--
	syncAllianceMemberSetToRedis(s.alliance.AllianceId, s.members)
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(req.TargetUserId)
	if allianceInfo != nil {
		if allianceInfo.ArenaJoined {
			notifyAllianceArenaRankDelta(req.TargetUserId, s.alliance.ServerId, s.alliance.AllianceId, -logicCommon.GetOtherPlayerArenaScoreFromRedis(req.TargetUserId))
		}
		if allianceInfo.RoundBestWin > 0 {
			notifyAllianceGloryArenaRoundRankDelta(req.TargetUserId, s.alliance.ServerId, s.alliance.AllianceId, -int64(allianceInfo.RoundBestWin))
		}
	}
	logicCommon.UpdatePlayerAllianceInfo(&logicCommon.PlayerAllianceInfo{
		ArenaJoined:  false,
		RoundBestWin: 0,
		UserId:       req.TargetUserId,
		AllianceId:   0,
		AllianceName: "",
		JoinTime:     0,
	})
	playerBattleInfo := logicCommon.GetPlayerBattleInfoFromRedis(req.TargetUserId)
	if playerBattleInfo != nil {
		s.alliance.AllianceTotalPower -= playerBattleInfo.GetMainFormationPower()
		if s.alliance.AllianceTotalPower < 0 {
			s.alliance.AllianceTotalPower = 0
		}
	}
	syncAllianceBasicToRedis(s.alliance)
	GetService().SendAllianceMail(gameConfig.GetAllianceKickMailId(), req.TargetUserId, s.alliance.Name)
	rpcController.NotifyAllianceOperationToGateway(req.TargetUserId, 0, pb.ALLIANCE_CHANGE_OPER_KICKOUT, nil)
	return pb.ERROR_CODE_SUCCESS
}

func (s *AllianceModel) ChangeMemberPosition(req *rpcPb.ChangeMemberPositionReq) pb.ERROR_CODE {
	if req.OperatorUserId == req.TargetUserId {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}

	targetMember := s.members[req.TargetUserId]
	if targetMember == nil {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	operatorMember := s.members[req.OperatorUserId]
	if operatorMember == nil {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	cfg := gameConfig.GetAlliancePositionCfg(operatorMember.Role)
	if cfg == nil {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	if !slices.Contains(cfg.Permit, int32(enum.CHANGE_ALLIANCE_MEMBER_ROLE)) {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	if targetMember.Role == int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER) {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}

	if req.TargetRole == int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER) {
		if operatorMember.Role != int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER) {
			return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
		}
		err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&model.AllianceMemberEntity{}).
				Where("alliance_id = ? AND user_id = ?", req.AllianceId, req.TargetUserId).
				Updates(map[string]interface{}{"role": req.TargetRole}).Error
			if err != nil {
				return err
			}
			err = tx.Model(&model.AllianceMemberEntity{}).
				Where("alliance_id = ? AND user_id = ?", req.AllianceId, req.OperatorUserId).
				Updates(map[string]interface{}{"role": int32(pb.ALLIANCE_POSITION_ALLIANCE_COMMON_MEMBER)}).Error
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return mapErrCode(err)
		}
		targetMember.Role = req.TargetRole
		operatorMember.Role = int32(pb.ALLIANCE_POSITION_ALLIANCE_COMMON_MEMBER)
		s.refreshLeaderNameCache()
	} else {
		tempCfg := gameConfig.GetAlliancePositionCfg(req.TargetRole)
		if tempCfg == nil {
			return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
		}
		num := int32(0)
		for _, member := range s.members {
			if member.Role == req.TargetRole {
				num++
			}
		}
		if tempCfg.PlayerNum != -1 && num >= tempCfg.PlayerNum {
			return pb.ERROR_CODE_ALLIANCE_POSITON_IS_FULL
		}
		err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
			return tx.Model(&model.AllianceMemberEntity{}).
				Where("alliance_id = ? AND user_id = ?", req.AllianceId, req.TargetUserId).
				Updates(map[string]interface{}{"role": req.TargetRole}).Error
		})
		if err != nil {
			return mapErrCode(err)
		}
		targetMember.Role = req.TargetRole
	}

	syncAllianceBasicToRedis(s.alliance)
	return pb.ERROR_CODE_SUCCESS
}

func (s *AllianceModel) QuitAlliance(req *rpcPb.QuitAllianceReq) pb.ERROR_CODE {
	member := s.members[req.UserId]
	if member == nil {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	if member.Role == int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER) {
		return pb.ERROR_CODE_ALLIANCE_LEADER_NOT_ALLOW_LEAVE_ALLIANCE
	}

	err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("alliance_id = ? AND user_id = ?", req.AllianceId, req.UserId).Delete(&model.AllianceMemberEntity{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return mapErrCode(err)
	}
	delete(s.members, req.UserId)
	s.alliance.MemberNum--
	syncAllianceMemberSetToRedis(s.alliance.AllianceId, s.members)
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(req.UserId)
	if allianceInfo != nil {
		if allianceInfo.ArenaJoined {
			notifyAllianceArenaRankDelta(req.UserId, s.alliance.ServerId, s.alliance.AllianceId, -logicCommon.GetOtherPlayerArenaScoreFromRedis(req.UserId))
		}
		if allianceInfo.RoundBestWin > 0 {
			notifyAllianceGloryArenaRoundRankDelta(req.UserId, s.alliance.ServerId, s.alliance.AllianceId, -int64(allianceInfo.RoundBestWin))
		}
	}
	logicCommon.UpdatePlayerAllianceInfo(&logicCommon.PlayerAllianceInfo{
		ArenaJoined:  false,
		RoundBestWin: 0,
		UserId:       req.UserId,
		AllianceId:   0,
		AllianceName: "",
		JoinTime:     0,
	})
	playerBattleInfo := logicCommon.GetPlayerBattleInfoFromRedis(req.UserId)
	if playerBattleInfo != nil {
		s.alliance.AllianceTotalPower -= playerBattleInfo.GetMainFormationPower()
		if s.alliance.AllianceTotalPower < 0 {
			s.alliance.AllianceTotalPower = 0
		}
	}
	syncAllianceBasicToRedis(s.alliance)
	return pb.ERROR_CODE_SUCCESS
}

func (a *AllianceModel) AddItems(items []*rpcPb.ItemInfo) error {
	for _, v := range items {
		err := a.PushItemToBag(v)
		if err != nil {
			return err
		}
	}
	a.checkLevelUp()
	return nil
}

func (a *AllianceModel) RemoveItems(items []*rpcPb.ItemInfo) error {
	for _, v := range items {
		_, err := a.CheckItemCount(v)
		if err != nil {
			return err
		}
		err = a.RemoveItemFromBag(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *AllianceModel) CheckItemCount(item *rpcPb.ItemInfo) (bool, error) {
	count, err := a.GetItemCountFromBag(item)
	if err != nil {
		return false, err
	}
	if count >= item.Count {
		return true, nil
	}
	return false, nil
}

func (a *AllianceModel) GetItemCount(item *rpcPb.ItemInfo) (int64, error) {
	return a.GetItemCountFromBag(item)
}

func (s *AllianceModel) PushItemToBag(item *rpcPb.ItemInfo) error {
	if s.allianceWarehouseEntity[item.ItemId] == nil {
		err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
			itemInfo := &model.AllianceWarehouseEntity{
				AllianceId: s.alliance.AllianceId,
				ItemId:     item.ItemId,
				Count:      item.Count,
			}
			if err := tx.Create(itemInfo).Error; err != nil {
				return err
			}
			s.allianceWarehouseEntity[item.ItemId] = itemInfo
			return nil
		})
		if err != nil {
			log.Println("PushItemToBag error:", err)
			return err
		}
	} else {
		s.allianceWarehouseEntity[item.ItemId].Count += item.Count
		err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&model.AllianceWarehouseEntity{}).
				Where("alliance_id = ? AND item_id = ?", s.alliance.AllianceId, item.ItemId).
				Updates(map[string]interface{}{"count": s.allianceWarehouseEntity[item.ItemId].Count}).Error
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Println("PushItemToBag error:", err)
			return err
		}
	}
	return nil
}

func (s *AllianceModel) RemoveItemFromBag(item *rpcPb.ItemInfo) error {
	if s.allianceWarehouseEntity[item.ItemId] == nil {
		return errors.New("alliance item not found")
	}
	s.allianceWarehouseEntity[item.ItemId].Count -= item.Count

	err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&model.AllianceWarehouseEntity{}).
			Where("alliance_id = ? AND item_id = ?", s.alliance.AllianceId, item.ItemId).
			Updates(map[string]interface{}{"count": s.allianceWarehouseEntity[item.ItemId].Count}).Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *AllianceModel) GetItemCountFromBag(item *rpcPb.ItemInfo) (int64, error) {
	if s.allianceWarehouseEntity[item.ItemId] == nil {
		return 0, errors.New("alliance item not found")
	}
	return s.allianceWarehouseEntity[item.ItemId].Count, nil
}

func (s *AllianceModel) GetAllianceMember() []int64 {
	res := make([]int64, 0)
	for k, _ := range s.members {
		res = append(res, k)
	}
	return res
}

func (s *AllianceModel) UpdateLastTickTime(time int64) {
	s.alliance.LastTickTime = time
	err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&model.AllianceEntity{}).
			Where("alliance_id = ?", s.alliance.AllianceId).
			Updates(map[string]interface{}{"last_tick_time": s.alliance.LastTickTime}).Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Println("UpdateLastTickTime error:", err)
	}
}

func (m *AllianceModel) checkLevelUp() {
	exp, ok := m.allianceWarehouseEntity[enum.ALLIANCE_TASK_EXP_ITEM_ID]
	if !ok {
		return
	}
	nextLevel := m.alliance.Level + 1
	for {
		cfg := gameConfig.GetAllianceLevelCfg(nextLevel)
		if cfg == nil {
			return
		}
		if exp.Count >= int64(cfg.ContribValue) && cfg.ContribValue != 0 {
			m.alliance.Level = nextLevel
			m.dirty["level"] = m.alliance.Level
			nextLevel = m.alliance.Level + 1
		} else {
			return
		}
	}
}
