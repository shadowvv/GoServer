package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("dungeon", &DungeonCfgLoader{})
}

type DungeonCfgLoader struct {
	temp1 map[int32]*TowerCfg
	temp2 map[int32]*DungeonAdventureCfg
	temp3 map[int32]*DungeonMonsterWaveCfg
	temp4 map[int32]map[int32]*DungeonAdventureCfg
}

var _ configLoaderInterface = (*DungeonCfgLoader)(nil)

func (s *DungeonCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/dungeon.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*TowerCfg)
	s.temp2 = make(map[int32]*DungeonAdventureCfg)
	s.temp3 = make(map[int32]*DungeonMonsterWaveCfg)
	s.temp4 = make(map[int32]map[int32]*DungeonAdventureCfg)
	for _, row := range rawData["tower"] {
		var v TowerCfg
		v.Id = ParseInt(row["id"])
		v.Level = ParseInt(row["level"])
		v.LevelReward = ParseInt(row["levelReward"])
		v.SweepReward = ParseInt(row["sweepReward"])
		v.Stage = ParseInt(row["stage"])
		v.StageReward = ParseInt(row["stageReward"])
		v.Power = ParseInt(row["power"])
		v.Lineup = ParseIntArray(row["Lineup"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load tower error duplicate Id:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	for _, row := range rawData["adventure"] {
		var v DungeonAdventureCfg
		v.Id = ParseInt(row["id"])
		v.Type = ParseInt(row["type"])
		v.InstanceType = ParseInt(row["instanceType"])
		v.MainStage = ParseIntArray(row["mainStage"])
		v.Level = ParseInt(row["level"])
		v.LevelReward = ParseInt(row["levelReward"])
		v.MonsterSpawn = ParseStrMatrix(row["monsterSpawn"])
		v.Lineup = ParseIntMatrix(row["Lineup"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error duplicate Id:%d", v.Id))
		}
		s.temp2[v.Id] = &v
		if s.temp4[v.Type] == nil {
			s.temp4[v.Type] = make(map[int32]*DungeonAdventureCfg)
		}
		if s.temp4[v.Type][v.Level] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error duplicate Type/Level:%d/%d,configId:%d", v.Type, v.Level, v.Id))
		}
		s.temp4[v.Type][v.Level] = &v
	}

	for _, row := range rawData["monsterWave"] {
		var v DungeonMonsterWaveCfg
		v.Id = ParseInt(row["id"])
		v.MonsterId = ParseInt(row["monsterId"])
		v.MonsterNum = ParseInt(row["monsterNum"])
		v.DropGroup = ParseInt(row["dropGroup"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon monsterWave error duplicate Id:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	return nil
}

func (s *DungeonCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load tower error invalid ID:%d", id))
		}
		if GetDropCfg(v.LevelReward) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load tower error invalid LevelReward:%d,configId:%d", v.LevelReward, id))
		}
		if GetDropCfg(v.SweepReward) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load tower error invalid SweepReward:%d,configId:%d", v.SweepReward, id))
		}
		if v.StageReward != 0 && GetDropCfg(v.StageReward) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load tower error invalid StageReward:%d,configId:%d", v.StageReward, id))
		}
		for _, v := range v.Lineup {
			if GetMonsterCfg(v) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load tower error invalid Lineup:%d,configId:%d", v, id))
			}
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid ID:%d", id))
		}

		switch enum.DungeonAdventureInstanceType(v.InstanceType) {
		case enum.DungeonAdventureInstanceType_ADVENTURE:
			if GetAdventureCfg(v.Type) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid Type:%d,configId:%d", v.Type, id))
			}
		case enum.DungeonAdventureInstanceType_INSTANCE:
			instanceCfg := GetInstanceCfg(v.Type)
			if instanceCfg == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid Type:%d,configId:%d", v.Type, id))
			}
			if _, ok := enum.GetResidentDungeonType(instanceCfg.InstanceType); !ok {
				return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid Type:%d,configId:%d", v.Type, id))
			}
			if v.LevelReward != 0 && GetDropCfg(v.LevelReward) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid LevelReward:%d,configId:%d", v.LevelReward, id))
			}
		default:
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid Type:%d,configId:%d", v.Type, id))
		}

		if len(v.MainStage) != 0 {
			if len(v.MainStage) != 2 || v.MainStage[0] <= 0 || v.MainStage[1] < v.MainStage[0] {
				return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid MainStage:%v,configId:%d", v.MainStage, id))
			}
		}
		if len(v.Lineup) == 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid Lineup,configId:%d", id))
		}
		if len(v.MonsterSpawn) != len(v.Lineup) {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid MonsterSpawn/Lineup,configId:%d", id))
		}
		for index, lineup := range v.Lineup {
			if len(lineup) == 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid Lineup,configId:%d", id))
			}
			if len(v.MonsterSpawn[index]) != len(lineup) {
				return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid MonsterSpawn/Lineup,configId:%d", id))
			}
			for _, waveId := range lineup {
				if GetDungeonMonsterWaveCfg(waveId) == nil {
					return errors.New(fmt.Sprintf("[gameConfig] load dungeon adventure error invalid Lineup:%d,configId:%d", waveId, id))
				}
			}
		}
	}
	for id, v := range s.temp3 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon monsterWave error invalid ID:%d", id))
		}
		if GetMonsterCfg(v.MonsterId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon monsterWave error invalid MonsterId:%d,configId:%d", v.MonsterId, id))
		}
		if v.MonsterNum <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon monsterWave error invalid MonsterNum:%d,configId:%d", v.MonsterNum, id))
		}
		if v.DropGroup != 0 && GetDropGroupCfg(v.DropGroup) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load dungeon monsterWave error invalid DropGroup:%d,configId:%d", v.DropGroup, id))
		}
	}
	return nil
}

