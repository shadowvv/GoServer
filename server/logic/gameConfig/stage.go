package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("stage", &StageCfgLoader{})
}

type StageCfgLoader struct {
	temp1 map[int32]*MainStageCfg
	temp2 map[int32]*MonsterWaveCfg
	temp3 map[int32]*SubStageCfg
}

var _ configLoaderInterface = (*StageCfgLoader)(nil)

func (s *StageCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/stage.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*MainStageCfg)
	for _, row := range rawData["mainStage"] {
		var v MainStageCfg
		v.Id = ParseInt(row["id"])
		v.BackStage = ParseInt(row["backStage"])
		v.InstanceId = ParseInt(row["instanceId"])
		v.ChapterId = ParseInt(row["chapterId"])
		v.SubStageId = ParseIntArray(row["subStageId"])
		v.IdleDrop = ParseIntArray(row["idleDrop"])
		v.MailId = ParseIntArray(row["mailId"])
		v.Unlock = ParseInt(row["unlock"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load mainStage error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*MonsterWaveCfg)
	for _, row := range rawData["monsterWave"] {
		var v MonsterWaveCfg
		v.Id = ParseInt(row["id"])
		v.MonsterId = ParseInt(row["monsterId"])
		v.MonsterNum = ParseInt(row["monsterNum"])
		v.FirstDrop = ParseInt(row["firstDrop"])
		v.EachDrop = ParseInt(row["eachDrop"])
		v.PrivilegeDrop = ParseInt(row["privilegeDrop"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load monsterWave error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*SubStageCfg)
	for _, row := range rawData["subStage"] {
		var v SubStageCfg
		v.Id = ParseInt(row["id"])
		v.RoomId = ParseIntArray(row["roomId"])
		v.MonsterSpawn = ParseStrMatrix(row["monsterSpawn"])
		v.MonsterWaveId = ParseIntMatrix(row["monsterWaveId"])
		v.BarrelSpawn = ParseStrArray(row["barrelSpawn"])
		v.BarrelWaveId = ParseIntArray(row["barrelWaveId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load subStage error duplicate ID:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	return nil
}

func (s *StageCfgLoader) checkData() error {
	subStageIdMap := make(map[int32]struct{})
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load mainStage error invalid ID:%d", id))
		}
		if v.BackStage != 0 && s.temp1[v.BackStage] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load mainStage error invalid BackStage:%d,configId:%d", v.BackStage, id))
		}
		if GetInstanceCfg(v.InstanceId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load mainStage error invalid InstanceId:%d,configId:%d", v.InstanceId, id))
		}
		if v.ChapterId <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load mainStage error invalid ChapterId:%d,configId:%d", v.ChapterId, id))
		}
		for _, v := range v.SubStageId {
			if s.temp3[v] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load mainStage error invalid SubStageId:%d,configId:%d", v, id))
			}
			if _, ok := subStageIdMap[v]; ok {
				return errors.New(fmt.Sprintf("[gameConfig] load mainStage error duplicate SubStageId:%d,configId:%d", v, id))
			}
			subStageIdMap[v] = struct{}{}
		}
		for _, dropId := range v.IdleDrop {
			if GetDropCfg(dropId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load mainStage error invalid IdleDrop:%d,configId:%d", dropId, id))
			}
		}
		for _, mailId := range v.MailId {
			if GetMailContentCfg(mailId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load mainStage error invalid MailId:%d,configId:%d", mailId, id))
			}
		}
		if v.Unlock != 0 && GetUnlockCfg(v.Unlock) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load mainStage error invalid Unlock:%d,configId:%d", v.Unlock, id))
		}

		if s.temp1[v.BackStage] != nil {
			s.temp1[v.BackStage].PreStage = v.Id
		}
	}
	monsterWaveMap := make(map[int32]struct{})
	for id, v := range s.temp3 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load subStage error invalid ID:%d", id))
		}
		if len(v.MonsterSpawn) != len(v.MonsterWaveId) {
			return errors.New(fmt.Sprintf("[gameConfig] load subStage error invalid MonsterSpawn/MonsterWaveId,configId:%d", id))
		}
		for index, spawn := range v.MonsterSpawn {
			if len(spawn) != len(v.MonsterWaveId[index]) {
				return errors.New(fmt.Sprintf("[gameConfig] load subStage error invalid MonsterSpawn/MonsterWaveId,configId:%d", id))
			}
		}
		for _, waveList := range v.MonsterWaveId {
			for _, waveId := range waveList {
				if s.temp2[waveId] == nil {
					return errors.New(fmt.Sprintf("[gameConfig] load subStage error invalid MonsterWaveId:%d,configId:%d", waveId, id))
				}
				if _, ok := monsterWaveMap[waveId]; ok {
					return errors.New(fmt.Sprintf("[gameConfig] load subStage error duplicate MonsterWaveId:%d,configId:%d", waveId, id))
				}
				monsterWaveMap[waveId] = struct{}{}
			}
		}
		if len(v.BarrelSpawn) != len(v.BarrelWaveId) {
			return errors.New(fmt.Sprintf("[gameConfig] load subStage error invalid BarrelSpawn/BarrelWaveId,configId:%d", id))
		}
		for _, barrelId := range v.BarrelWaveId {
			if s.temp2[barrelId] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load subStage error invalid BarrelWaveId:%d,configId:%d", barrelId, id))
			}
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monsterWave error invalid ID:%d", id))
		}
		if GetMonsterCfg(v.MonsterId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load monsterWave error invalid MonsterId:%d,configId:%d", v.MonsterId, id))
		}
		if v.MonsterNum <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load monsterWave error invalid MonsterNum:%d,configId:%d", v.MonsterNum, id))
		}
		if v.EachDrop != 0 && GetDropCfg(v.EachDrop) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load monsterWave error invalid EachDrop:%d,configId:%d", v.EachDrop, id))
		}
		if v.FirstDrop != 0 && GetDropCfg(v.FirstDrop) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load monsterWave error invalid FirstDrop:%d,configId:%d", v.FirstDrop, id))
		}
		if v.PrivilegeDrop != 0 && GetDropCfg(v.PrivilegeDrop) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load monsterWave error invalid PrivilegeDrop:%d,configId:%d", v.PrivilegeDrop, id))
		}
	}
	return nil
}

