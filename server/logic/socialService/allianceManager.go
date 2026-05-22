package socialService

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"gorm.io/gorm"
)

type AllianceManager struct {
	mu            sync.RWMutex
	allianceInfos map[int64]*AllianceModel
	nameIndex     map[int32]map[string]int64
	flushInterval time.Duration
}

func NewAllianceManager(flushInterval time.Duration) *AllianceManager {
	if flushInterval <= 0 {
		flushInterval = 2 * time.Second
	}
	return &AllianceManager{
		allianceInfos: make(map[int64]*AllianceModel),
		nameIndex:     make(map[int32]map[string]int64),
		flushInterval: flushInterval,
	}
}

func (m *AllianceManager) LoadAlliances() {
	alliances, err := easyDB.GetPlayerEntitiesByWhere[model.AllianceEntity](map[string]interface{}{})
	if err != nil {
		logger.ErrorBySprintf("[allianceManager] load alliances failed err:%v", err)
		return
	}
	members, err := easyDB.GetPlayerEntitiesByWhere[model.AllianceMemberEntity](map[string]interface{}{})
	if err != nil {
		logger.ErrorBySprintf("[allianceManager] load alliance members failed err:%v", err)
		return
	}

	memberMap := make(map[int64]map[int64]*model.AllianceMemberEntity)
	memberIdMap := make(map[int64][]int64)
	for _, member := range members {
		if member == nil {
			continue
		}
		allianceMembers := memberMap[member.AllianceId]
		if allianceMembers == nil {
			allianceMembers = make(map[int64]*model.AllianceMemberEntity)
			memberMap[member.AllianceId] = allianceMembers
			memberIdMap[member.AllianceId] = make([]int64, 0)
		}
		memberIdMap[member.AllianceId] = append(memberIdMap[member.AllianceId], member.UserId)
		allianceMembers[member.UserId] = member
	}

	newActors := make(map[int64]*AllianceModel, len(alliances))
	newNameIndex := make(map[int32]map[string]int64)
	for _, alliance := range alliances {
		if alliance == nil {
			continue
		}
		modelObj := &AllianceModel{
			alliance: alliance,
			members:  memberMap[alliance.AllianceId],
			dirty:    make(map[string]interface{}),
		}
		if modelObj.members == nil {
			modelObj.members = make(map[int64]*model.AllianceMemberEntity)
		} else {
			playerBattleInfos := logicCommon.GetPlayerBattleInfosFromRedis(memberIdMap[alliance.AllianceId])
			for _, playerBattleInfo := range playerBattleInfos {
				modelObj.alliance.AllianceTotalPower += playerBattleInfo.GetMainFormationPower()
			}
		}
		modelObj.refreshLeaderNameCache()

		newActors[alliance.AllianceId] = modelObj

		name := alliance.Name
		if name != "" {
			serverMap := newNameIndex[alliance.ServerId]
			if serverMap == nil {
				serverMap = make(map[string]int64)
				newNameIndex[alliance.ServerId] = serverMap
			}
			serverMap[name] = alliance.AllianceId
		}
		modelObj.alliance.MemberNum = int32(len(modelObj.members))
	}

	m.mu.Lock()
	m.allianceInfos = newActors
	m.nameIndex = newNameIndex
	m.mu.Unlock()
	rebuildAlliancesBasicToRedis(alliances)
	rebuildAllianceMemberInfoToRedis(newActors)
}

func (m *AllianceManager) attachCreatedAlliance(alliance *model.AllianceEntity, members []*model.AllianceMemberEntity) {
	if alliance == nil {
		return
	}
	modelObj := &AllianceModel{
		alliance: alliance,
		members:  make(map[int64]*model.AllianceMemberEntity),
		dirty:    make(map[string]interface{}),
	}
	for _, member := range members {
		if member == nil {
			continue
		}
		modelObj.members[member.UserId] = member
	}
	modelObj.alliance.MemberNum = int32(len(modelObj.members))
	modelObj.refreshLeaderNameCache()

	m.mu.Lock()
	m.allianceInfos[alliance.AllianceId] = modelObj
	m.mu.Unlock()
	m.setNameIndex(alliance.ServerId, alliance.Name, alliance.AllianceId)
	syncAllianceBasicToRedis(alliance)
}

