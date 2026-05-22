// File: equipmentServer.go
// Description: 装备系统服务实现
// Author: 木村凉太
// Create Time: 2025.11

package equipment

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

var EquipmentIdGenerator *tool.IdGenerator

func InitEquipment() {
	EquipmentIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_EQUIPMENT))
}

var _ logicCommon.EquipmentInterface = (*EquipmentServer)(nil)

// AttributeAffix 属性词条
type AttributeAffix struct {
	AffixID   int32 `json:"affixId"`   // 词条ID
	AttrID    int32 `json:"attrId"`    // 属性ID
	StatValue int32 `json:"statValue"` // 属性值
}

// SkillAffix 技能词条
type SkillAffix struct {
	SkillAffixID int32 `json:"skillAffixId"` // 技能词条ID
	SkillID      int32 `json:"skillId"`      // 技能ID
	SkillLevel   int32 `json:"skillLevel"`   // 技能等级
}

type EquipmentServer struct {
	sessionManager logicCommon.SessionManagerInterface
	messageSender  logicCommon.MessageSenderInterface
}

func NewEquipmentServer(sessionManager logicCommon.SessionManagerInterface, messageSender logicCommon.MessageSenderInterface) *EquipmentServer {
	return &EquipmentServer{
		sessionManager: sessionManager,
		messageSender:  messageSender,
	}
}

// AddEquipment 添加装备（掉落生成）
func (s *EquipmentServer) AddEquipment(userId int64, equipmentID int32, level int32) (int64, error) {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return 0, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return 0, errors.New("player not found")
	}

	// 获取装备配置（通过equipmentId）
	cfg := gameConfig.GetEquipmentBaseCfgByEquipmentID(equipmentID)
	if cfg == nil {
		return 0, fmt.Errorf("equipment config not found: %d", equipmentID)
	}

	// 生成装备唯一ID
	equipmentOwnID := EquipmentIdGenerator.NextId()

	// 获取品质配置，确定词条数量
	qualityCfg := gameConfig.GetEquipmentQualityCfg(cfg.EquipmentQuality)
	attrEntryNum := int32(0)
	if qualityCfg != nil {
		attrEntryNum = qualityCfg.AttrEntryNum
	}

	// 生成属性词条
	attributeAffix := s.generateAttributeAffix(cfg, attrEntryNum, level)
	attributeAffixJSON, _ := json.Marshal(attributeAffix)

	// 生成技能词条
	skillAffix := s.generateSkillAffix(cfg)
	skillAffixJSON, _ := json.Marshal(skillAffix)

	// 创建装备实体
	equipment := &model.EquipmentEntity{
		EquipmentOwnID: equipmentOwnID,
		UserID:         userId,
		EquipmentID:    equipmentID,
		HeroOwnID:      0, // 初始在仓库
		SlotType:       cfg.Type,
		SlotIndex:      cfg.EquipmentSlot,
		Level:          level, // 默认等级1
		StarLevel:      0,
		ForgeLevel:     0,
		AttributeAffix: string(attributeAffixJSON),
		SkillAffix:     string(skillAffixJSON),
		SetID:          cfg.SetID,
		IsLocked:       false,
		IsDeleted:      false,
		StrongLevel:    0,
	}

	// 保存到数据库
	err := easyDB.CreatePlayerEntity[model.EquipmentEntity](equipment)
	if err != nil {
		return 0, err
	}

	// 添加到玩家模型
	if player.EquipmentModel == nil {
		equipmentModel, err := model.CreateEquipmentModel(userId, player)
		if err != nil {
			return 0, err
		}
		player.EquipmentModel = equipmentModel
	}
	player.EquipmentModel.AddEquipment(equipment)

	info := pb.EquipmentDetailInfo{
		EquipmentOwnId: equipmentOwnID,
		EquipmentId:    equipmentID,
		HeroOwnId:      0,
		Level:          level,
		SlotType:       cfg.Type,
		SlotIndex:      cfg.EquipmentSlot,
		StarLevel:      0,
		SetId:          cfg.SetID,
		IsLocked:       false,
		BaseStats:      s.getBaseStats(equipmentID, level, equipment.StrongLevel),
		AttributeAffix: s.convertAffixToPB(attributeAffix),
		SkillAffix:     s.convertSkillAffixToPB(skillAffix),
		SetInfo:        s.calculateSetEffect(player, cfg.SetID, 0),
		Power:          s.equipmentScore(userId, equipment),
		StrongLevel:    equipment.StrongLevel,
	}
	player.EquipmentModel.AddPushEquipInfoForMemory(&info)
	//s.messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_EQUIPMENT_DETAIL, &pb.PushEquipmentDetail{
	//	Info: &info})

	return equipmentOwnID, nil
}

