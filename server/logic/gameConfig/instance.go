package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("instance", &InstanceCfgLoader{})
}

type InstanceCfgLoader struct {
	temp1 map[int32]*InstanceCfg
}

var _ configLoaderInterface = (*InstanceCfgLoader)(nil)

func (s *InstanceCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/instance.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*InstanceCfg)
	for _, row := range rawData["instance"] {
		var v InstanceCfg
		v.Id = ParseInt(row["id"])
		v.InstanceType = ParseInt(row["instanceType"])
		v.StageId = ParseInt(row["stageId"])
		v.SceneId = ParseInt(row["sceneId"])
		v.TicketID = ParseItemArray(row["ticketID"])
		v.RecoveryTicketID = ParseItemArray(row["recoveryTicketID"])
		v.IsKf = ParseInt(row["isKf"])
		v.IsTeamable = ParseInt(row["isTeamable"])
		v.PlayerLimit = ParseIntArray(row["playerLimit"])
		v.SkillValue = ParseIntArray(row["skillValue"])
		v.IsSweep = ParseInt(row["isSweep"])
		v.CanSkip = ParseInt(row["canSkip"])
		v.SkipTime = ParseInt(row["skipTime"])
		v.CanSpeed = ParseInt(row["canSpeed"])
		v.CanControl = ParseInt(row["canControl"])
		v.Revival = ParseInt(row["revival"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load instance error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *InstanceCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load instance error invalid ID:%d", id))
		}
		if !enum.IsValidInstanceType(v.InstanceType) {
			return errors.New(fmt.Sprintf("[gameConfig] load instance error invalid instanceType:%d,configId:%d", v.InstanceType, id))
		}
		// 副本类型判断参数
		switch v.InstanceType {
		case int32(enum.InstanceType_MAIN):
			if v.Id != int32(enum.MAIN_INSTANCE_ID) {
				return errors.New(fmt.Sprintf("[gameConfig] load instance invalid main instance id error instanceType:%d,configId:%d", v.InstanceType, id))
			}
			if GetMainStageCfg(v.StageId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load instance error invalid stageId:%d,configId:%d", v.StageId, id))
			}
		case int32(enum.InstanceType_TOWER):

		}
		if v.SceneId != 0 && GetSceneCfg(v.SceneId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load instance error invalid sceneId:%d,configId:%d", v.SceneId, id))
		}
		for _, item := range v.TicketID {
			if GetItemCfg(item.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load instance error invalid ticketID configId:%d,item not exist itemId:%d", id, item.ID))
			}
		}
		if v.RecoveryTicketID != nil {
			find := false
			for _, t1 := range v.RecoveryTicketID {
				for _, t2 := range v.TicketID {
					if t1.ID == t2.ID {
						find = true
						break
					}
				}
			}
			if !find {
				return errors.New(fmt.Sprintf("[gameConfig] load instance error invalid recoveryTicketID,configId:%d", id))
			}
		}
		if len(v.PlayerLimit) != 0 {
			if len(v.PlayerLimit) != 2 || v.PlayerLimit[0] <= 0 || v.PlayerLimit[1] <= 0 || v.PlayerLimit[0] > v.PlayerLimit[1] {
				return errors.New(fmt.Sprintf("[gameConfig] load instance error invalid playerLimit:%v,configId:%d", v.PlayerLimit, id))
			}
		}
		//TODO:副本技能判断
	}
	return nil
}

func (s *InstanceCfgLoader) apply() {
	instance.Store(s.temp1)
}

var instance atomic.Value

type InstanceCfg struct {
	// 副本ID
	Id int32 `json:"id"`
	// 副本类型
	InstanceType int32 `json:"instanceType"`
	// 初始关卡id
	StageId int32 `json:"stageId"`
	// 场景id
	SceneId int32 `json:"sceneId"`
	// 副本消耗门票
	TicketID []*ItemConfig `json:"ticketID"`
	// 副本恢复门票
	RecoveryTicketID []*ItemConfig `json:"recoveryTicketID"`
	// 是否跨服
	IsKf int32 `json:"isKf"`
	// 是否可以组队
	IsTeamable int32 `json:"isTeamable"`
	// 人数限制
	PlayerLimit []int32 `json:"playerLimit"`
	// 副本技能
	SkillValue []int32 `json:"skillValue"`
	// 能否副本扫荡
	IsSweep int32 `json:"isSweep"`
	// 是否可跳过
	CanSkip int32 `json:"canSkip"`
	// 跳过等待时间
	SkipTime int32 `json:"skipTime"`
	// 是否可加速
	CanSpeed int32 `json:"canSpeed"`
	// 是否可手动操作
	CanControl int32 `json:"canControl"`
	// 是否复活
	Revival int32 `json:"revival"`
}

func GetInstanceCfg(id int32) *InstanceCfg {
	cfgMap := instance.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*InstanceCfg)[id]
}

func GetAllInstance() map[int32]*InstanceCfg {
	cfgMap := instance.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*InstanceCfg)
}

func GetArenaInstanceCfg() *InstanceCfg {
	return GetInstanceCfg(int32(enum.ARENA_INSTANCE_ID))
}
