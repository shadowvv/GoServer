// 七日试炼配置加载：每日 foremost、进度奖励 trialReward、试炼子任务 trialTask，并建 actID/taskGroup 索引。
package gameConfig

import (
	"errors"
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("trial", &TrialCfgLoader{})
}

type TrialCfgLoader struct {
	temp1 map[int32]*TrialForemostCfg
	temp2 map[int32]*TrialRewardCfg
	temp3 map[int32]*TrialTaskCfg

	idxForemostByAct map[int32][]*TrialForemostCfg
	idxTaskByGroup   map[int32][]*TrialTaskCfg
	idxRewardByAct   map[int32][]*TrialRewardCfg
	idxTaskIDsByAct  map[int32][]int32
	idxActByTask     map[int32]int32
	idxAllActIDs     []int32
}

var _ configLoaderInterface = (*TrialCfgLoader)(nil)

func (s *TrialCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/trial.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*TrialForemostCfg)
	for _, row := range rawData["Foremost"] {
		var v TrialForemostCfg
		v.Id = ParseInt(row["id"])
		v.ActID = ParseInt(row["actID"])
		v.TaskGroup = ParseIntArray(row["taskGroup"])
		v.Value = ParseInt(row["value"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load trial Foremost error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*TrialRewardCfg)
	for _, row := range rawData["Reward"] {
		var v TrialRewardCfg
		v.Id = ParseInt(row["id"])
		v.Value = &ItemConfig{ID: s.getProgressItemIDByActID(ParseInt(row["actID"])), Num: ParseInt64(row["value"])}
		v.Reward = ParseItemArray(row["reward"])
		v.ActID = ParseInt(row["actID"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load trial Reward error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*TrialTaskCfg)
	for _, row := range rawData["Task"] {
		var v TrialTaskCfg
		v.Id = ParseInt(row["id"])
		v.TaskGroup = ParseInt(row["taskGroup"])
		v.Order = ParseInt(row["order"])
		v.TaskId = ParseInt(row["taskId"])
		v.TaskReward = ParseItemArray(row["taskReward"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load trial Task error duplicate ID:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	s.buildIndexes()
	return nil
}

func (s *TrialCfgLoader) getProgressItemIDByActID(actID int32) int32 {
	for _, v := range s.temp1 {
		if v.ActID == actID {
			return v.Value
		}
	}
	return 0
}

func (s *TrialCfgLoader) buildIndexes() {
	s.idxForemostByAct = make(map[int32][]*TrialForemostCfg)
	for _, v := range s.temp1 {
		s.idxForemostByAct[v.ActID] = append(s.idxForemostByAct[v.ActID], v)
	}
	for actID := range s.idxForemostByAct {
		sort.Slice(s.idxForemostByAct[actID], func(i, j int) bool {
			return s.idxForemostByAct[actID][i].Id < s.idxForemostByAct[actID][j].Id
		})
	}

	s.idxTaskByGroup = make(map[int32][]*TrialTaskCfg)
	for _, v := range s.temp3 {
		s.idxTaskByGroup[v.TaskGroup] = append(s.idxTaskByGroup[v.TaskGroup], v)
	}
	for g := range s.idxTaskByGroup {
		sort.Slice(s.idxTaskByGroup[g], func(i, j int) bool {
			return s.idxTaskByGroup[g][i].Id < s.idxTaskByGroup[g][j].Id
		})
	}

	s.idxRewardByAct = make(map[int32][]*TrialRewardCfg)
	for _, v := range s.temp2 {
		s.idxRewardByAct[v.ActID] = append(s.idxRewardByAct[v.ActID], v)
	}
	for actID := range s.idxRewardByAct {
		sort.Slice(s.idxRewardByAct[actID], func(i, j int) bool {
			return s.idxRewardByAct[actID][i].Id < s.idxRewardByAct[actID][j].Id
		})
	}

	s.idxTaskIDsByAct = make(map[int32][]int32)
	s.idxActByTask = make(map[int32]int32)
	for actID, foremosts := range s.idxForemostByAct {
		for _, fc := range foremosts {
			for _, group := range fc.TaskGroup {
				for _, task := range s.idxTaskByGroup[group] {
					s.idxTaskIDsByAct[actID] = append(s.idxTaskIDsByAct[actID], task.Id)
					s.idxActByTask[task.Id] = actID
				}
			}
		}
		sort.Slice(s.idxTaskIDsByAct[actID], func(i, j int) bool {
			return s.idxTaskIDsByAct[actID][i] < s.idxTaskIDsByAct[actID][j]
		})
	}

	actSet := make(map[int32]bool)
	for _, v := range s.temp1 {
		actSet[v.ActID] = true
	}
	s.idxAllActIDs = make([]int32, 0, len(actSet))
	for id := range actSet {
		s.idxAllActIDs = append(s.idxAllActIDs, id)
	}
	sort.Slice(s.idxAllActIDs, func(i, j int) bool { return s.idxAllActIDs[i] < s.idxAllActIDs[j] })
}

func (s *TrialCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] trial Foremost invalid ID:%d", id))
		}
		if v.ActID <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] trial Foremost invalid ActID:%d,configId:%d", v.ActID, id))
		}
		if len(v.TaskGroup) == 0 {
			return errors.New(fmt.Sprintf("[gameConfig] trial Foremost invalid TaskGroup:%v,configId:%d", v.TaskGroup, id))
		}
		if v.Value <= 0 || GetItemCfg(v.Value) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] trial Foremost invalid Value:%d,configId:%d", v.Value, id))
		}
		for _, group := range v.TaskGroup {
			if group <= 0 || len(s.idxTaskByGroup[group]) == 0 {
				return errors.New(fmt.Sprintf("[gameConfig] trial Foremost invalid TaskGroup:%d,configId:%d", group, id))
			}
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] trial Reward invalid ID:%d", id))
		}
		if v.ActID <= 0 || len(s.idxForemostByAct[v.ActID]) == 0 {
			return errors.New(fmt.Sprintf("[gameConfig] trial Reward invalid ActID:%d,configId:%d", v.ActID, id))
		}
		if err := checkTrialItem(v.Value); err != nil {
			return errors.New(fmt.Sprintf("[gameConfig] trial Reward invalid Value configId:%d", id))
		}
		if err := checkTrialItemArray(v.Reward); err != nil {
			return errors.New(fmt.Sprintf("[gameConfig] trial Reward invalid Reward configId:%d", id))
		}
	}
	for id, v := range s.temp3 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] trial Task invalid ID:%d", id))
		}
		if v.TaskGroup <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] trial Task invalid TaskGroup:%d,configId:%d", v.TaskGroup, id))
		}
		if v.TaskId <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] trial Task invalid TaskId:%d,configId:%d", v.TaskId, id))
		}
		if GetCoreCfg(v.TaskId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] trial Task id:%d references unknown Core taskId:%d", id, v.TaskId))
		}
		if err := checkTrialItemArray(v.TaskReward); err != nil {
			return errors.New(fmt.Sprintf("[gameConfig] trial Task invalid TaskReward configId:%d", id))
		}
		if actID := s.idxActByTask[v.Id]; actID <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] trial Task configId:%d not referenced by any Foremost.TaskGroup", id))
		}
	}
	return nil
}