func (s *StageCfgLoader) apply() {
	mainStage.Store(s.temp1)
	monsterWave.Store(s.temp2)
	subStage.Store(s.temp3)
}

var mainStage atomic.Value
var monsterWave atomic.Value
var subStage atomic.Value

type MainStageCfg struct {
	// 关卡id
	Id int32 `json:"id"`
	// 后置关卡
	BackStage int32 `json:"backStage"`
	// 副本id
	InstanceId int32 `json:"instanceId"`
	// 章节id
	ChapterId int32 `json:"chapterId"`
	// 子关卡id
	SubStageId []int32 `json:"subStageId"`
	// 挂机奖励
	IdleDrop []int32 `json:"idleDrop"`
	// 通关邮件
	MailId []int32 `json:"mailId"`
	// 解锁条件
	Unlock int32 `json:"unlock"`
	// 前置关卡
	PreStage int32
}

type MonsterWaveCfg struct {
	// 怪物波次id
	Id int32 `json:"id"`
	// 怪物id
	MonsterId int32 `json:"monsterId"`
	// 怪物数量
	MonsterNum int32 `json:"monsterNum"`
	// 首次掉落
	FirstDrop int32 `json:"firstDrop"`
	// 每次掉落
	EachDrop int32 `json:"eachDrop"`
	// 特权掉落
	PrivilegeDrop int32 `json:"privilegeDrop"`
}

type SubStageCfg struct {
	// 子关卡id
	Id int32 `json:"id"`
	// 房间id
	RoomId []int32 `json:"roomId"`
	// 怪物出生点
	MonsterSpawn [][]string `json:"monsterSpawn"`
	// 怪物波次id
	MonsterWaveId [][]int32 `json:"monsterWaveId"`
	// 木桶出生点
	BarrelSpawn []string `json:"barrelSpawn"`
	// 木桶波次id
	BarrelWaveId []int32 `json:"barrelWaveId"`
}

func GetMainStageCfg(id int32) *MainStageCfg {
	cfgMap := mainStage.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*MainStageCfg)[id]
}

func GetMonsterWaveCfg(id int32) *MonsterWaveCfg {
	cfgMap := monsterWave.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*MonsterWaveCfg)[id]
}

func GetSubStageCfg(id int32) *SubStageCfg {
	cfgMap := subStage.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*SubStageCfg)[id]
}