func (m *AllianceManager) GetAllianceById(allianceID int64) *AllianceModel {
	if allianceID <= 0 {
		return nil
	}
	alliance, err := m.getOrLoadAllianceByID(allianceID)
	if err != nil {
		return nil
	}
	return alliance
}

func (m *AllianceManager) getOrLoadAllianceByID(allianceID int64) (*AllianceModel, error) {
	m.mu.RLock()
	if actor, ok := m.allianceInfos[allianceID]; ok {
		m.mu.RUnlock()
		return actor, nil
	}
	m.mu.RUnlock()

	alliance, err := easyDB.GetPlayerEntityByWhere[model.AllianceEntity](map[string]interface{}{"alliance_id": allianceID})
	if err != nil {
		return nil, err
	}
	members, err := easyDB.GetPlayerEntitiesByWhere[model.AllianceMemberEntity](map[string]interface{}{"alliance_id": allianceID})
	if err != nil {
		return nil, err
	}
	modelObj := &AllianceModel{
		alliance: alliance,
		members:  make(map[int64]*model.AllianceMemberEntity, len(members)),
		dirty:    make(map[string]interface{}),
	}
	for _, member := range members {
		if member == nil {
			continue
		}
		modelObj.members[member.UserId] = member
	}
	modelObj.refreshLeaderNameCache()

	m.mu.Lock()
	if exist, ok := m.allianceInfos[allianceID]; ok {
		m.mu.Unlock()
		return exist, nil
	}
	m.allianceInfos[allianceID] = modelObj
	m.mu.Unlock()

	m.setNameIndex(modelObj.alliance.ServerId, modelObj.alliance.Name, modelObj.alliance.AllianceId)
	modelObj.alliance.MemberNum = int32(len(members))
	return modelObj, nil
}