// generateAttributeAffix 生成属性词条
func (s *EquipmentServer) generateAttributeAffix(cfg *gameConfig.EquipmentBaseCfg, attrEntryNum int32, level int32) []AttributeAffix {
	if attrEntryNum <= 0 || len(cfg.AttrEntryWeight) == 0 {
		return nil
	}

	var affixes []AttributeAffix
	usedAffixIDs := make(map[int32]bool)

	// 根据权重选择词条
	for i := int32(0); i < attrEntryNum; i++ {
		// 从权重映射中随机选择词条品质
		affixQualityID := s.selectAffixByWeightMap(cfg.AttrEntryWeight)
		if affixQualityID <= 0 {
			continue
		}

		// 根据词条品质随机选择词条ID
		affixCfg := gameConfig.GetRandomEquipmentAffixCfgByQuality(affixQualityID, usedAffixIDs)
		if affixCfg == nil {
			continue
		}

		// 从属性范围中随机选择值
		statValue := s.selectValueFromRanges(affixCfg.AttributeRanges, level)

		affix := AttributeAffix{
			AffixID:   affixCfg.AttrEntryID,
			AttrID:    affixCfg.AttrID,
			StatValue: statValue,
		}
		affixes = append(affixes, affix)
		usedAffixIDs[affixCfg.AttrEntryID] = true
	}

	return affixes
}

// generateSkillAffix 生成技能词条
func (s *EquipmentServer) generateSkillAffix(cfg *gameConfig.EquipmentBaseCfg) []SkillAffix {
	var res []SkillAffix
	// 暂时只有一个技能
	res = append(res, SkillAffix{
		SkillAffixID: cfg.SkillID,
		SkillID:      0,
		SkillLevel:   0,
	})

	return res
}

// selectAffixByWeightMap 根据权重映射选择词条
func (s *EquipmentServer) selectAffixByWeightMap(weightMap map[int32]int32) int32 {
	var totalWeight int32
	for _, w := range weightMap {
		totalWeight += w
	}
	if totalWeight <= 0 {
		return 0
	}
	randValue := tool.RandInt32(1, totalWeight)
	var currentWeight int32
	for affixID, w := range weightMap {
		currentWeight += w
		if randValue <= currentWeight {
			return affixID
		}
	}
	return 0
}

// selectValueFromRanges 根据等级从属性范围规则中选择值
// 找到等级落在的区间，从对应的属性值范围中随机
func (s *EquipmentServer) selectValueFromRanges(rules []gameConfig.AttributeRangeRule, level int32) int32 {
	if len(rules) == 0 {
		return 0
	}

	// 查找等级落在的区间
	for _, rule := range rules {
		minLevel := rule.LevelRange[0]
		maxLevel := rule.LevelRange[1]

		// 判断等级是否在这个区间内
		if level >= minLevel && level <= maxLevel {
			minValue := rule.ValueRange[0]
			maxValue := rule.ValueRange[1]

			if maxValue < minValue {
				return minValue
			}

			// 从属性值范围中随机选择
			return tool.RandInt32(minValue, maxValue)
		}
	}

	// 如果没有找到匹配的区间，使用第一个规则（兜底）
	if len(rules) > 0 {
		rule := rules[0]
		minValue := rule.ValueRange[0]
		maxValue := rule.ValueRange[1]
		if maxValue < minValue {
			return minValue
		}
		return tool.RandInt32(minValue, maxValue)
	}

	return 0
}

