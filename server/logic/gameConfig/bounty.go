package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("bounty", &BountyCfgLoader{})
}

var bountyList atomic.Value // []*BountyBaseCfg

type BountyCfgLoader struct {
	temp1 map[int32]*BountyBaseCfg
	temp2 map[int32]*BountyTaskCfg
}

var _ configLoaderInterface = (*BountyCfgLoader)(nil)

func (s *BountyCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/bounty.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*BountyBaseCfg)
	for _, row := range rawData["bountyBase"] {
		var v BountyBaseCfg
		v.Id = ParseInt(row["id"])
		v.ActivityId = ParseInt(row["activityId"])
		v.Drop = ParseItemArray(row["drop"])
		v.Duration = ParseInt(row["duration"])
		v.UnlockId = ParseIntArray(row["unlockId"])
		v.BanUnlockid = ParseIntArray(row["banUnlockid"])
		v.TaskId = ParseIntArray(row["taskId"])
		v.ChainBounty = ParseInt(row["chainBounty"])
		v.PreviousBounty = ParseInt(row["previousBounty"])
		v.NextBounty = ParseInt(row["nextBounty"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load bountyBase error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*BountyTaskCfg)
	for _, row := range rawData["bountyTask"] {
		var v BountyTaskCfg
		v.Id = ParseInt(row["id"])
		v.TaskId = ParseInt(row["taskId"])
		v.TaskReward = ParseItemArray(row["taskReward"])
		v.FollowTasks = ParseInt(row["followTasks"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load bountyTask error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	return nil
}

func (s *BountyCfgLoader) checkData() error {
	taskIds := make(map[int32]int32)
	for id, v := range s.temp1 {
		for _, value := range v.TaskId {
			taskIds[value]++
			if s.temp2[value] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load bountyBase error duplicate ID:%d taskId not found in bountyTask", value))
			}
			if taskIds[value] >= 2 {
				return errors.New(fmt.Sprintf("[gameConfig] load bountyBase error duplicate ID:%d taskId is used", value))
			}
		}
		if v.ChainBounty != 0 {
			if v.PreviousBounty == 0 && v.NextBounty == 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load bountyBase error duplicate ID:%d ChainBounty or other is zero", v.Id))
			}
			if v.NextBounty == v.Id {
				return errors.New(fmt.Sprintf("[gameConfig] load bountyBase error duplicate ID:%d", v.Id))
			}
		}
		if v.TaskId == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load bountyBase error duplicate ID:%d taskId is nil", v.Id))
		}
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load bountyBase error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp2 {
		if GetCoreCfg(v.TaskId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load bountyTask error duplicate ID:%d taskId:%d not in taskCore", id, v.TaskId))
		}
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load bountyTask error invalid ID:%d", id))
		}
	}
	return nil
}

func (s *BountyCfgLoader) apply() {
	bountyBase.Store(s.temp1)
	bountyTask.Store(s.temp2)
	InitBounty()
}

var bountyBase atomic.Value
var bountyTask atomic.Value

type BountyBaseCfg struct {
	// 悬赏令id
	Id int32 `json:"id"`
	// 活动id
	ActivityId int32 `json:"activityId"`
	// 奖励掉落
	Drop []*ItemConfig `json:"drop"`
	// 持续时间
	Duration int32 `json:"duration"`
	// 解锁条件
	UnlockId []int32 `json:"unlockId"`
	// 禁止解锁条件
	BanUnlockid []int32 `json:"banUnlockid"`
	// 任务id
	TaskId []int32 `json:"taskId"`
	// 链式悬赏令类型
	ChainBounty int32 `json:"chainBounty"`
	// 上一个悬赏令
	PreviousBounty int32 `json:"previousBounty"`
	// 下一个悬赏令
	NextBounty int32 `json:"nextBounty"`
	// 备注
	BountyNotes string `json:"bountyNotes"`
}

type BountyTaskCfg struct {
	// 悬赏令任务id
	Id int32 `json:"id"`
	// 任务id
	TaskId int32 `json:"taskId"`
	// 任务奖励
	TaskReward []*ItemConfig `json:"taskReward"`
	// 后置任务
	FollowTasks int32 `json:"followTasks"`
}

func GetBountyBaseCfg(id int32) *BountyBaseCfg {
	cfgMap := bountyBase.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*BountyBaseCfg)[id]
}

func (m *BountyTaskCfg) GetTaskId() int32 {
	return m.TaskId
}

var _ taskCfgServer = (*BountyTaskCfg)(nil)

func GetBountyTaskCfg(id int32) *BountyTaskCfg {
	cfgMap := bountyTask.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*BountyTaskCfg)[id]
}

func GetBountyList() []*BountyBaseCfg {
	v := bountyList.Load()
	if v == nil {
		return nil
	}
	return v.([]*BountyBaseCfg)
}

func InitBounty() {
	cfgMap := bountyBase.Load()
	allCfg := cfgMap.(map[int32]*BountyBaseCfg)

	list := make([]*BountyBaseCfg, 0, len(allCfg))
	for _, v := range allCfg {
		list = append(list, v)
	}
	bountyList.Store(list)
}
