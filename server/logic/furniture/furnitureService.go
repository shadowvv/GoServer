package furniture

import (
	"errors"
	"sort"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
)

const (
	furnitureEffectTypeOutput int32 = 1 // 家具效果类型：产出加成
	furnitureEffectTypeLimit  int32 = 2 // 家具效果类型：储量加成

	architectureStatusUnbuilt  int32 = 0 // 建筑状态：未建造
	architectureStatusBuilding int32 = 3 // 建筑状态：建造中（首次建造）
)

// FurnitureService 家具通用服务
type FurnitureService struct {
	beforeEffectChange BeforeEffectChangeFunc
}

type BeforeEffectChangeFunc func(player *model.PlayerModel, buildingType int32) error

func NewFurnitureService(beforeEffectChange ...BeforeEffectChangeFunc) *FurnitureService {
	service := &FurnitureService{}
	if len(beforeEffectChange) > 0 {
		service.beforeEffectChange = beforeEffectChange[0]
	}
	return service
}

// FurnitureEffect 家具聚合效果
type FurnitureEffect struct {
	Output   map[int32]int64 // 每窗口产出加成
	Limit    map[int32]int64 // 储存上限加成
	Progress int32           // 家具最终进度
}

// LevelUp 家具升级（每次升一级）
func (s *FurnitureService) LevelUp(player *model.PlayerModel, buildingType int32, furnitureType int32) error {
	archEntity, err := s.getArchitecture(player, buildingType)
	if err != nil {
		return err
	}
	if player.FurnitureModel == nil {
		return errors.New("furniture model nil")
	}
	if gameConfig.GetFurnitureMaxLevel(buildingType, furnitureType) <= 0 {
		return errors.New("furniture type error")
	}
	if !gameConfig.IsFurnitureUnlocked(buildingType, archEntity.Level, furnitureType) {
		return errors.New("furniture not unlock")
	}

	currentLevel := player.FurnitureModel.GetLevel(buildingType, furnitureType)
	nextLevel := currentLevel + 1
	if nextLevel > archEntity.Level {
		return errors.New("furniture level max")
	}
	maxLevel := gameConfig.GetFurnitureMaxLevel(buildingType, furnitureType)
	if currentLevel >= maxLevel {
		return errors.New("furniture level max")
	}

	cfg := gameConfig.GetFurnitureCfgByBuildingTypeLevel(buildingType, furnitureType, nextLevel)
	if cfg == nil {
		return errors.New("furniture config not found")
	}
	ok, err := itemService.CheckItemsCount(player, cfg.Item)
	if !ok || err != nil {
		return errors.New("item not enough")
	}
	if _, err := player.FurnitureModel.GetOrCreate(buildingType, furnitureType); err != nil {
		return err
	}
	if s.beforeEffectChange != nil {
		if err := s.beforeEffectChange(player, buildingType); err != nil {
			return err
		}
	}
	if err := itemService.RemoveItems(player, cfg.Item, enum.ITEM_CHANGE_REASON_CITY_FURNITURE_LEVEL_UP); err != nil {
		return err
	}
	return player.FurnitureModel.UpdateLevel(buildingType, furnitureType, nextLevel)
}

// CalcEffect 计算指定建筑的家具效果
func (s *FurnitureService) CalcEffect(player *model.PlayerModel, buildingType int32) *FurnitureEffect {
	effect := &FurnitureEffect{
		Output: make(map[int32]int64),
		Limit:  make(map[int32]int64),
	}
	if player == nil || player.FurnitureModel == nil || player.FurnitureModel.Entities[buildingType] == nil {
		return effect
	}
	for furnitureType, entity := range player.FurnitureModel.Entities[buildingType] {
		cfg := gameConfig.GetFurnitureCfgByBuildingTypeLevel(buildingType, furnitureType, entity.Level)
		if cfg == nil {
			continue
		}
		effect.Progress += cfg.Progress
		switch cfg.EffectType {
		case furnitureEffectTypeOutput:
			for _, item := range cfg.BaseEffect {
				effect.Output[item.ID] += item.Num
			}
		case furnitureEffectTypeLimit:
			for _, item := range cfg.BaseEffect {
				effect.Limit[item.ID] += item.Num
			}
		}
	}
	return effect
}

// BuildFurnitureInfos 构造指定建筑当前已解锁家具信息
func (s *FurnitureService) BuildFurnitureInfos(player *model.PlayerModel, buildingType int32, buildingLevel int32) []*pb.CityFurnitureInfo {
	unlockedFurniture := gameConfig.GetUnlockedFurnitureTypes(buildingType, buildingLevel)
	furnitureInfos := make([]*pb.CityFurnitureInfo, 0, len(unlockedFurniture))
	for _, furnitureType := range unlockedFurniture {
		level := int32(0)
		if player != nil && player.FurnitureModel != nil {
			level = player.FurnitureModel.GetLevel(buildingType, furnitureType)
		}
		furnitureInfos = append(furnitureInfos, &pb.CityFurnitureInfo{
			FurnitureType: furnitureType,
			Level:         level,
		})
	}
	sort.Slice(furnitureInfos, func(i, j int) bool {
		return furnitureInfos[i].FurnitureType < furnitureInfos[j].FurnitureType
	})
	return furnitureInfos
}

func (s *FurnitureService) getArchitecture(player *model.PlayerModel, buildingType int32) (*model.ArchitectureEntity, error) {
	if player == nil || player.ArchitectureModel == nil {
		return nil, errors.New("building not built")
	}
	archEntity := player.ArchitectureModel.Entities[buildingType]
	if archEntity == nil || !isBuildingAvailable(archEntity.Status) {
		return nil, errors.New("building not built")
	}
	return archEntity, nil
}

func isBuildingAvailable(status int32) bool {
	return status != architectureStatusUnbuilt && status != architectureStatusBuilding
}