// EquipEquipment 穿戴装备
func (s *EquipmentServer) EquipEquipment(userId int64, equipmentOwnID int64, heroOwnID int64) error {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return errors.New("equipment model not loaded")
	}

	equipment := player.EquipmentModel.GetEquipment(equipmentOwnID)
	if equipment == nil {
		return errors.New("equipment not found")
	}

	if equipment.HeroOwnID != 0 {
		return errors.New("equipment already equipped")
	}

	// 检查英雄是否存在
	if player.HeroDetailsModel == nil || player.HeroDetailsModel.Entities == nil {
		return errors.New("hero model not loaded")
	}

	hero := player.HeroDetailsModel.GetHero(heroOwnID)
	if hero == nil {
		return errors.New("hero not found")
	}

	// 检查装备配置
	cfg := gameConfig.GetEquipmentBaseCfgByEquipmentID(equipment.EquipmentID)
	if cfg == nil {
		return errors.New("equipment config not found")
	}

	// 职业匹配逻辑
	heroClassCfg := gameConfig.GetHeroClassCfg(hero.EvolutionPath)
	if heroClassCfg == nil {
		return errors.New("heroClassCfg config not found")
	}

	if !s.isEquipmentClassAllowed(cfg, heroClassCfg) {
		return errors.New("Equipment and class mismatch")
	}

	// 检查是否已有装备在该槽位
	oldEquipment := s.findEquipmentInSlot(player, heroOwnID, equipment.SlotIndex)
	if oldEquipment != nil {
		return errors.New("slot already occupied")
	}

	// 穿戴装备
	player.EquipmentModel.UpdateHeroOwnID(equipmentOwnID, heroOwnID)

	// 更新英雄的装备ID
	player.HeroDetailsModel.UpdateEquipmentId(heroOwnID, equipmentOwnID, cfg.EquipmentSlot)
	return nil
}

// UnequipEquipment 卸下装备
func (s *EquipmentServer) UnequipEquipment(userId int64, equipmentOwnID int64) error {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return errors.New("equipment model not loaded")
	}

	equipment := player.EquipmentModel.GetEquipment(equipmentOwnID)
	if equipment == nil {
		return errors.New("equipment not found")
	}

	if equipment.HeroOwnID == 0 {
		return errors.New("equipment not equipped")
	}

	//获取装备配置
	cfg := gameConfig.GetEquipmentBaseCfgByEquipmentID(equipment.EquipmentID)
	if cfg == nil {
		return errors.New("equipment config not found")
	}

	heroOwnID := equipment.HeroOwnID

	// 卸下装备
	player.EquipmentModel.UpdateHeroOwnID(equipmentOwnID, 0)

	// 更新英雄的装备ID
	player.HeroDetailsModel.UpdateEquipmentId(heroOwnID, 0, cfg.EquipmentSlot)

	return nil
}

// SwapEquipment 替换装备
func (s *EquipmentServer) SwapEquipment(userId int64, equipmentOwnID int64, heroOwnID int64) (int64, error) {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return 0, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return 0, errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return 0, errors.New("equipment model not loaded")
	}

	newEquipment := player.EquipmentModel.GetEquipment(equipmentOwnID)
	if newEquipment == nil {
		return 0, errors.New("equipment not found")
	}

	// 查找该槽位上的旧装备
	oldEquipment := s.findEquipmentInSlot(player, heroOwnID, newEquipment.SlotIndex)
	var oldEquipmentOwnID int64 = 0

	if oldEquipment != nil {
		oldEquipmentOwnID = oldEquipment.EquipmentOwnID
		// 先卸下旧装备
		if err := s.UnequipEquipment(userId, oldEquipmentOwnID); err != nil {
			return 0, err
		}
	}

	// 穿戴新装备
	if err := s.EquipEquipment(userId, equipmentOwnID, heroOwnID); err != nil {
		// 如果失败，尝试恢复旧装备
		if oldEquipmentOwnID > 0 {
			_ = s.EquipEquipment(userId, oldEquipmentOwnID, heroOwnID)
		}
		return 0, err
	}

	return oldEquipmentOwnID, nil
}