func (m *AllianceManager) nameExists(serverID int32, name string, excludeAllianceID int64) (bool, error) {
	n := name
	if n == "" {
		return false, nil
	}

	m.mu.RLock()
	serverMap := m.nameIndex[serverID]
	if serverMap != nil {
		if allianceID, ok := serverMap[n]; ok {
			if excludeAllianceID <= 0 || allianceID != excludeAllianceID {
				m.mu.RUnlock()
				return true, nil
			}
		}
	}
	m.mu.RUnlock()

	query := easyDB.GetPlayerDB().Where("server_id = ? AND name = ?", serverID, n)
	if excludeAllianceID > 0 {
		query = query.Where("alliance_id <> ?", excludeAllianceID)
	}
	entity := model.AllianceEntity{}
	err := query.Take(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	m.setNameIndex(entity.ServerId, entity.Name, entity.AllianceId)
	return true, nil
}

func (m *AllianceManager) claimRename(serverID int32, oldName, newName string, allianceID int64) error {
	oldNorm := oldName
	newNorm := newName
	if oldNorm == newNorm {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	serverMap := m.nameIndex[serverID]
	if serverMap == nil {
		serverMap = make(map[string]int64)
		m.nameIndex[serverID] = serverMap
	}
	if existID, ok := serverMap[newNorm]; ok && existID != allianceID {
		return errAllianceNameAlreadyExists
	}
	if oldNorm != "" {
		if existID, ok := serverMap[oldNorm]; ok && existID == allianceID {
			delete(serverMap, oldNorm)
		}
	}
	serverMap[newNorm] = allianceID
	return nil
}

func (m *AllianceManager) FlushDirtyByProcessor(processorID int32, processorNum int) {
	if processorNum <= 0 {
		return
	}
	now := time.Now()
	m.mu.RLock()
	targets := make([]*AllianceModel, 0)
	for allianceID, modelObj := range m.allianceInfos {
		index := int(allianceID % int64(processorNum))
		if index < 0 {
			index = -index
		}
		if int32(index) != processorID {
			continue
		}
		targets = append(targets, modelObj)
	}
	m.mu.RUnlock()

	for _, modelObj := range targets {
		m.flushIfNeed(modelObj, now)
	}
}

func (m *AllianceManager) HeartbeatByProcessor(processorID int32, processorNum int) {
	if processorNum <= 0 {
		return
	}
	now := time.Now()
	m.mu.RLock()
	targets := make([]*AllianceModel, 0)
	for allianceID, modelObj := range m.allianceInfos {
		index := int(allianceID % int64(processorNum))
		if index < 0 {
			index = -index
		}
		if int32(index) != processorID {
			continue
		}
		targets = append(targets, modelObj)
	}
	m.mu.RUnlock()

	for _, modelObj := range targets {
		m.heartbeatIfNeed(modelObj, now)
	}
}

func (m *AllianceManager) flushIfNeed(modelObj *AllianceModel, now time.Time) {
	if modelObj == nil {
		return
	}
	if modelObj.alliance == nil || len(modelObj.dirty) == 0 {
		return
	}
	if !modelObj.lastFlushTime.IsZero() && now.Sub(modelObj.lastFlushTime) < m.flushInterval {
		return
	}
	allianceID := modelObj.alliance.AllianceId
	updates := copyUpdates(modelObj.dirty)
	modelObj.dirty = make(map[string]interface{})
	modelObj.lastFlushTime = now

	err := easyDB.GetPlayerDB().
		Model(&model.AllianceEntity{}).
		Where("alliance_id = ?", allianceID).
		Updates(updates).Error
	if err == nil {
		return
	}

	logger.ErrorBySprintf("[allianceManager] flush alliance failed allianceId:%d err:%v", allianceID, err)
	for key, value := range updates {
		if _, ok := modelObj.dirty[key]; ok {
			continue
		}
		modelObj.dirty[key] = value
	}
}

func (m *AllianceManager) heartbeatIfNeed(modelObj *AllianceModel, now time.Time) {
	if modelObj == nil || modelObj.alliance == nil {
		return
	}
	if !modelObj.lastHeartbeatCheckTime.IsZero() && now.Sub(modelObj.lastHeartbeatCheckTime) < enum.AllianceHeartbeatCheckInterval {
		return
	}
	modelObj.lastHeartbeatCheckTime = now

	memberIds := make([]int64, len(modelObj.members))
	index := 0
	for id, _ := range modelObj.members {
		memberIds[index] = id
		index++
	}

	playerCacheInfos := logicCommon.GetPlayerRedisInfos(memberIds)
	nowMilli := now.UnixMilli()
	oldLeaderID, newLeaderID, err := m.tryAutoTransferLeaderForHeartbeat(modelObj, playerCacheInfos, nowMilli)
	if err != nil {
		logger.ErrorBySprintf("[allianceManager] heartbeat transfer leader failed allianceId:%d err:%v", modelObj.alliance.AllianceId, err)
	}
	if oldLeaderID > 0 && newLeaderID > 0 && oldLeaderID != newLeaderID {
		logger.InfoWithSprintf("[allianceManager] heartbeat transfer leader success allianceId:%d oldLeader:%d newLeader:%d", modelObj.alliance.AllianceId, oldLeaderID, newLeaderID)
	}
	modelObj.alliance.AllianceTotalPower = 0
	for _, info := range playerCacheInfos {
		if info.BattleInfo != nil {
			modelObj.alliance.AllianceTotalPower += info.BattleInfo.GetMainFormationPower()
		}
	}
	modelObj.refreshLeaderNameCache()
	syncAllianceBasicToRedis(modelObj.alliance)

	shouldDissolve := m.shouldAutoDissolveByOfflineForHeartbeat(playerCacheInfos, nowMilli)
	if !shouldDissolve {
		return
	}
	if err = m.dissolveAlliance(modelObj); err != nil {
		logger.ErrorBySprintf("[allianceManager] heartbeat dissolve failed allianceId:%d err:%v", modelObj.alliance.AllianceId, err)
		return
	}
	for _, member := range modelObj.members {
		GetService().SendAllianceMail(gameConfig.GetAllianceDissolveMailId(), member.UserId, modelObj.alliance.Name)
	}
	logger.InfoWithSprintf("[allianceManager] heartbeat dissolve success allianceId:%d", modelObj.alliance.AllianceId)
}

func (m *AllianceManager) tryAutoTransferLeaderForHeartbeat(modelObj *AllianceModel, basicInfos map[int64]*logicCommon.PlayerRedisInfo, nowMilli int64) (int64, int64, error) {
	if modelObj == nil || modelObj.alliance == nil || len(modelObj.members) == 0 {
		return 0, 0, nil
	}

	leaderID := int64(0)
	for userID, member := range modelObj.members {
		if member == nil {
			continue
		}
		if member.Role == int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER) {
			leaderID = userID
			break
		}
	}

	if leaderID > 0 {
		playerBasicInfo := basicInfos[leaderID]
		if playerBasicInfo == nil {
			return 0, 0, fmt.Errorf("[allianceManager] get player basic info failed leaderId:%d", leaderID)
		}
		leaderOffline := calcOfflineDurationForHeartbeat(nowMilli, playerBasicInfo.BasicInfo.LastLoginTime, playerBasicInfo.BasicInfo.LastOfflineTime)
		if leaderOffline <= enum.AllianceLeaderTransferCoLeaderMaxOffline {
			return 0, 0, nil
		}
	}

	targetID := int64(0)
	bestLastLogin := int64(-1)
	for userID, member := range modelObj.members {
		if member == nil || userID == leaderID {
			continue
		}
		if member.Role != int32(pb.ALLIANCE_POSITION_ALLIANCE_COLEADER) {
			continue
		}
		playerBasicInfo := basicInfos[userID]
		if playerBasicInfo == nil {
			logger.ErrorBySprintf("[allianceManager] get player basic info failed userId:%d", userID)
			continue
		}
		offline := calcOfflineDurationForHeartbeat(nowMilli, playerBasicInfo.BasicInfo.LastLoginTime, playerBasicInfo.BasicInfo.LastOfflineTime)
		if offline > enum.AllianceLeaderTransferCoLeaderMaxOffline {
			continue
		}
		if playerBasicInfo.BasicInfo.LastOfflineTime > bestLastLogin || (playerBasicInfo.BasicInfo.LastOfflineTime == bestLastLogin && (targetID == 0 || userID < targetID)) {
			targetID = userID
			bestLastLogin = playerBasicInfo.BasicInfo.LastOfflineTime
		}
	}

	if targetID == 0 {
		bestLastLogin = -1
		for userID, member := range modelObj.members {
			if member == nil || userID == leaderID {
				continue
			}
			playerBasicInfo := basicInfos[userID]
			if playerBasicInfo == nil {
				logger.ErrorBySprintf("[allianceManager] get player basic info failed userId:%d", userID)
				continue
			}
			offline := calcOfflineDurationForHeartbeat(nowMilli, playerBasicInfo.BasicInfo.LastLoginTime, playerBasicInfo.BasicInfo.LastOfflineTime)
			if offline >= enum.AllianceLeaderTransferMemberMaxOffline {
				continue
			}
			if playerBasicInfo.BasicInfo.LastOfflineTime > bestLastLogin || (playerBasicInfo.BasicInfo.LastOfflineTime == bestLastLogin && (targetID == 0 || userID < targetID)) {
				targetID = userID
				bestLastLogin = playerBasicInfo.BasicInfo.LastOfflineTime
			}
		}
	}

	if targetID <= 0 || targetID == leaderID {
		return 0, 0, nil
	}
	if err := m.transferLeaderForHeartbeat(modelObj, leaderID, targetID); err != nil {
		return 0, 0, err
	}
	return leaderID, targetID, nil
}

func (m *AllianceManager) transferLeaderForHeartbeat(modelObj *AllianceModel, oldLeaderID, newLeaderID int64) error {
	if modelObj == nil || modelObj.alliance == nil || newLeaderID <= 0 {
		return gorm.ErrInvalidData
	}

	allianceID := modelObj.alliance.AllianceId
	err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
		if oldLeaderID > 0 && oldLeaderID != newLeaderID {
			if err := tx.Model(&model.AllianceMemberEntity{}).
				Where("alliance_id = ? AND user_id = ?", allianceID, oldLeaderID).
				Update("role", int32(pb.ALLIANCE_POSITION_ALLIANCE_COMMON_MEMBER)).Error; err != nil {
				return err
			}
		}

		updateRes := tx.Model(&model.AllianceMemberEntity{}).
			Where("alliance_id = ? AND user_id = ?", allianceID, newLeaderID).
			Update("role", int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER))
		if updateRes.Error != nil {
			return updateRes.Error
		}
		if updateRes.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		return err
	}

	if oldLeaderID > 0 && oldLeaderID != newLeaderID {
		if oldMember := modelObj.members[oldLeaderID]; oldMember != nil {
			oldMember.Role = int32(pb.ALLIANCE_POSITION_ALLIANCE_COMMON_MEMBER)
		}
	}
	if newLeader := modelObj.members[newLeaderID]; newLeader != nil {
		newLeader.Role = int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER)
	}
	modelObj.refreshLeaderNameCache()
	return nil
}

