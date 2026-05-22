// petService：宠物模块（规则 + 对外编排 + PB 组装），controller / 其他模块同包调用。
package pet

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/tool"
)

var (
	ErrPetNotFound                        = errors.New("pet not found")
	ErrPetDeleted                         = errors.New("pet is deleted")
	ErrPetCfgNotFound                     = errors.New("pet config not found")
	ErrPetCardCfgNotFound                 = errors.New("pet card config not found")
	ErrPetLevelMax                        = errors.New("pet level max")
	ErrPetLevelUpConditionNotMet          = errors.New("pet level up condition not met")
	ErrPetStarMax                         = errors.New("pet star max")
	ErrPetStarUpConditionNotMet           = errors.New("pet star up condition not met")
	ErrPetAffinityCfgNotFound             = errors.New("pet affinity config not found")
	ErrPetAffinityNotFound                = errors.New("pet affinity not found")
	ErrPetAffinityNotActive               = errors.New("pet affinity not active")
	ErrPetAffinityLevelMax                = errors.New("pet affinity level max")
	ErrPetAffinityActivateConditionNotMet = errors.New("pet affinity activate condition not met")
	ErrPetAffinityLevelUpConditionNotMet  = errors.New("pet affinity level up condition not met")
	ErrPetSkillCfgNotFound                = errors.New("pet skill config not found")
	ErrPetSkillNotFound                   = errors.New("pet skill not found")
	ErrPetSkillRollFailed                 = errors.New("pet skill roll failed")
	ErrPetLevelMin                        = errors.New("pet level min")
)

// PetBagMaxNum 宠物背包上限（与英雄类似，后续可接配置）。
var PetBagMaxNum = 1200

// PetIdGenerator 宠物唯一ID生成器（使用雪花算法，与英雄/装备一致）。
var PetIdGenerator *tool.IdGenerator

func InitPet() {
	PetIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_ITEM))
}

// IsPetUncultivated 判断宠物是否“未培养”，用于分解等入口校验。
// 说明：当前项目里宠物“培养”主要体现在等级、星级、被动技能与是否穿戴。
func IsPetUncultivated(p *model.PetEntity) bool {
	if p == nil || p.IsDeleted {
		return false
	}
	// 未培养：未穿戴 + 默认等级/星级 + 无被动技能
	return p.HeroOwnId == 0 && p.Level <= 1 && p.Star <= 0 && len(p.PassiveSkills) == 0
}

func addItemConfigsToMap(dst map[int32]int64, items []*gameConfig.ItemConfig, multiplier int64) {
	if dst == nil || multiplier <= 0 {
		return
	}
	for _, item := range items {
		if item == nil || item.ID <= 0 || item.Num <= 0 {
			continue
		}
		dst[item.ID] += item.Num * multiplier
	}
}