// QuickEquip 一键穿戴
func (s *EquipmentServer) QuickEquip(userId int64, heroOwnIDs []int64) ([]*pb.EquipmentInfo, error) {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return nil, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return nil, errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return nil, errors.New("equipment model not loaded")
	}

	if player.HeroDetailsModel == nil || player.HeroDetailsModel.Entities == nil {
		return nil, errors.New("hero model not loaded")
	}

	var updatedEquipments []*pb.EquipmentInfo

	for _, heroOwnID := range heroOwnIDs {
		hero := player.HeroDetailsModel.GetHero(heroOwnID)
		if hero == nil {
			continue
		}

		heroCfg := gameConfig.GetHeroBaseCfg(int32(hero.HeroID))
		if heroCfg == nil {
			continue
		}

		bestPerSlot := make(map[int32]*model.EquipmentEntity)

		for _, equipment := range player.EquipmentModel.Entities {
			if equipment == nil || equipment.IsDeleted || equipment.HeroOwnID != 0 {
				continue
			}

			cfg := gameConfig.GetEquipmentBaseCfgByEquipmentID(equipment.EquipmentID)
			if cfg == nil {
				continue
			}

			// 职业匹配逻辑
			heroClassCfg := gameConfig.GetHeroClassCfg(hero.EvolutionPath)
			if heroClassCfg == nil {
				continue
			}

			if !s.isEquipmentClassAllowed(cfg, heroClassCfg) {
				continue
			}

			slotIndex := equipment.SlotIndex
			currentBest := bestPerSlot[slotIndex]
			if currentBest == nil || s.equipmentScore(userId, equipment) > s.equipmentScore(userId, currentBest) {
				bestPerSlot[slotIndex] = equipment
			}
		}

		for slotIndex, bestEquipment := range bestPerSlot {
			if bestEquipment == nil {
				continue
			}

			currentEquipment := s.findEquipmentInSlot(player, heroOwnID, slotIndex)
			if currentEquipment != nil {
				if s.equipmentScore(userId, currentEquipment) >= s.equipmentScore(userId, bestEquipment) {
					continue
				}
				if _, err := s.SwapEquipment(userId, bestEquipment.EquipmentOwnID, heroOwnID); err != nil {
					return nil, err
				}
			} else {
				if err := s.EquipEquipment(userId, bestEquipment.EquipmentOwnID, heroOwnID); err != nil {
					return nil, err
				}
			}

			info := s.convertToEquipmentInfo(userId, bestEquipment)
			if info != nil {
				updatedEquipments = append(updatedEquipments, info)
			}
		}
	}

	return updatedEquipments, nil
}

// QuickUnequip 一键卸下
func (s *EquipmentServer) QuickUnequip(userId int64, heroOwnIDs []int64) error {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return errors.New("equipment model not loaded")
	}

	heroSet := make(map[int64]struct{}, len(heroOwnIDs))
	for _, heroOwnID := range heroOwnIDs {
		heroSet[heroOwnID] = struct{}{}
	}

	for _, equipment := range player.EquipmentModel.Entities {
		if equipment == nil || equipment.IsDeleted {
			continue
		}
		if equipment.HeroOwnID == 0 {
			continue
		}
		if _, ok := heroSet[equipment.HeroOwnID]; !ok {
			continue
		}
		if err := s.UnequipEquipment(userId, equipment.EquipmentOwnID); err != nil {
			return err
		}
	}

	return nil
}

// findEquipmentInSlot 查找槽位上的装备
func (s *EquipmentServer) findEquipmentInSlot(player *model.PlayerModel, heroOwnID int64, slotIndex int32) *model.EquipmentEntity {
	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return nil
	}

	for _, equipment := range player.EquipmentModel.Entities {
		if equipment == nil {
			continue
		}
		if equipment.HeroOwnID == heroOwnID && equipment.SlotIndex == slotIndex {
			return equipment
		}
	}

	return nil
}

// LockEquipment 锁定装备
func (s *EquipmentServer) LockEquipment(userId int64, equipmentOwnID int64) error {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return errors.New("equipment model not loaded")
	}

	equipment := player.EquipmentModel.GetEquipment(equipmentOwnID)
	if equipment == nil {
		return errors.New("equipment not found")
	}

	player.EquipmentModel.UpdateIsLocked(equipmentOwnID, true)
	return nil
}