func (s *DungeonCfgLoader) apply() {
	tower.Store(s.temp1)
	dungeonAdventure.Store(s.temp2)
	dungeonMonsterWave.Store(s.temp3)
	dungeonAdventureTypeLevel.Store(s.temp4)
}

var tower atomic.Value
var dungeonAdventure atomic.Value
var dungeonMonsterWave atomic.Value
var dungeonAdventureTypeLevel atomic.Value

type TowerCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 关卡层数
	Level int32 `json:"level"`
	// 关卡奖励
	LevelReward int32 `json:"levelReward"`
	// 关卡扫荡奖励
	SweepReward int32 `json:"sweepReward"`
	// 关卡阶段
	Stage int32 `json:"stage"`
	// 阶段奖励
	StageReward int32 `json:"stageReward"`
	// 战力显示
	Power int32 `json:"power"`
	// 阵容配置
	Lineup []int32 `json:"Lineup"`
}

type DungeonAdventureCfg struct {
	Id           int32      `json:"id"`
	Type         int32      `json:"type"`
	InstanceType int32      `json:"instanceType"`
	MainStage    []int32    `json:"mainStage"`
	Level        int32      `json:"level"`
	LevelReward  int32      `json:"levelReward"`
	MonsterSpawn [][]string `json:"monsterSpawn"`
	Lineup       [][]int32  `json:"Lineup"`
}

type DungeonMonsterWaveCfg struct {
	Id         int32 `json:"id"`
	MonsterId  int32 `json:"monsterId"`
	MonsterNum int32 `json:"monsterNum"`
	DropGroup  int32 `json:"dropGroup"`
}

func GetTowerCfg(Id int32) *TowerCfg {
	cfgMap := tower.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*TowerCfg)[Id]
}

func GetDungeonAdventureCfg(id int32) *DungeonAdventureCfg {
	cfgMap := dungeonAdventure.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*DungeonAdventureCfg)[id]
}

func GetAllDungeonAdventureCfg() map[int32]*DungeonAdventureCfg {
	cfgMap := dungeonAdventure.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*DungeonAdventureCfg)
}

func GetDungeonAdventureCfgByTypeAndLevel(adventureType int32, level int32) *DungeonAdventureCfg {
	cfgMap := dungeonAdventureTypeLevel.Load()
	if cfgMap == nil {
		return nil
	}
	typeCfg := cfgMap.(map[int32]map[int32]*DungeonAdventureCfg)[adventureType]
	if typeCfg == nil {
		return nil
	}
	return typeCfg[level]
}

func GetDungeonMonsterWaveCfg(id int32) *DungeonMonsterWaveCfg {
	cfgMap := dungeonMonsterWave.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*DungeonMonsterWaveCfg)[id]
}