func itemMapToConfigs(itemMap map[int32]int64) []*gameConfig.ItemConfig {
	if len(itemMap) == 0 {
		return nil
	}
	ids := make([]int32, 0, len(itemMap))
	for id, num := range itemMap {
		if id <= 0 || num <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	items := make([]*gameConfig.ItemConfig, 0, len(ids))
	for _, id := range ids {
		items = append(items, &gameConfig.ItemConfig{ID: id, Num: itemMap[id]})
	}
	return items
}

func calcPetLevelRefundMap(base *gameConfig.PetBaseCfg, level int32, getLevelCfg func(int32, int32) *gameConfig.PetLevelCfg) map[int32]int64 {
	refundMap := make(map[int32]int64)
	if base == nil || getLevelCfg == nil || level <= 1 {
		return refundMap
	}

	for fromLevel := int32(1); fromLevel < level; fromLevel++ {
		lcfg := getLevelCfg(base.PetPotential, fromLevel)
		if lcfg == nil {
			continue
		}
		addItemConfigsToMap(refundMap, lcfg.Cost, 1)
	}
	return refundMap
}

func calcPetStarDecomposeParts(petID int32, star int32, getStarCfg func(int32, int32) *gameConfig.PetStarCfg) (int64, map[int32]int64, error) {
	refundMap := make(map[int32]int64)
	if star <= 0 {
		return 1, refundMap, nil
	}
	if petID <= 0 || getStarCfg == nil {
		return 0, nil, ErrPetCfgNotFound
	}

	bodyCount := int64(1)
	for fromStar := int32(0); fromStar < star; fromStar++ {
		cfg := getStarCfg(petID, fromStar)
		if cfg == nil {
			return 0, nil, ErrPetCfgNotFound
		}
		if cfg.CostNum1 > 0 {
			bodyCount += int64(cfg.CostNum1)
		}
		if cfg.CostNum2 != nil {
			addItemConfigsToMap(refundMap, []*gameConfig.ItemConfig{cfg.CostNum2}, 1)
		}
	}
	return bodyCount, refundMap, nil
}

// CalcPetDecomposeRewardMap 计算单只宠物分解时应返还的全部材料。
func CalcPetDecomposeRewardMap(p *model.PetEntity) (map[int32]int64, error) {
	if p == nil || p.IsDeleted {
		return nil, ErrPetNotFound
	}

	base := gameConfig.GetPetBaseCfg(p.PetID)
	if base == nil {
		return nil, ErrPetCfgNotFound
	}
	if len(base.SalvageYield) == 0 {
		return nil, ErrPetCfgNotFound
	}

	level := p.Level
	if level < 1 {
		level = 1
	}
	star := p.Star
	if star < 0 {
		star = 0
	}

	rewardMap := calcPetLevelRefundMap(base, level, gameConfig.GetPetLevelCfgByPotentialLevel)
	bodyCount, starRefundMap, err := calcPetStarDecomposeParts(p.PetID, star, gameConfig.GetPetStarCfgByPetIdStar)
	if err != nil {
		return nil, err
	}
	for id, num := range starRefundMap {
		rewardMap[id] += num
	}
	addItemConfigsToMap(rewardMap, base.SalvageYield, bodyCount)
	return rewardMap, nil
}

// ObtainPet 为玩家添加一只新宠物
// 规范写法：统一由该函数负责生成 PetOwnID、初始化默认字段，并调用 PetModel.AddPet。
func ObtainPet(player *model.PlayerModel, petID int32) (*model.PetEntity, error) {
	return obtainPetWithLevel(player, petID, 1)
}

func obtainPetWithLevel(player *model.PlayerModel, petID int32, level int32) (*model.PetEntity, error) {
	if player == nil || player.PetModel == nil {
		return nil, ErrPetNotFound
	}
	// 背包容量校验（简单按实体数量限制）
	if len(player.PetModel.Entities)+1 > PetBagMaxNum {
		return nil, fmt.Errorf("pet bag full")
	}
	base := gameConfig.GetPetBaseCfg(petID)
	if base == nil {
		return nil, ErrPetCfgNotFound
	}
	if level <= 0 {
		level = 1
	}

	petOwnID := PetIdGenerator.NextId()
	entity := &model.PetEntity{
		PetOwnID:      petOwnID,
		UserID:        player.GetUserId(),
		PetID:         petID,
		Level:         level,
		Star:          0,
		HeroOwnId:     0,
		PassiveSkills: nil,
		IsDeleted:     false,
	}
	if err := player.PetModel.AddPet(entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// ObtainPetByCard 按“宠物卡道具ID”发放宠物（与英雄道具自动使用路径一致）。
// 仅当 item 的 ShowGroup 为 ITEM_TYPE_PET 时生效，实际宠物ID来自 itemCfg.TargetId。
func ObtainPetByCard(player *model.PlayerModel, cardItemID int32) (*model.PetEntity, error) {
	if cardItemID <= 0 {
		return nil, ErrPetCardCfgNotFound
	}
	itemCfg := gameConfig.GetItemCfg(cardItemID)
	if itemCfg == nil || itemCfg.ShowGroup != int32(enum.ITEM_TYPE_PET) {
		return nil, ErrPetCardCfgNotFound
	}
	return obtainPetWithLevel(player, itemCfg.TargetId, itemCfg.Level)
}

var unlockService logicCommon.UnlockServiceInterface

// InitPetService 由 gameController 在 unlockService 就绪后注入（仅用于升级 UnlockId 判断）。
func InitPetService(unlock logicCommon.UnlockServiceInterface) {
	unlockService = unlock
}

func BuildPetAffinityInfo(player *model.PlayerModel, affinityId int32) *pb.PetAffinityInfo {
	if player == nil || player.PetModel == nil || player.PetAffinityModel == nil {
		return nil
	}
	cfg := gameConfig.GetPetAffinityCfg(affinityId)
	if cfg == nil {
		return nil
	}

	level := int32(0)
	if ent := player.PetAffinityModel.Entities[affinityId]; ent != nil {
		level = ent.Level
	}

	// 根据当前 level 计算缘分带来的属性（与 HeroAttr 聚合中一致）：取 AttrNum[level-1]
	attrs := make(map[int32]int64)
	if level > 0 {
		row := level - 1
		if row >= 0 && int(row) < len(cfg.AttrNum) && cfg.Attr > 0 {
			attrs[cfg.Attr] = int64(cfg.AttrNum[row])
		}
	}

	return &pb.PetAffinityInfo{
		AffinityId: affinityId,
		Level:      level,
		Attributes: attrs,
	}
}

// GetPetAffinityList 获取缘分全量信息（按 affinityId 升序返回，便于客户端稳定展示）
func GetPetAffinityList(player *model.PlayerModel) *pb.PetAffinityListResp {
	if player == nil || player.PetModel == nil || player.PetAffinityModel == nil {
		return &pb.PetAffinityListResp{Info: make([]*pb.PetAffinityInfo, 0)}
	}

	cfgMap := gameConfig.GetAllPetAffinityCfg()
	if len(cfgMap) == 0 {
		return &pb.PetAffinityListResp{Info: make([]*pb.PetAffinityInfo, 0)}
	}

	ids := make([]int32, 0, len(cfgMap))
	for affinityID := range cfgMap {
		ids = append(ids, affinityID)
	}
	slices.Sort(ids)

	info := make([]*pb.PetAffinityInfo, 0, len(ids))
	for _, affinityID := range ids {
		if v := BuildPetAffinityInfo(player, affinityID); v != nil {
			info = append(info, v)
		}
	}
	return &pb.PetAffinityListResp{Info: info}
}

// MeetAffinityRequirementAtLevel 缘分条件（按目标缘分等级档位判断）。
func MeetAffinityRequirementAtLevel(player *model.PlayerModel, aff *gameConfig.PetAffinityCfg, level int32) bool {
	if player == nil || player.PetAffinityModel == nil || aff == nil {
		return false
	}
	return player.PetAffinityModel.MeetAffinityRequirementAtLevel(aff, level)
}

// UpgradeAffinity 升级一个缘分组合：若当前 level=0，则按“1级条件”直接升到1（等价于原激活）；否则按下一档升级。
func UpgradeAffinity(player *model.PlayerModel, affinityId int32) (*pb.PetAffinityLevelUpResp, error) {
	if player == nil || player.PetModel == nil || player.PetAffinityModel == nil {
		return nil, ErrPetNotFound
	}
	ent := player.PetAffinityModel.EnsureEntity(affinityId)
	cfg := gameConfig.GetPetAffinityCfg(affinityId)
	if cfg == nil {
		return nil, ErrPetAffinityCfgNotFound
	}
	if int(ent.Level) >= len(cfg.PetStar) {
		return nil, ErrPetAffinityLevelMax
	}
	nextLevel := ent.Level + 1
	// 先满足目标档位条件才允许升级（当 ent.Level=0 时，即原激活条件）。
	if !MeetAffinityRequirementAtLevel(player, cfg, nextLevel) {
		return nil, ErrPetAffinityLevelUpConditionNotMet
	}
	player.PetAffinityModel.UpdateLevel(affinityId, nextLevel)
	return &pb.PetAffinityLevelUpResp{Info: BuildPetAffinityInfo(player, affinityId)}, nil
}

// previewPetPassiveSkillsAtStar 仅做校验与预计算，不写库。
func previewPetPassiveSkillsAtStar(player *model.PlayerModel, petOwnID int64, targetStar int32) ([]int32, error) {
	if player == nil || player.PetModel == nil {
		return nil, ErrPetNotFound
	}
	p := player.PetModel.GetPet(petOwnID)
	if p == nil {
		return nil, ErrPetNotFound
	}
	if p.IsDeleted {
		return nil, ErrPetDeleted
	}
	base := gameConfig.GetPetBaseCfg(p.PetID)
	if base == nil {
		return nil, ErrPetCfgNotFound
	}

	if targetStar < 0 {
		targetStar = 0
	}

	// 被动激活按“达到该星级立即生效”处理：
	// 配置 star=n 且 passiveSkill=1，宠物达到 n 星时即计入解锁。
	target := int32(0)
	for fromStar := int32(0); fromStar <= targetStar; fromStar++ {
		cfg := gameConfig.GetPetStarCfgByPetIdStar(p.PetID, fromStar)
		if cfg == nil {
			// 配置缺失时不允许继续（避免“少配一档导致解锁数计算错误”）
			return nil, ErrPetSkillCfgNotFound
		}
		if cfg.PassiveSkill == 1 {
			target++
		}
	}
	cur := make([]int32, 0)
	seen := make(map[int32]struct{})
	for _, sid := range p.PassiveSkills {
		if sid <= 0 {
			continue
		}
		if _, ok := seen[sid]; ok {
			continue
		}
		seen[sid] = struct{}{}
		cur = append(cur, sid)
	}

	if target <= 0 {
		return cur, nil
	}
	// 仅在确实需要解锁/补齐被动技能时，才要求被动技能池配置完整。
	if len(base.PassiveSkillGroup) == 0 || len(base.SkillGroupWeight) == 0 {
		return nil, ErrPetSkillCfgNotFound
	}

	// 强一致：被动技能数量必须等于目标解锁数量
	if int32(len(cur)) > target {
		cur = cur[:target]
		return cur, nil
	}
	if int32(len(cur)) == target {
		return cur, nil
	}

	for int32(len(cur)) < target {
		skillId, err := rollOnePassiveSkill(base, seen)
		if err != nil {
			return nil, err
		}
		seen[skillId] = struct{}{}
		cur = append(cur, skillId)
	}

	return cur, nil
}

// EnsurePetPassiveSkills 兼容旧调用：按当前星级预计算并落库。
func EnsurePetPassiveSkills(player *model.PlayerModel, petOwnID int64) error {
	if player == nil || player.PetModel == nil {
		return ErrPetNotFound
	}
	p := player.PetModel.GetPet(petOwnID)
	if p == nil {
		return ErrPetNotFound
	}
	skills, err := previewPetPassiveSkillsAtStar(player, petOwnID, p.Star)
	if err != nil {
		return err
	}
	player.PetModel.UpdatePassiveSkills(petOwnID, tool.JSONInt32Slice(skills))
	return nil
}

// PreviewPetPassiveSkillsAfterStarUp 预计算升星后的被动技能（不落库）。
func PreviewPetPassiveSkillsAfterStarUp(player *model.PlayerModel, petOwnID int64, newStar int32) ([]int32, error) {
	return previewPetPassiveSkillsAtStar(player, petOwnID, newStar)
}

// 随机被动技能：roll宠物被动技能
func rollOnePassiveSkill(base *gameConfig.PetBaseCfg, seen map[int32]struct{}) (int32, error) {
	if base == nil {
		return 0, ErrPetCfgNotFound
	}
	if len(base.PassiveSkillGroup) == 0 || len(base.SkillGroupWeight) == 0 {
		return 0, ErrPetSkillCfgNotFound
	}
	for i := 0; i < 20; i++ {
		group := gameConfig.WeightedRandomChoice(base.PassiveSkillGroup, base.SkillGroupWeight)
		ps := gameConfig.GetPetPassiveSkillGroupCfg(group)
		if ps == nil || len(ps.Skill) == 0 || len(ps.SkillWeight) == 0 {
			continue
		}
		sid := gameConfig.WeightedRandomChoice(ps.Skill, ps.SkillWeight)
		if sid <= 0 {
			continue
		}
		if _, ok := seen[sid]; ok {
			continue
		}
		return sid, nil
	}
	return 0, ErrPetSkillRollFailed
}

// LevelUp 宠物升级：完成校验与目标等级计算，并汇总升级所需的道具消耗
func LevelUp(player *model.PlayerModel, petOwnID int64, deltaLevel int32) (targetLevel int32, costMap map[int32]int64, err error) {
	if player == nil || player.PetModel == nil {
		return 0, nil, ErrPetNotFound
	}
	p := player.PetModel.GetPet(petOwnID)
	if p == nil {
		return 0, nil, ErrPetNotFound
	}
	if deltaLevel <= 0 {
		return p.Level, nil, nil
	}

	base := gameConfig.GetPetBaseCfg(p.PetID)
	if base == nil {
		return 0, nil, ErrPetCfgNotFound
	}

	target := max(p.Level+deltaLevel, 1)
	cur := p.Level

	// 累加每一级的消耗，使用 map[itemId]num 方便与道具服务对接。
	// level 配置语义：level=n 表示 n->n+1 的升级配置。
	costMap = make(map[int32]int64)
	realTarget := cur

	for fromLevel := cur; fromLevel < target; fromLevel++ {
		stepCfg := gameConfig.GetPetLevelCfgByPotentialLevel(base.PetPotential, fromLevel)
		if !gameConfig.HasPetLevelUpgradeStep(stepCfg) {
			break
		}
		if unlockService != nil && stepCfg.UnlockId != 0 && !unlockService.CheckUnlock(stepCfg.UnlockId, player) {
			break
		}
		for _, cfgItem := range stepCfg.Cost {
			if cfgItem == nil || cfgItem.ID == 0 || cfgItem.Num <= 0 {
				continue
			}
			costMap[cfgItem.ID] += cfgItem.Num
		}
		realTarget = fromLevel + 1
	}

	// 若 realTarget 未超过当前等级，说明无法升级
	if realTarget <= cur {
		// 区分“已满级”与“条件未满足”
		if !gameConfig.HasPetLevelUpgradeStep(gameConfig.GetPetLevelCfgByPotentialLevel(base.PetPotential, cur)) {
			return cur, nil, ErrPetLevelMax
		}
		return cur, nil, ErrPetLevelUpConditionNotMet
	}

	return realTarget, costMap, nil
}

// 宠物升星，根据配置确认下一星档位是否存在
func StarUp(player *model.PlayerModel, petOwnID int64, materialsPetOwnIds []int64) (newStar int32, sacrificePetOwnIds []int64, err error) {
	if player == nil || player.PetModel == nil {
		return 0, nil, ErrPetNotFound
	}
	p := player.PetModel.GetPet(petOwnID)
	if p == nil {
		return 0, nil, ErrPetNotFound
	}

	nextStar := p.Star + 1
	cfg := gameConfig.GetPetStarCfgByPetIdStar(p.PetID, p.Star)

	// star 配置语义：star=n 表示 n->n+1。
	// 若当前星级档已无升级增量（策划用“空档”作为终点标记），视为满星。
	if !gameConfig.HasPetStarUpgradeStep(cfg) {
		return 0, nil, ErrPetStarMax
	}

	// 需要消耗的材料宠物数量
	requiredCount := max(cfg.CostNum1, 0)

	if int32(len(materialsPetOwnIds)) != requiredCount {
		// 材料数量不匹配
		return 0, nil, ErrPetStarUpConditionNotMet
	}

	used := make(map[int64]bool)
	for _, ownID := range materialsPetOwnIds {
		// 不允许使用目标宠物自身
		if ownID == petOwnID {
			return 0, nil, ErrPetStarUpConditionNotMet
		}
		if used[ownID] {
			return 0, nil, ErrPetStarUpConditionNotMet
		}
		used[ownID] = true

		mat := player.PetModel.GetPet(ownID)
		if mat == nil || mat.IsDeleted {
			return 0, nil, ErrPetNotFound
		}
		// 升星材料：必须与目标宠物同配置ID（同一种宠物互相吞噬升星）
		if mat.PetID != p.PetID {
			return 0, nil, ErrPetStarUpConditionNotMet
		}
		sacrificePetOwnIds = append(sacrificePetOwnIds, ownID)
	}

	return nextStar, sacrificePetOwnIds, nil
}

// Rebirth 宠物重生：按策划仅重置等级为 1，同时按历史升级消耗返还材料（材料列表由调用方发放）。
func Rebirth(player *model.PlayerModel, petOwnID int64) ([]*gameConfig.ItemConfig, error) {
	if player == nil || player.PetModel == nil {
		return nil, ErrPetNotFound
	}
	p := player.PetModel.GetPet(petOwnID)
	if p == nil {
		return nil, ErrPetNotFound
	}

	if p.Level <= 1 {
		// 已经是初始等级，无需重生
		return nil, ErrPetLevelMin
	}

	base := gameConfig.GetPetBaseCfg(p.PetID)
	if base == nil {
		return nil, ErrPetCfgNotFound
	}

	return itemMapToConfigs(calcPetLevelRefundMap(base, p.Level, gameConfig.GetPetLevelCfgByPotentialLevel)), nil
}

// EquipPet 宠物穿戴/卸下：opType==0 穿戴，否则卸下。
// 返回：操作后的宠物实体 + 受影响英雄信息（用于战力推送）。
func EquipPet(player *model.PlayerModel, petOwnID int64, heroOwnID int64, opType int32) (pet *model.PetEntity, heroInfos []*pb.HeroBagInfo, err error) {
	if player == nil || player.PetModel == nil || player.HeroDetailsModel == nil {
		return nil, nil, ErrPetNotFound
	}

	p := player.PetModel.GetPet(petOwnID)
	if p == nil || p.IsDeleted {
		return nil, nil, ErrPetNotFound
	}

	affected := make(map[int64]struct{})
	if p.HeroOwnId != 0 {
		affected[p.HeroOwnId] = struct{}{}
	}

	if opType == 0 {
		if heroOwnID == 0 || player.HeroDetailsModel.GetHero(heroOwnID) == nil {
			return nil, nil, errors.New("hero not found")
		}
		if old := player.PetModel.GetEquippedPetByHero(heroOwnID); old != nil && old.PetOwnID != p.PetOwnID {
			affected[heroOwnID] = struct{}{}
		} else {
			affected[heroOwnID] = struct{}{}
		}
		player.PetModel.WearPet(p.PetOwnID, heroOwnID)
	} else {
		player.PetModel.UnwearPet(p.PetOwnID)
	}

	heroInfos = make([]*pb.HeroBagInfo, 0, len(affected))
	for hid := range affected {
		if info := player.HeroDetailsModel.GetHeroInfoByOwnID(player, hid); info != nil {
			heroInfos = append(heroInfos, info)
		}
	}

	return player.PetModel.GetPet(p.PetOwnID), heroInfos, nil
}

// BuildPetDetailInfo 构建完整的 PetDetailInfo（属性 / 战力 / 技能），供 controller / 其它模块复用。
// player 目前主要用于未来若需要叠加其它来源（如缘分、Buff）时扩展，这里只按配置计算宠物自身属性。
func BuildPetDetailInfo(player *model.PlayerModel, e *model.PetEntity) *pb.PetDetailInfo {
	if e == nil || e.IsDeleted {
		return nil
	}

	detail := &pb.PetDetailInfo{
		PetOwnId:  e.PetOwnID,
		PetId:     e.PetID,
		Level:     e.Level,
		Star:      e.Star,
		HeroOwnId: e.HeroOwnId,
	}

	// 被动技能：直接返回宠物自身已有的被动技能列表
	detail.PassiveSkills = append(detail.PassiveSkills, e.PassiveSkills...)

	// 属性：按配置里填的 attr 做 KV 汇总（key=attrId，value=总和）
	attrSum := make(map[int32]int64)

	// base 配置
	if baseCfg := gameConfig.GetPetBaseCfg(e.PetID); baseCfg != nil {
		for i, id := range baseCfg.Attr {
			if i < 0 || i >= len(baseCfg.AttrNum) {
				continue
			}
			attrSum[id] += int64(baseCfg.AttrNum[i])
		}
	}

	// 等级配置：累加 1..(当前等级-1)
	// level 配置语义：level=n 表示 n->n+1 的属性增量。
	if baseCfg := gameConfig.GetPetBaseCfg(e.PetID); baseCfg != nil {
		if e.Level < 1 {
			e.Level = 1
		}
		for fromLevel := int32(1); fromLevel < e.Level; fromLevel++ {
			lcfg := gameConfig.GetPetLevelCfgByPotentialLevel(baseCfg.PetPotential, fromLevel)
			if lcfg == nil {
				continue
			}
			for i, id := range lcfg.Attr {
				if i < 0 || i >= len(lcfg.AttrNum) {
					continue
				}
				attrSum[id] += int64(lcfg.AttrNum[i])
			}
		}
	}

	// 星级配置：累加 0..(当前星级-1)
	// star 配置语义：star=n 表示 n->n+1 的属性增量。
	if e.Star < 0 {
		e.Star = 0
	}
	for fromStar := int32(0); fromStar < e.Star; fromStar++ {
		scfg := gameConfig.GetPetStarCfgByPetIdStar(e.PetID, fromStar)
		if scfg == nil {
			continue
		}
		for i, id := range scfg.Attr {
			if i < 0 || i >= len(scfg.AttrNum) {
				continue
			}
			attrSum[id] += int64(scfg.AttrNum[i])
		}
	}

	detail.Attributes = attrSum

	return detail
}

// GetPetBagInfo 获取玩家宠物背包信息（返回详情列表），用于登录整包下发。
func GetPetBagInfo(player *model.PlayerModel) []*pb.PetDetailInfo {
	if player == nil || player.PetModel == nil {
		return nil
	}
	res := make([]*pb.PetDetailInfo, 0, len(player.PetModel.Entities))
	for _, e := range player.PetModel.Entities {
		if e == nil || e.IsDeleted {
			continue
		}
		res = append(res, BuildPetDetailInfo(player, e))
	}
	return res
}

// GetPetDetail 详情（只读，不写库；被动技能在升星时 roll）。
func GetPetDetail(player *model.PlayerModel, petOwnID int64) (*pb.PetDetailResp, error) {
	if player == nil || player.PetModel == nil {
		return nil, ErrPetNotFound
	}
	e := player.PetModel.GetPet(petOwnID)
	if e == nil || e.IsDeleted {
		return nil, ErrPetNotFound
	}
	detail := BuildPetDetailInfo(player, e)
	return &pb.PetDetailResp{Pet: detail}, nil
}
