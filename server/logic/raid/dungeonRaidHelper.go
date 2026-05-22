package raid

import (
	"slices"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
)

func buildDungeonRaidSubStage(raidInfo *logicCommon.PlayerInstanceRaid, dungeonCfg *gameConfig.DungeonAdventureCfg) *logicCommon.SubStageData {
	monsterIndex := int32(0)
	subStageData := &logicCommon.SubStageData{
		SubStageId:  raidInfo.CurrentStageId,
		RoomID:      0,
		MonsterInfo: make(map[int32]*logicCommon.MonsterInfo),
		ComboSkills: make([]int32, 0),
	}
	monsterFormation := &logicCommon.FormationBasicInfo{
		Heroes:      make([]int64, 0),
		BattlePower: 0,
	}
	monsterHeroDetails := make(map[int64]*logicCommon.HeroBasicInfo)
	for waveIndex, lineup := range dungeonCfg.Lineup {
		for spawnIndex, waveId := range lineup {
			waveCfg := gameConfig.GetDungeonMonsterWaveCfg(waveId)
			monsterCfg := gameConfig.GetMonsterCfg(waveCfg.MonsterId)
			spawnId := dungeonCfg.MonsterSpawn[waveIndex][spawnIndex]
			monsterFormation.Heroes = append(monsterFormation.Heroes, int64(waveCfg.MonsterId))
			monsterHeroDetails[int64(waveCfg.MonsterId)] = &logicCommon.HeroBasicInfo{
				Id:   int64(waveCfg.MonsterId),
				Star: monsterCfg.Star,
			}
			for i := int32(0); i < waveCfg.MonsterNum; i++ {
				monsterIndex++
				monsterInfo := &logicCommon.MonsterInfo{
					SpawnId:      spawnId,
					WaveId:       waveCfg.Id,
					WaveSequence: int32(waveIndex + 1),
					Id:           subStageData.SubStageId*adventureMonsterInstanceIdFactor + monsterIndex,
					MonsterId:    waveCfg.MonsterId,
					DropItems:    make([]*gameConfig.ItemConfig, 0),
				}
				if waveCfg.DropGroup != 0 {
					monsterInfo.DropItems = gameConfig.DropGroupItems(waveCfg.DropGroup, nil)
				}
				subStageData.MonsterInfo[monsterInfo.Id] = monsterInfo
			}
			if _, ok := raidInfo.MonsterTemplates[int64(monsterCfg.Id)]; !ok {
				raidInfo.MonsterTemplates[int64(monsterCfg.Id)] = buildDungeonMonsterTemplate(monsterCfg)
			}
		}
	}
	subStageData.ComboSkills = GetEnemyComberSkillIds(0, monsterFormation, monsterHeroDetails)
	return subStageData
}

func buildDungeonMonsterTemplate(monsterCfg *gameConfig.MonsterCfg) *logicCommon.MonsterTemplate {
	monsterTemplate := &logicCommon.MonsterTemplate{
		MonsterId:   int64(monsterCfg.Id),
		UnitId:      monsterCfg.Units,
		AtkSpeed:    monsterCfg.AtkSpeed,
		MoveSpeed:   monsterCfg.MoveSpeed,
		PatrolRange: monsterCfg.PatrolRange,
		AggroRange:  monsterCfg.AggroRange,
		AttackRange: monsterCfg.AttackRange,
		BasicSkill:  monsterCfg.NormalAtk,
		AttrInfo:    make(map[int32]int64),
		SkillId:     make([]int32, 0),
	}
	for id, attr := range monsterCfg.Attr {
		monsterTemplate.AttrInfo[id] = attr
	}
	for _, skill := range monsterCfg.Skill {
		monsterTemplate.SkillId = append(monsterTemplate.SkillId, skill)
	}
	return monsterTemplate
}

func mergeItemConfigs(items []*gameConfig.ItemConfig) []*gameConfig.ItemConfig {
	if len(items) == 0 {
		return items
	}
	itemMap := make(map[int32]int64)
	for _, item := range items {
		itemMap[item.ID] += item.Num
	}
	itemIds := make([]int32, 0, len(itemMap))
	for itemId := range itemMap {
		itemIds = append(itemIds, itemId)
	}
	slices.Sort(itemIds)
	mergeItems := make([]*gameConfig.ItemConfig, 0, len(itemIds))
	for _, itemId := range itemIds {
		mergeItems = append(mergeItems, &gameConfig.ItemConfig{
			ID:  itemId,
			Num: itemMap[itemId],
		})
	}
	return mergeItems
}

func addDungeonSettleDropItems(player *model.PlayerModel, info *logicCommon.PlayerInstanceRaid, extraItems ...*gameConfig.ItemConfig) {
	dropItems := make([]*gameConfig.ItemConfig, 0)
	for _, subStageInfo := range info.SubStageInfo {
		for _, monster := range subStageInfo.MonsterInfo {
			if monster.IsDead != 1 {
				continue
			}
			dropItems = append(dropItems, monster.DropItems...)
		}
	}
	dropItems = mergeItemConfigs(append(dropItems, extraItems...))
	if len(dropItems) > 0 {
		_ = itemService.AddItems(player, dropItems, enum.ITEM_CHANGE_REASON_ADVENTURE_REWARD)
	}
}