func (m *AllianceManager) shouldAutoDissolveByOfflineForHeartbeat(basicInfos map[int64]*logicCommon.PlayerRedisInfo, nowMilli int64) bool {
	if len(basicInfos) == 0 {
		return false
	}
	for _, info := range basicInfos {
		offline := calcOfflineDurationForHeartbeat(nowMilli, info.BasicInfo.LastLoginTime, info.BasicInfo.LastOfflineTime)
		if offline <= enum.AllianceAutoDissolveOfflineThreshold {
			return false
		}
	}
	return true
}

func calcOfflineDurationForHeartbeat(nowMilli, lastLoginMilli, lastOfflineTilli int64) int64 {
	if lastLoginMilli > lastOfflineTilli {
		return 0
	}
	if lastOfflineTilli > nowMilli {
		return 0
	}
	return nowMilli - lastLoginMilli
}

func (m *AllianceManager) setNameIndex(serverID int32, name string, allianceID int64) {
	n := name
	if n == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	serverMap := m.nameIndex[serverID]
	if serverMap == nil {
		serverMap = make(map[string]int64)
		m.nameIndex[serverID] = serverMap
	}
	serverMap[n] = allianceID
}

func copyUpdates(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return map[string]interface{}{}
	}
	dst := make(map[string]interface{}, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func (m *AllianceManager) dissolveAlliance(alliance *AllianceModel) error {
	if alliance == nil || alliance.alliance.AllianceId <= 0 {
		return gorm.ErrInvalidData
	}
	allianceID := alliance.alliance.AllianceId
	err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("alliance_id = ?", allianceID).Delete(&model.AllianceMemberEntity{}).Error; err != nil {
			return err
		}
		return tx.Where("alliance_id = ?", allianceID).Delete(&model.AllianceEntity{}).Error
	})
	if err != nil {
		return err
	}
	m.removeAlliance(alliance)
	return nil
}

func (m *AllianceManager) removeAlliance(alliance *AllianceModel) {
	if alliance == nil {
		return
	}

	m.mu.Lock()
	delete(m.allianceInfos, alliance.alliance.AllianceId)
	serverMap := m.nameIndex[alliance.alliance.ServerId]
	name := alliance.alliance.Name
	if serverMap != nil && name != "" {
		if allianceID, ok := serverMap[name]; ok && allianceID == alliance.alliance.AllianceId {
			delete(serverMap, name)
		}
		if len(serverMap) == 0 {
			delete(m.nameIndex, alliance.alliance.ServerId)
		}
	}
	m.mu.Unlock()

	removeAllianceBasicFromRedis(alliance.alliance.AllianceId, alliance.alliance.ServerId)
	removeAllianceNameIndexFromRedis(alliance.alliance.ServerId, alliance.alliance.Name)
	removeAllianceApplyListFromRedis(alliance.alliance.AllianceId)
	removeAllianceMemberInfoFromRedis(alliance)
}