// UnlockEquipment 解锁装备
func (s *EquipmentServer) UnlockEquipment(userId int64, equipmentOwnID int64) error {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return errors.New("equipment model not loaded")
	}

	equipment := player.EquipmentModel.GetEquipment(equipmentOwnID)
	if equipment == nil {
		return errors.New("equipment not found")
	}

	player.EquipmentModel.UpdateIsLocked(equipmentOwnID, false)
	return nil
}

// DecomposeEquipment 分解装备
func (s *EquipmentServer) DecomposeEquipment(userId int64, equipmentOwnID int64) (*pb.EquipmentDecomposeResp, error) {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return nil, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return nil, errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return nil, errors.New("equipment model not loaded")
	}

	equipment := player.EquipmentModel.GetEquipment(equipmentOwnID)
	if equipment == nil {
		return nil, errors.New("equipment not found")
	}

	if equipment.IsLocked {
		return nil, errors.New("equipment is locked")
	}

	if equipment.HeroOwnID != 0 {
		return nil, errors.New("equipment is equipped")
	}

	// TODO: 根据装备品质和等级计算分解产物
	baseCfg := gameConfig.GetEquipmentBaseCfgByEquipmentID(equipment.EquipmentID)
	if baseCfg == nil {
		return nil, errors.New("equipment config not found")
	}

	cfg := gameConfig.GetEquipmentQualityCfg(baseCfg.EquipmentQuality)
	if cfg == nil {
		return nil, errors.New("equipment quality config not found")
	}

	// 这里简化处理，返回固定的分解产物
	resp := &pb.EquipmentDecomposeResp{
		DropId: []int32{
			cfg.Item1,
			cfg.Item2,
		},
	}

	// 标记装备为已删除
	player.EquipmentModel.UpdateIsDeleted(equipmentOwnID, true)

	return resp, nil
}

// DecomposeEquipments 批量分解装备
func (s *EquipmentServer) DecomposeEquipments(userId int64, equipmentOwnIDs []int64) (*pb.EquipmentDecomposeResp, error) {
	if len(equipmentOwnIDs) == 0 {
		return &pb.EquipmentDecomposeResp{}, nil
	}

	totalResp := &pb.EquipmentDecomposeResp{
		DropId: make([]int32, 0),
	}

	for _, equipmentOwnID := range equipmentOwnIDs {
		resp, err := s.DecomposeEquipment(userId, equipmentOwnID)
		if err != nil {
			return nil, err
		}

		totalResp.DropId = append(totalResp.DropId, resp.DropId...)
	}

	return totalResp, nil
}

// GetEquipmentList 获取装备列表
func (s *EquipmentServer) GetEquipmentList(userId int64) ([]*pb.EquipmentInfo, error) {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return nil, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return nil, errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return nil, errors.New("equipment model not loaded")
	}

	var equipmentList []*pb.EquipmentInfo
	for _, equipment := range player.EquipmentModel.Entities {
		if equipment == nil || equipment.IsDeleted {
			continue
		}

		info := s.convertToEquipmentInfo(userId, equipment)
		equipmentList = append(equipmentList, info)
	}

	return equipmentList, nil
}

// GetEquipmentDetail 获取装备详情
func (s *EquipmentServer) GetEquipmentDetail(userId int64, equipmentOwnID int64) (*pb.EquipmentDetailInfo, error) {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return nil, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return nil, errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return nil, errors.New("equipment model not loaded")
	}

	equipment := player.EquipmentModel.GetEquipment(equipmentOwnID)
	if equipment == nil {
		return nil, errors.New("equipment not found")
	}

	// 获取装备配置
	cfg := gameConfig.GetEquipmentBaseCfgByEquipmentID(equipment.EquipmentID)
	if cfg == nil {
		return nil, errors.New("equipment config not found")
	}

	// 解析词条
	var attributeAffix []AttributeAffix
	json.Unmarshal([]byte(equipment.AttributeAffix), &attributeAffix)

	var skillAffix []SkillAffix
	json.Unmarshal([]byte(equipment.SkillAffix), &skillAffix)

	// 计算套装效果
	setInfo := s.calculateSetEffect(player, equipment.SetID, equipment.HeroOwnID)

	detail := &pb.EquipmentDetailInfo{
		EquipmentOwnId: equipmentOwnID,
		EquipmentId:    equipment.EquipmentID,
		HeroOwnId:      equipment.HeroOwnID,
		SlotType:       equipment.SlotType,
		SlotIndex:      equipment.SlotIndex,
		Level:          equipment.Level,
		StarLevel:      equipment.StarLevel,
		ForgeLevel:     equipment.ForgeLevel,
		SetId:          equipment.SetID,
		IsLocked:       equipment.IsLocked,
		BaseStats:      s.getBaseStats(equipment.EquipmentID, equipment.Level, equipment.StrongLevel),
		AttributeAffix: s.convertAffixToPB(attributeAffix),
		SkillAffix:     s.convertSkillAffixToPB(skillAffix),
		SetInfo:        setInfo,
		Power:          s.equipmentScore(userId, equipment),
		StrongLevel:    equipment.StrongLevel,
	}

	return detail, nil
}

