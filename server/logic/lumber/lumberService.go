package lumber

import (
	"errors"
	"sort"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/furniture"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

const (
	architectureStatusUnbuilt  int32 = 0 // 建筑状态：未建造
	architectureStatusBuilding int32 = 3 // 建筑状态：建造中（首次建造）
)

// LumberService 伐木场生产服务
// 负责伐木场的核心业务逻辑：生产结算、暂存领取、派驻英雄、家具升级等
type LumberService struct {
	furnitureService *furniture.FurnitureService
}

var Service = NewLumberService()

func NewLumberService() *LumberService {
	return &LumberService{
		furnitureService: furniture.NewFurnitureService(),
	}
}

// GetDetailInfo 获取伐木场详情
// 先结算完整的生产窗口，再返回当前伐木场的完整信息
// 用于客户端打开伐木场界面
func (s *LumberService) GetDetailInfo(player *model.PlayerModel) (*pb.CityLumberInfo, error) {
	archEntity, err := s.getLumberArchitecture(player)
	if err != nil {
		return nil, err
	}
	prodEntity, err := player.LumberModel.GetOrCreate(archEntity.Type)
	if err != nil {
		return nil, err
	}
	if err := s.settle(player, archEntity.Type, archEntity.Level, prodEntity); err != nil {
		return nil, err
	}
	return s.buildLumberInfo(player, archEntity, prodEntity)
}

// Collect 领取暂存产物
// 指定一个道具ID进行领取，先结算完整窗口的产出，再将对应暂存清空发放给玩家
// itemID: 要领取的道具ID
// 返回: 本次领取的奖励、剩余暂存（当前固定nil）、错误
func (s *LumberService) Collect(player *model.PlayerModel, itemID int32) (*pb.ItemBasicInfo, *pb.ItemBasicInfo, error) {
	if itemID <= 0 {
		return nil, nil, errors.New("store empty")
	}
	archEntity, err := s.getLumberArchitecture(player)
	if err != nil {
		return nil, nil, err
	}
	prodEntity, err := player.LumberModel.GetOrCreate(archEntity.Type)
	if err != nil {
		return nil, nil, err
	}
	// 领取前先结算完整窗口
	if err := s.settle(player, archEntity.Type, archEntity.Level, prodEntity); err != nil {
		return nil, nil, err
	}

	stored, err := decodeStored(prodEntity)
	if err != nil {
		return nil, nil, err
	}
	count := stored[itemID]
	if count <= 0 {
		return nil, nil, errors.New("store empty")
	}

	// 发放奖励
	reward := &pb.ItemBasicInfo{ItemId: itemID, Count: count}
	if err := itemService.AddItems(player, []*gameConfig.ItemConfig{{ID: itemID, Num: count}}, enum.ITEM_CHANGE_REASON_CITY_LUMBER_COLLECT); err != nil {
		return nil, nil, err
	}

	// 清空已领取的暂存项
	delete(stored, itemID)
	player.LumberModel.UpdateStored(archEntity.Type, model.EncodeStored(stored))

	return reward, nil, nil
}

// AssignHero 设置派驻英雄（全量替换）
// heroOwnIDs: 新的派驻英雄列表（传空表示全部解绑）
// 如果英雄集合没变（只是顺序变了），则只更新顺序不触发结算
// 如果英雄集合有变化，则先按旧派驻参数结算，再保存新列表
// 返回: 最终派驻列表、当前暂存、错误
func (s *LumberService) AssignHero(player *model.PlayerModel, heroOwnIDs []int64) ([]int64, []*pb.ItemBasicInfo, error) {
	archEntity, err := s.getLumberArchitecture(player)
	if err != nil {
		return nil, nil, err
	}
	prodEntity, err := player.LumberModel.GetOrCreate(archEntity.Type)
	if err != nil {
		return nil, nil, err
	}

	currentHeroOwnIDs := []int64(prodEntity.HeroOwnIds)

	// 英雄集合没变（只是顺序变了），不需要结算，直接更新顺序
	sameSet := len(currentHeroOwnIDs) == len(heroOwnIDs)
	if sameSet {
		counts := make(map[int64]int32, len(currentHeroOwnIDs))
		for _, heroOwnID := range currentHeroOwnIDs {
			counts[heroOwnID]++
		}
		for _, heroOwnID := range heroOwnIDs {
			counts[heroOwnID]--
			if counts[heroOwnID] < 0 {
				sameSet = false
				break
			}
		}
	}
	if sameSet {
		stored, err := decodeStored(prodEntity)
		if err != nil {
			return nil, nil, err
		}
		// 顺序不同时更新一下
		sameOrder := true
		for i, heroOwnID := range currentHeroOwnIDs {
			if heroOwnID != heroOwnIDs[i] {
				sameOrder = false
				break
			}
		}
		if !sameOrder {
			player.LumberModel.UpdateHeroOwnIds(archEntity.Type, tool.JSONInt64Slice(heroOwnIDs))
		}
		return heroOwnIDs, mapToItemBasicInfo(stored), nil
	}

	// 英雄集合有变化，校验配置和英雄合法性
	cfg := gameConfig.GetLumberCfg(archEntity.Level)
	if cfg == nil {
		return nil, nil, errors.New("config not found")
	}
	if int32(len(heroOwnIDs)) > cfg.HeroSlots {
		return nil, nil, errors.New("hero slot not enough")
	}

	// 校验英雄存在性和去重
	seen := make(map[int64]bool, len(heroOwnIDs))
	for _, ownID := range heroOwnIDs {
		if seen[ownID] {
			return nil, nil, errors.New("hero duplicate")
		}
		seen[ownID] = true
		hero := player.HeroDetailsModel.GetHero(ownID)
		if hero == nil || hero.IsDeleted {
			return nil, nil, errors.New("hero invalid")
		}
	}

	// 先按旧派驻参数结算一次，再保存新英雄列表
	if err := s.settle(player, archEntity.Type, archEntity.Level, prodEntity); err != nil {
		return nil, nil, err
	}
	player.LumberModel.UpdateHeroOwnIds(archEntity.Type, tool.JSONInt64Slice(heroOwnIDs))

	stored, err := decodeStored(prodEntity)
	if err != nil {
		return nil, nil, err
	}
	return heroOwnIDs, mapToItemBasicInfo(stored), nil
}

// OnBuildingUpgradeComplete 建筑升级完成回调
// 由 ArchitectureModel.OnUpgradeCallback 触发，在建筑升级完成时自动调用
// 按旧等级/旧参数结算一次产出，新等级的产出参数从此刻开始生效
func (s *LumberService) OnBuildingUpgradeComplete(player *model.PlayerModel, archType int32, oldLevel int32) {
	if archType != int32(enum.ARCHITECTURE_TYPE_LUMBERYARD) {
		return
	}
	prodEntity, err := player.LumberModel.GetOrCreate(archType)
	if err != nil {
		logger.ErrorBySprintf("[lumber] building upgrade complete get production error userId:%d archType:%d err:%v", player.GetUserId(), archType, err)
		return
	}
	// 按旧等级结算一次，确保升级前这段时间的产出用旧参数计算
	if err := s.settle(player, archType, oldLevel, prodEntity); err != nil {
		logger.ErrorBySprintf("[lumber] building upgrade complete settle error userId:%d archType:%d err:%v", player.GetUserId(), archType, err)
	}
}

// BeforeAssignedHeroChange 派驻英雄属性变化前的结算钩子
// 当英雄升级/升星/转职等操作影响了已派驻英雄的属性时调用
// 先按旧的英雄加成结算一次，之后英雄属性更新后新加成自然生效
func (s *LumberService) BeforeAssignedHeroChange(player *model.PlayerModel, heroOwnID int64) {
	archType := int32(enum.ARCHITECTURE_TYPE_LUMBERYARD)
	prodEntity := player.LumberModel.Entities[archType]
	if prodEntity == nil {
		return
	}
	assigned := false
	for _, assignedHeroOwnID := range prodEntity.HeroOwnIds {
		if assignedHeroOwnID == heroOwnID {
			assigned = true
			break
		}
	}
	if !assigned {
		return
	}
	archEntity := player.ArchitectureModel.Entities[archType]
	if archEntity == nil || archEntity.Status == architectureStatusUnbuilt || archEntity.Status == architectureStatusBuilding {
		return
	}
	if err := s.settle(player, archType, archEntity.Level, prodEntity); err != nil {
		logger.ErrorBySprintf("[lumber] assigned hero change settle error userId:%d heroOwnID:%d err:%v", player.GetUserId(), heroOwnID, err)
	}
}

// BeforeAssignedHeroesDelete 派驻英雄被消耗/删除前的下阵钩子
// 如果被删除的英雄仍在伐木场派驻列表中，先按旧派驻结算，再从派驻列表移除。
func (s *LumberService) BeforeAssignedHeroesDelete(player *model.PlayerModel, heroOwnIDs []int64) {
	if len(heroOwnIDs) == 0 {
		return
	}
	archType := int32(enum.ARCHITECTURE_TYPE_LUMBERYARD)
	prodEntity := player.LumberModel.Entities[archType]
	if prodEntity == nil || len(prodEntity.HeroOwnIds) == 0 {
		return
	}

	removeSet := make(map[int64]bool, len(heroOwnIDs))
	for _, heroOwnID := range heroOwnIDs {
		removeSet[heroOwnID] = true
	}

	assignedHeroOwnIDs := []int64(prodEntity.HeroOwnIds)
	newHeroOwnIDs := make([]int64, 0, len(assignedHeroOwnIDs))
	changed := false
	for _, heroOwnID := range assignedHeroOwnIDs {
		if removeSet[heroOwnID] {
			changed = true
			continue
		}
		newHeroOwnIDs = append(newHeroOwnIDs, heroOwnID)
	}
	if !changed {
		return
	}

	archEntity := player.ArchitectureModel.Entities[archType]
	if archEntity != nil && archEntity.Status != architectureStatusUnbuilt && archEntity.Status != architectureStatusBuilding {
		if err := s.settle(player, archType, archEntity.Level, prodEntity); err != nil {
			logger.ErrorBySprintf("[lumber] assigned hero delete settle error userId:%d heroOwnIDs:%v err:%v", player.GetUserId(), heroOwnIDs, err)
		}
	}
	player.LumberModel.UpdateHeroOwnIds(archType, tool.JSONInt64Slice(newHeroOwnIDs))
}

// BeforeFurnitureEffectChange 家具效果变化前的结算钩子
// 家具升级会改变生产参数，先按旧参数结算已产生的完整窗口产出。
func (s *LumberService) BeforeFurnitureEffectChange(player *model.PlayerModel, archType int32) error {
	if archType != int32(enum.ARCHITECTURE_TYPE_LUMBERYARD) {
		return errors.New("architecture type not supported")
	}
	archEntity, err := s.getLumberArchitecture(player)
	if err != nil {
		return err
	}
	prodEntity, err := player.LumberModel.GetOrCreate(archType)
	if err != nil {
		return err
	}
	return s.settle(player, archType, archEntity.Level, prodEntity)
}

// CheckUpgradeProgress 检查伐木场升级前的家具进度条件
// 升级到下一级时，需要家具累计进度 >= 配置的 progress 值
// 非伐木场建筑类型直接返回 true（不校验）
func (s *LumberService) CheckUpgradeProgress(player *model.PlayerModel, archType int32) bool {
	if archType != int32(enum.ARCHITECTURE_TYPE_LUMBERYARD) {
		return true
	}
	archEntity := player.ArchitectureModel.Entities[archType]
	if archEntity == nil {
		return true
	}
	nextCfg := gameConfig.GetLumberCfg(archEntity.Level + 1)
	if nextCfg == nil || nextCfg.Progress <= 0 {
		return true
	}
	effect := s.calcEffect(player, archEntity.Level, nil)
	return effect.Progress >= nextCfg.Progress
}

// ===================== 内部方法 =====================

// getLumberArchitecture 获取伐木场建筑实体，校验建筑是否已建成
func (s *LumberService) getLumberArchitecture(player *model.PlayerModel) (*model.ArchitectureEntity, error) {
	archType := int32(enum.ARCHITECTURE_TYPE_LUMBERYARD)
	archEntity := player.ArchitectureModel.Entities[archType]
	if archEntity == nil || archEntity.Status == architectureStatusUnbuilt || archEntity.Status == architectureStatusBuilding {
		return nil, errors.New("building not built")
	}
	return archEntity, nil
}

// LumberEffect 伐木场生产效果（一次性计算所有参数）
// 由 calcEffect 生成，供 settle / CheckUpgradeProgress 等统一使用
type LumberEffect struct {
	Output    map[int32]int64 // 每窗口基础产出（建筑产出 + 家具产出加成）
	Limit     map[int32]int64 // 储存上限（建筑上限 + 家具储量加成）
	HeroBonus int32           // 派驻英雄加成百分比（如 30 表示 +30%）
	Progress  int32           // 家具累计进度
}

// calcEffect 一次性计算伐木场全部生产效果参数
// 遍历建筑配置和家具配置各一次，同时得出产出/上限/进度，再叠加英雄加成
//
// buildingLevel: 用于查配置的建筑等级（升级完成回调时传旧等级）
// heroOwnIDs:    当前派驻英雄列表（仅查进度时可传nil跳过英雄计算）
func (s *LumberService) calcEffect(player *model.PlayerModel, buildingLevel int32, heroOwnIDs []int64) *LumberEffect {
	archType := int32(enum.ARCHITECTURE_TYPE_LUMBERYARD)
	effect := &LumberEffect{
		Output: make(map[int32]int64),
		Limit:  make(map[int32]int64),
	}

	// 建筑等级对应的基础产出和上限
	cfg := gameConfig.GetLumberCfg(buildingLevel)
	if cfg != nil {
		for _, item := range cfg.Output {
			effect.Output[item.ID] += item.Num
		}
		for _, item := range cfg.Limit {
			effect.Limit[item.ID] += item.Num
		}
	}

	// 家具效果由通用家具服务聚合
	furnitureEffect := s.furnitureService.CalcEffect(player, archType)
	for itemID, num := range furnitureEffect.Output {
		effect.Output[itemID] += num
	}
	for itemID, num := range furnitureEffect.Limit {
		effect.Limit[itemID] += num
	}
	effect.Progress += furnitureEffect.Progress

	// 英雄加成
	effect.HeroBonus = s.calcHeroBonus(player, heroOwnIDs)

	return effect
}

// calcHeroBonus 计算派驻英雄的加成百分比
// 加成来源：
// - 职业加成：英雄职业匹配 lumberBonusHeroClass 时，加 classPercent%
// - 潜力加成：英雄潜力 * cityProductBonusHeroPotential%
// - 星级加成：英雄星级 * cityProductBonusHeroStar%
func (s *LumberService) calcHeroBonus(player *model.PlayerModel, heroOwnIDs []int64) int32 {
	if len(heroOwnIDs) == 0 {
		return 0
	}
	totalBonus := int32(0)
	bonusClass, classPercent := gameConfig.GetLumberBonusHeroClass()
	potentialPercent := gameConfig.GetCityProductBonusHeroPotential()
	starPercent := gameConfig.GetCityProductBonusHeroStar()

	for _, ownID := range heroOwnIDs {
		hero := player.HeroDetailsModel.GetHero(ownID)
		if hero == nil || hero.IsDeleted {
			continue
		}
		heroCfg := gameConfig.GetHeroBaseCfg(int32(hero.HeroID))
		if heroCfg == nil {
			continue
		}
		if bonusClass > 0 && heroCfg.HeroClass == bonusClass {
			totalBonus += classPercent
		}
		if potentialPercent > 0 {
			totalBonus += heroCfg.HeroPotential * potentialPercent
		}
		if starPercent > 0 {
			totalBonus += hero.StarLevel * starPercent
		}
	}
	return totalBonus
}

// settle 生产结算（核心方法）
// 按完整窗口数结算产出，不足一个窗口的时间不产出
// 产出公式: floor(基础产出 * 完整窗口数 * (100 + 英雄加成百分比) / 100)
// 加入暂存时按储存上限截断
func (s *LumberService) settle(player *model.PlayerModel, archType int32, buildingLevel int32, prodEntity *model.LumberEntity) error {
	now := tool.UnixNowMilli()
	if prodEntity.LastCalcTime == 0 {
		player.LumberModel.UpdateLastCalcTime(archType, now)
		return nil
	}

	windowMs := int64(gameConfig.GetCityProductionTime()) * 1000
	if windowMs <= 0 {
		return nil
	}
	delta := now - prodEntity.LastCalcTime
	times := delta / windowMs
	if times <= 0 {
		return nil
	}

	effect := s.calcEffect(player, buildingLevel, []int64(prodEntity.HeroOwnIds))

	stored, err := decodeStored(prodEntity)
	if err != nil {
		return err
	}

	for itemID, basePerWindow := range effect.Output {
		produced := basePerWindow * times * int64(100+effect.HeroBonus) / 100
		stored[itemID] += produced
		if maxVal, ok := effect.Limit[itemID]; ok && stored[itemID] > maxVal {
			stored[itemID] = maxVal
		}
	}

	player.LumberModel.UpdateStored(archType, model.EncodeStored(stored))
	player.LumberModel.UpdateLastCalcTime(archType, prodEntity.LastCalcTime+times*windowMs)
	return nil
}

// buildLumberInfo 构建伐木场完整信息（用于协议返回和推送）
// 包含：暂存产物、派驻英雄列表、已解锁家具及其等级
func (s *LumberService) buildLumberInfo(player *model.PlayerModel, archEntity *model.ArchitectureEntity, prodEntity *model.LumberEntity) (*pb.CityLumberInfo, error) {
	stored, err := decodeStored(prodEntity)
	if err != nil {
		return nil, err
	}

	return &pb.CityLumberInfo{
		Stored:        mapToItemBasicInfo(stored),
		HeroOwnId:     []int64(prodEntity.HeroOwnIds),
		FurnitureInfo: s.furnitureService.BuildFurnitureInfos(player, archEntity.Type, archEntity.Level),
	}, nil
}

// ===================== 工具函数 =====================

// decodeStored 解析暂存产物JSON字符串为 map[道具ID]数量
func decodeStored(prodEntity *model.LumberEntity) (map[int32]int64, error) {
	stored, err := model.DecodeStored(prodEntity.Stored)
	if err != nil {
		return nil, errors.New("stored decode error")
	}
	return stored, nil
}

// mapToItemBasicInfo 将 map[道具ID]数量 转换为协议的 ItemBasicInfo 列表
// 过滤掉数量为0的项，按道具ID排序保证顺序稳定
func mapToItemBasicInfo(m map[int32]int64) []*pb.ItemBasicInfo {
	result := make([]*pb.ItemBasicInfo, 0, len(m))
	for itemID, count := range m {
		if count > 0 {
			result = append(result, &pb.ItemBasicInfo{ItemId: itemID, Count: count})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ItemId < result[j].ItemId
	})
	return result
}