func checkTrialItem(item *ItemConfig) error {
	if item == nil || item.ID <= 0 || item.Num <= 0 || GetItemCfg(item.ID) == nil {
		return errors.New("invalid item")
	}
	return nil
}

func checkTrialItemArray(items []*ItemConfig) error {
	if len(items) == 0 {
		return errors.New("empty item")
	}
	for _, item := range items {
		if err := checkTrialItem(item); err != nil {
			return err
		}
	}
	return nil
}

func (s *TrialCfgLoader) apply() {
	trialForemost.Store(s.temp1)
	trialReward.Store(s.temp2)
	trialTask.Store(s.temp3)
	trialForemostByAct.Store(s.idxForemostByAct)
	trialTaskByGroup.Store(s.idxTaskByGroup)
	trialRewardByAct.Store(s.idxRewardByAct)
	trialTaskIDsByAct.Store(s.idxTaskIDsByAct)
	trialActByTask.Store(s.idxActByTask)
	trialAllActIDs.Store(s.idxAllActIDs)
}

var trialForemost atomic.Value
var trialReward atomic.Value
var trialTask atomic.Value
var trialForemostByAct atomic.Value
var trialTaskByGroup atomic.Value
var trialRewardByAct atomic.Value
var trialTaskIDsByAct atomic.Value
var trialActByTask atomic.Value
var trialAllActIDs atomic.Value

type TrialForemostCfg struct {
	Id        int32   `json:"id"`
	ActID     int32   `json:"actID"`
	TaskGroup []int32 `json:"taskGroup"`
	Value     int32   `json:"value"`
}

type TrialRewardCfg struct {
	Id     int32         `json:"id"`
	Value  *ItemConfig   `json:"value"`
	Reward []*ItemConfig `json:"reward"`
	ActID  int32         `json:"actID"`
}

type TrialTaskCfg struct {
	Id         int32         `json:"id"`
	TaskGroup  int32         `json:"taskGroup"`
	Order      int32         `json:"order"`
	TaskId     int32         `json:"taskId"`
	TaskReward []*ItemConfig `json:"taskReward"`
}

func (m *TrialTaskCfg) GetTaskId() int32 {
	return m.TaskId
}

var _ taskCfgServer = (*TrialTaskCfg)(nil)

func GetTrialForemostCfg(id int32) *TrialForemostCfg {
	cfgMap := trialForemost.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*TrialForemostCfg)[id]
}

func GetTrialRewardCfg(id int32) *TrialRewardCfg {
	cfgMap := trialReward.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*TrialRewardCfg)[id]
}

func GetTrialTaskCfg(id int32) *TrialTaskCfg {
	cfgMap := trialTask.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*TrialTaskCfg)[id]
}

func GetTrialForemostsByActID(actID int32) []*TrialForemostCfg {
	v := trialForemostByAct.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]*TrialForemostCfg)[actID]
}

func GetTrialTasksByGroup(taskGroup int32) []*TrialTaskCfg {
	v := trialTaskByGroup.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]*TrialTaskCfg)[taskGroup]
}

func GetTrialRewardsByActID(actID int32) []*TrialRewardCfg {
	v := trialRewardByAct.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]*TrialRewardCfg)[actID]
}

func GetTrialTaskIDsByActID(actID int32) []int32 {
	v := trialTaskIDsByAct.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]int32)[actID]
}

func GetTrialActIDByTaskID(taskID int32) int32 {
	v := trialActByTask.Load()
	if v == nil {
		return 0
	}
	return v.(map[int32]int32)[taskID]
}

func GetTrialAllActIDs() []int32 {
	v := trialAllActIDs.Load()
	if v == nil {
		return nil
	}
	return v.([]int32)
}