// convertToEquipmentInfo 转换为装备信息
func (s *EquipmentServer) convertToEquipmentInfo(userId int64, equipment *model.EquipmentEntity) *pb.EquipmentInfo {
	cfg := gameConfig.GetEquipmentBaseCfgByEquipmentID(equipment.EquipmentID)
	if cfg == nil {
		return nil
	}

	return &pb.EquipmentInfo{
		EquipmentOwnId: equipment.EquipmentOwnID,
		EquipmentId:    equipment.EquipmentID,
		HeroOwnId:      equipment.HeroOwnID,
		SlotType:       equipment.SlotType,
		SlotIndex:      equipment.SlotIndex,
		Level:          equipment.Level,
		StarLevel:      equipment.StarLevel,
		SetId:          equipment.SetID,
		IsLocked:       equipment.IsLocked,
		Power:          s.equipmentScore(userId, equipment),
		StrongLevel:    equipment.StrongLevel,
	}
}

// getBaseStats 获取装备基础属性（从等级属性配置中获取）
func (s *EquipmentServer) getBaseStats(equipmentID int32, level int32, strongLevel int32) map[int32]int32 {
	result := make(map[int32]int32)

	// 从装备等级属性配置中获取
	levelAttrCfg := gameConfig.GetEquipmentLevelAttrCfg(equipmentID, level)
	if levelAttrCfg != nil {
		for _, attr := range levelAttrCfg.Attributes {
			result[attr.AttrID] = attr.Value
		}
	}

	equipmentCfg := gameConfig.GetEquipmentBaseCfg(equipmentID)
	if equipmentCfg == nil {
		return result
	}
	equipmentStrongAttrCfgId := model.GetEquipmentStrongId(equipmentCfg)
	equipmentStrongAttrCfg := gameConfig.GetEquipEnhanceCfg(equipmentStrongAttrCfgId)
	if equipmentStrongAttrCfg != nil {
		for id, attrID := range equipmentStrongAttrCfg.Attr {
			result[attrID] += equipmentStrongAttrCfg.AttrNum[id] * strongLevel
		}
	}

	return result
}

// convertAffixToPB 转换属性词条
func (s *EquipmentServer) convertAffixToPB(affixes []AttributeAffix) []*pb.EquipmentAffixInfo {
	var result []*pb.EquipmentAffixInfo
	for _, affix := range affixes {
		result = append(result, &pb.EquipmentAffixInfo{
			AffixId:   affix.AffixID,
			StatType:  affix.AttrID, // 使用 AttrID 作为 StatType
			StatValue: affix.StatValue,
		})
	}
	return result
}

// convertSkillAffixToPB 转换技能词条
func (s *EquipmentServer) convertSkillAffixToPB(affixes []SkillAffix) []*pb.EquipmentSkillAffixInfo {
	var result []*pb.EquipmentSkillAffixInfo
	for _, affix := range affixes {
		if affix.SkillAffixID != 0 {
			result = append(result, &pb.EquipmentSkillAffixInfo{
				SkillAffixId: affix.SkillAffixID,
				SkillId:      affix.SkillID,
				SkillLevel:   affix.SkillLevel,
			})
		}

	}
	return result
}

// calculateSetEffect 计算套装效果
func (s *EquipmentServer) calculateSetEffect(player *model.PlayerModel, setID int32, heroOwnID int64) *pb.EquipmentSetInfo {
	setInfo := &pb.EquipmentSetInfo{
		SetId:      setID,
		PieceCount: 0,
	}

	if setID <= 0 || heroOwnID == 0 {
		return nil
	}

	setCfg := gameConfig.GetEquipmentSetCfg(setID)
	if setCfg == nil {
		return nil
	}

	// 统计该英雄穿戴的该套装装备数量
	setCount := int32(0)
	for _, equipment := range player.EquipmentModel.Entities {
		if equipment == nil || equipment.IsDeleted {
			continue
		}
		if equipment.HeroOwnID == heroOwnID && equipment.SetID == setID {
			setCount++
		}
	}

	if setCount < 2 {
		return nil
	}

	setInfo.PieceCount = setCount

	// 根据件数返回对应的套装效果
	// 新的配置结构中只有技能ID，没有属性列表
	// 根据实际件数匹配 setLevels 中的件数，返回对应的技能ID
	activeStats := make(map[int32]int32)
	for i, level := range setCfg.SetLevels {
		if setCount >= level {
			// 如果达到该件数，激活对应的技能
			if i < len(setCfg.SkillIDs) && setCfg.SkillIDs[i] > 0 {
				// 技能ID作为key，技能等级作为value（这里暂时用1，实际应该从配置中获取）
				activeStats[setCfg.SkillIDs[i]] = 1
				setInfo.ActivePieceCount = level
			}
		}
	}
	setInfo.ActiveStats = activeStats

	return setInfo
}

// isEquipmentClassAllowed 判断装备职业是否匹配
func (s *EquipmentServer) isEquipmentClassAllowed(cfg *gameConfig.EquipmentBaseCfg, heroClassCfg *gameConfig.HeroClassCfg) bool {
	if cfg == nil || heroClassCfg == nil {
		return false
	}

	for _, i := range heroClassCfg.ArmorType {
		if i == cfg.Type {
			return true
		}
	}

	return false
}

// equipmentScore 装备战力
func (s *EquipmentServer) equipmentScore(userId int64, equipment *model.EquipmentEntity) int64 {
	if equipment == nil {
		return 0
	}

	var attributeAffix []AttributeAffix
	if equipment.AttributeAffix != "" {
		if err := json.Unmarshal([]byte(equipment.AttributeAffix), &attributeAffix); err != nil {
			return 0
		}
	}

	attrs := make(map[int32]int64)
	for _, affix := range attributeAffix {
		attrs[affix.AttrID] = int64(affix.StatValue)
	}

	// 1. 计算基础属性加成
	baseStats := s.getBaseStats(equipment.EquipmentID, equipment.Level, equipment.StrongLevel)
	for k, v := range baseStats {
		if _, ok := attrs[k]; ok {
			attrs[k] += int64(v)
		} else {
			attrs[k] = int64(v)
		}
	}

	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return 0
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return 0
	}

	power := gameConfig.GetAttrMapPower(player.ArchitectureModel.GetMainLevel(), attrs)

	return int64(power)
}

// CalculateHeroEquipmentAttr 计算英雄穿戴的装备对指定属性的总加成
// 返回该英雄穿戴的所有装备对指定属性的加成总和
func (s *EquipmentServer) CalculateHeroEquipmentAttr(equipmentModel *model.EquipmentCollectionModel, heroId int64, attrId int32) int64 {
	if equipmentModel == nil || equipmentModel.Entities == nil {
		return 0
	}

	var totalAttr int64 = 0

	// 遍历该英雄穿戴的所有装备
	for _, equipment := range equipmentModel.Entities {
		if equipment == nil || equipment.IsDeleted {
			continue
		}
		if equipment.HeroOwnID != heroId {
			continue
		}

		// 1. 计算基础属性加成
		baseStats := s.getBaseStats(equipment.EquipmentID, equipment.Level, equipment.StrongLevel)
		if value, ok := baseStats[attrId]; ok {
			totalAttr += int64(value)
		}

		// 2. 计算属性词条加成
		if equipment.AttributeAffix != "" {
			var attributeAffix []AttributeAffix
			if err := json.Unmarshal([]byte(equipment.AttributeAffix), &attributeAffix); err == nil {
				for _, affix := range attributeAffix {
					if affix.AttrID == attrId {
						totalAttr += int64(affix.StatValue)
					}
				}
			}
		}

		// 3. 套装效果暂时不计算属性加成（当前配置只有技能ID）
	}

	return totalAttr
}

func (s *EquipmentServer) StrongEquipment(userId int64, equipmentOwnID int64, isUseStone bool) (*pb.EquipmentStrongResp, error) {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return nil, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return nil, errors.New("player not found")
	}

	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return nil, errors.New("equipment model not loaded")
	}

	equipment := player.EquipmentModel.GetEquipment(equipmentOwnID)
	if equipment == nil {
		return nil, errors.New("equipment not found")
	}

	equipmentCfg := gameConfig.GetEquipmentBaseCfg(equipment.EquipmentID)
	if equipmentCfg == nil {
		return nil, errors.New("equipment config not found")
	}

	strongCfg := gameConfig.GetEquipEnhanceCfg(model.GetEquipmentStrongId(equipmentCfg))
	if strongCfg == nil {
		return nil, errors.New("equipment strong config not found")
	}

	if equipment.StrongLevel >= strongCfg.Level[1] {
		return nil, errors.New("equipment strong level is max")
	}

	costCfg := gameConfig.GetEnhanceCostCfg(equipment.StrongLevel/10 + 1)
	if costCfg == nil {
		return nil, errors.New("equipment strong cost config not found")
	}
	flag, err := itemService.CheckItemsCount(player, costCfg.EnhanceCost)
	if !flag || err != nil {
		return nil, errors.New("item count is not enough")
	}
	if isUseStone {
		flag, err = itemService.CheckItemsCount(player, costCfg.SuccessCost)
		if !flag || err != nil {
			return nil, errors.New("item count is not enough")
		}
	}
	successRate := costCfg.SuccessRate
	if isUseStone {
		successRate = successRate + gameConfig.GetConstantCfg(gameConfig.CONSTANT_successChanceIncrease).Value[0]
	}
	isSuccess := gameConfig.WeightedRandomChoice([]int32{1, 0}, []int32{successRate, 10000 - successRate})
	if isSuccess == 1 {
		player.EquipmentModel.UpdateStrongLevel(equipmentOwnID, equipment.StrongLevel+1)
	}
	if err := itemService.RemoveItems(player, costCfg.EnhanceCost, enum.ITEM_CHANGE_REASON_STRONG_EQUIPMENT); err != nil {
		return nil, errors.New("failed to remove items")
	}
	if isUseStone {
		if err := itemService.RemoveItems(player, costCfg.SuccessCost, enum.ITEM_CHANGE_REASON_STRONG_EQUIPMENT); err != nil {
			return nil, errors.New("failed to remove items")
		}
	}

	return &pb.EquipmentStrongResp{
		EquipmentDetail: &pb.EquipmentDetailInfo{
			EquipmentOwnId: equipmentOwnID,
			StrongLevel:    equipment.StrongLevel,
		},
	}, nil
}

func (s *EquipmentServer) RebirthEquipment(userId int64, equipmentOwnID int64) (*pb.EquipmentRebirthResp, error) {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return nil, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return nil, errors.New("player not found")
	}
	if player.EquipmentModel == nil || player.EquipmentModel.Entities == nil {
		return nil, errors.New("equipment model not loaded")
	}
	equipment := player.EquipmentModel.GetEquipment(equipmentOwnID)
	if equipment == nil {
		return nil, errors.New("equipment not found")
	}

	items := make([]*gameConfig.ItemConfig, 0)
	addItems := gameConfig.GetRebirthEquipItems(equipment.StrongLevel)
	if addItems == nil {
		return nil, errors.New("equipment strong items not loaded")
	}
	for key, v := range addItems {
		items = append(items, &gameConfig.ItemConfig{ID: key, Num: v})
	}
	err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_REBRITH_EQUIPMENT)
	if err != nil {
		return nil, errors.New("failed to add items")
	}
	player.EquipmentModel.UpdateStrongLevel(equipmentOwnID, 0)

	return &pb.EquipmentRebirthResp{
		EquipmentDetail: &pb.EquipmentDetailInfo{
			EquipmentOwnId: equipmentOwnID,
			StrongLevel:    equipment.StrongLevel,
		},
	}, nil
}
