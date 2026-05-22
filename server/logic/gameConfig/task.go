package gameConfig

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("task", &TaskCfgLoader{})
}

var sideTaskMap atomic.Value // map[int32][]*SecondaryCfg
var passTaskMap atomic.Value // map[int32][]*PassCfg
var actTaskMap atomic.Value

type taskCfgServer interface {
	GetTaskId() int32
}

type TaskCfgLoader struct {
	temp1 map[int32]*DailyCfg
	temp2 map[int32]*CoreCfg
	temp3 map[int32]*MainCfg
	temp4 map[int32]*SecondaryCfg
	temp5 map[int32]*DailyAwardsCfg
	temp6 map[int32]*PassCfg
	temp7 map[int32]*ActCfg
}

var _ configLoaderInterface = (*TaskCfgLoader)(nil)

func (s *TaskCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/task.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*DailyCfg)
	for _, row := range rawData["Daily"] {
		var v DailyCfg
		v.Id = ParseInt(row["id"])
		v.TaskId = ParseInt(row["taskId"])
		v.TaskAttribution = ParseInt(row["taskAttribution"])
		v.TaskReward = ParseItemArray(row["taskReward"])
		v.Unlock = ParseInt(row["unlock"])
		v.UnlockStop = ParseInt(row["unlockStop"])
		v.Duration = ParseInt(row["duration"])
		v.Refresh = ParseInt(row["refresh"])
		v.Item = ParseItemArray(row["item"])
		v.ActId = ParseIntArray(row["actId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load Daily error duplicate ID:%d", v.TaskId))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*CoreCfg)
	for _, row := range rawData["Core"] {
		var v CoreCfg
		v.Id = ParseInt(row["id"])
		v.TaskType = ParseInt(row["taskType"])
		v.TaskPara = ParseIntArray(row["taskPara"])
		v.TaskNum = ParseInt(row["taskNum"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load Core error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*MainCfg)
	for _, row := range rawData["Main"] {
		var v MainCfg
		v.Id = ParseInt(row["id"])
		v.TaskId = ParseInt(row["taskId"])
		v.TaskAttribution = ParseInt(row["taskAttribution"])
		v.TaskReward = ParseItemArray(row["taskReward"])
		v.FollowTasks = ParseInt(row["followTasks"])
		v.Unlock = ParseInt(row["unlock"])
		v.Num = ParseInt(row["num"])
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load Main error duplicate ID:%d", v.TaskId))
		}
		s.temp3[v.Id] = &v
	}

	s.temp4 = make(map[int32]*SecondaryCfg)
	for _, row := range rawData["Secondary"] {
		var v SecondaryCfg
		v.Id = ParseInt(row["id"])
		v.TaskId = ParseInt(row["taskId"])
		v.TaskAttribution = ParseInt(row["taskAttribution"])
		v.TaskGroup = ParseInt(row["taskGroup"])
		v.TaskReward = ParseItemArray(row["taskReward"])
		v.PreTasks = ParseInt(row["preTasks"])
		v.FollowTasks = ParseInt(row["followTasks"])
		v.Unlock = ParseInt(row["unlock"])
		if v.Id <= 0 {
			continue
		}
		if s.temp4[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load Secondary error duplicate ID:%d", v.TaskId))
		}
		s.temp4[v.Id] = &v
	}

	s.temp5 = make(map[int32]*DailyAwardsCfg)
	for _, row := range rawData["DailyAwards"] {
		var v DailyAwardsCfg
		v.Id = ParseInt(row["id"])
		v.Type = ParseInt(row["type"])
		v.Point = ParseInt(row["point"])
		v.DropId = ParseInt(row["dropId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp5[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load DailyAwards error duplicate ID:%d", v.Id))
		}
		s.temp5[v.Id] = &v
	}

	s.temp6 = make(map[int32]*PassCfg)
	for _, row := range rawData["Pass"] {
		var v PassCfg
		v.Id = ParseInt(row["id"])
		v.PassId = ParseInt(row["passId"])
		v.TaskId = ParseInt(row["taskId"])
		v.TaskAttribution = ParseInt(row["taskAttribution"])
		v.TaskReward = ParseItemArray(row["taskReward"])
		v.Auto = ParseInt(row["auto"])
		v.Repeat = ParseInt(row["repeat"])
		if v.Id <= 0 {
			continue
		}
		if s.temp6[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load PassCard error duplicate ID:%d", v.TaskId))
		}
		s.temp6[v.Id] = &v
	}

	s.temp7 = make(map[int32]*ActCfg)
	for _, row := range rawData["Act"] {
		var v ActCfg
		v.Id = ParseInt(row["id"])
		v.ActId = ParseInt(row["actId"])
		v.TaskId = ParseInt(row["taskId"])
		v.TaskReward = ParseItemArray(row["taskReward"])
		v.Reflect = ParseInt(row["reflect"])
		if v.Id <= 0 {
			continue
		}
		if s.temp7[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load Act error duplicate ID:%d", v.Id))
		}
		s.temp7[v.Id] = &v
	}

	return nil
}

func (s *TaskCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if s.temp2[v.TaskId] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load Daily error nil ID:%d", id))
		}
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load Daily error invalid ID:%d", id))
		}
		if len(v.ActId) != len(v.Item) {
			return errors.New(fmt.Sprintf("[gameConfig] load Daily error invalid ActId length Not equal Item length ID:%d", id))
		}
	}
	for id, v := range s.temp2 {
		if v.TaskType < enum.ObjectiveTypeKillAnyMonsterHowMany || v.TaskType > enum.ObjectiveTypeWearHowManyEquipmentLevel {
			return errors.New(fmt.Sprintf("[gameConfig] load Core error invalid TaskPara limit ID:%d", id))
		}
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load Core error invalid ID:%d", id))
		}
		if v.TaskNum <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load Core error invalid TaskNum ID:%d", id))
		}

	}
	for id, v := range s.temp3 {
		if s.temp2[v.TaskId] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load Main error nil ID:%d", id))
		}
		if v.FollowTasks != 0 {
			if s.temp3[v.Id] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load Main error duplicate ID:%d", id))
			}
		}
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load Main error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp4 {
		if s.temp2[v.TaskId] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load Secondary error nil ID:%d", id))
		}
		if v.PreTasks != 0 {
			if s.temp4[v.PreTasks] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load Secondary error invalid PreTasks ID:%d", id))
			}
		}
		if v.FollowTasks != 0 {
			if s.temp4[v.FollowTasks] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load Secondary error invalid FollowTasks ID:%d", id))
			}
		}
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load Secondary error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp6 {
		if s.temp2[v.TaskId] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load pass task core error nil ID:%d", id))
		}
		if GetBasePassCfg(v.PassId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load pass card id error invalid ID:%d", id))
		}
		for _, value := range v.TaskReward {
			if GetItemCfg(value.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load pass task item error invalid ID:%d", id))
			}
		}
		if v.Repeat < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load pass task repeat error invalid ID:%d", id))
		}
	}
	for id, v := range s.temp7 {
		if s.temp2[v.TaskId] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load act task core error nil ID:%d", id))
		}
		if v.ActId <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load act task actId error invalid ID:%d", id))
		}
		for _, value := range v.TaskReward {
			if GetItemCfg(value.ID) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load act task item error invalid ID:%d", id))
			}
		}
		if v.Reflect < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load act task reflect error invalid ID:%d", id))
		}
	}
	err := checkTackTypePara()
	if err != nil {
		return err
	}
	return nil
}

func (s *TaskCfgLoader) apply() {
	Daily.Store(s.temp1)
	Core.Store(s.temp2)
	Main.Store(s.temp3)
	Secondary.Store(s.temp4)
	DailyAwards.Store(s.temp5)
	PassCard.Store(s.temp6)
	Act.Store(s.temp7)
	InitSideTask()
	InitPassTask()
	InitActTask()
}

var Daily atomic.Value
var Core atomic.Value
var Main atomic.Value
var Secondary atomic.Value
var DailyAwards atomic.Value
var PassCard atomic.Value
var Act atomic.Value

type CoreCfg struct {
	// 序号id
	Id int32 `json:"id"`
	// 任务类型
	TaskType int32 `json:"taskType"`
	// 任务参数
	TaskPara []int32 `json:"taskPara"`
	// 需求数量
	TaskNum int32 `json:"taskNum"`
}

type DailyCfg struct {
	// 每日任务id
	Id int32 `json:"id"`
	// 任务id
	TaskId int32 `json:"taskId"`
	// 任务归属
	TaskAttribution int32 `json:"taskAttribution"`
	// 任务奖励
	TaskReward []*ItemConfig `json:"taskReward"`
	// 特殊掉落
	Item []*ItemConfig `json:"item"`
	// 活动id
	ActId []int32 `json:"actId"`
	// 解锁条件
	Unlock int32 `json:"unlock"`
	// 解锁失效
	UnlockStop int32 `json:"unlockStop"`
	// 持续时间
	Duration int32 `json:"duration"`
	// 刷新类型
	Refresh int32 `json:"refresh"`
}

type DailyAwardsCfg struct {
	// id
	Id int32 `json:"id"`
	// 类型
	Type int32 `json:"type"`
	// 活跃积分
	Point int32 `json:"point"`
	// 奖励掉落id
	DropId int32 `json:"dropId"`
}

type MainCfg struct {
	// 主线任务id
	Id int32 `json:"id"`
	// 任务id
	TaskId int32 `json:"taskId"`
	// 任务归属
	TaskAttribution int32 `json:"taskAttribution"`
	// 任务奖励
	TaskReward []*ItemConfig `json:"taskReward"`
	// 后置任务
	FollowTasks int32 `json:"followTasks"`
	// 解锁条件
	Unlock int32 `json:"unlock"`
	// 主线任务序号
	Num int32 `json:"num"`
}

type PassCfg struct {
	//主线任务Id
	Id int32 `json:"id"`
	//通行证Id
	PassId int32 `json:"passId"`
	//任务id
	TaskId int32 `json:"taskId"`
	// 任务归属
	TaskAttribution int32 `json:"taskAttribution"`
	// 任务奖励
	TaskReward []*ItemConfig `json:"taskReward"`
	// 自动领奖
	Auto int32 `json:"auto"`
	// 循环做
	Repeat int32 `json:"repeat"`
}

type ActCfg struct {
	Id         int32         `json:"id"`
	ActId      int32         `json:"actId"`
	TaskId     int32         `json:"taskId"`
	TaskReward []*ItemConfig `json:"taskReward"`
	Reflect    int32         `json:"reflect"`
}

func (m *MainCfg) GetTaskId() int32 {
	return m.TaskId
}

var _ taskCfgServer = (*MainCfg)(nil)

type SecondaryCfg struct {
	// 支线任务id
	Id int32 `json:"id"`
	// 任务id
	TaskId int32 `json:"taskId"`
	// 任务归属
	TaskAttribution int32 `json:"taskAttribution"`
	// 任务组
	TaskGroup int32 `json:"taskGroup"`
	// 任务奖励
	TaskReward []*ItemConfig `json:"taskReward"`
	// 前置任务
	PreTasks int32 `json:"preTasks"`
	// 后置任务
	FollowTasks int32 `json:"followTasks"`
	// 解锁条件
	Unlock int32 `json:"unlock"`
}

func (m *DailyCfg) GetTaskId() int32 {
	return m.TaskId
}

var _ taskCfgServer = (*DailyCfg)(nil)

func GetDailyCfg(id int32) *DailyCfg {
	cfgMap := Daily.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*DailyCfg)[id]
}

func GetCoreCfg(id int32) *CoreCfg {
	cfgMap := Core.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*CoreCfg)[id]
}

func GetDailyAwardsCfg(id int32) *DailyAwardsCfg {
	cfgMap := DailyAwards.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*DailyAwardsCfg)[id]
}

func GetMainCfg(id int32) *MainCfg {
	cfgMap := Main.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*MainCfg)[id]
}

func (m *SecondaryCfg) GetTaskId() int32 {
	return m.TaskId
}

var _ taskCfgServer = (*SecondaryCfg)(nil)

func GetSecondaryCfg(id int32) *SecondaryCfg {
	cfgMap := Secondary.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*SecondaryCfg)[id]
}

func GetTaskRewardsByAttribution(attribution int32, taskId int32) []*ItemConfig {
	switch attribution {
	case enum.TaskAffiliationDaily:
		if cfg := GetDailyCfg(taskId); cfg != nil {
			return cfg.TaskReward
		}
	case enum.TaskAffiliationMain:
		if cfg := GetMainCfg(taskId); cfg != nil {
			return cfg.TaskReward
		}
	case enum.TaskAffiliationSide:
		if cfg := GetSecondaryCfg(taskId); cfg != nil {
			return cfg.TaskReward
		}
	case enum.TaskAffiliationAct:
		if cfg := GetActTaskCfg(taskId); cfg != nil {
			return cfg.TaskReward
		}
	}
	return nil
}

func GetTaskTypeByAttribution(attribution int32, taskId int32) int32 {
	switch attribution {
	case enum.TaskAffiliationDaily:
		if cfg := GetDailyCfg(taskId); cfg != nil {
			if core := GetCoreCfg(cfg.TaskId); core != nil {
				return core.TaskType
			}
		}
	case enum.TaskAffiliationMain:
		if cfg := GetMainCfg(taskId); cfg != nil {
			if core := GetCoreCfg(cfg.TaskId); core != nil {
				return core.TaskType
			}
		}
	case enum.TaskAffiliationSide:
		if cfg := GetSecondaryCfg(taskId); cfg != nil {
			if core := GetCoreCfg(cfg.TaskId); core != nil {
				return core.TaskType
			}
		}
	case enum.TaskAffiliationAct:
		if cfg := GetActTaskCfg(taskId); cfg != nil {
			if core := GetCoreCfg(cfg.TaskId); core != nil {
				return core.TaskType
			}
		}
	}
	return 0
}

func GetDailyTaskNum() int32 {
	cfgMap := Daily.Load()
	if cfgMap == nil {
		return 0
	}
	return int32(len(cfgMap.(map[int32]*DailyCfg)))
}

func GetSideTaskGroupMaxNum() int32 {
	cfgMap := Secondary.Load()
	if cfgMap == nil {
		return 0
	}
	maxGroup := int32(0)
	for _, v := range cfgMap.(map[int32]*SecondaryCfg) {
		if v.TaskGroup > maxGroup {
			maxGroup = v.TaskGroup
		}
	}
	return maxGroup
}

func GetSideTaskIdByGroup(group int32) int32 {
	cfgMap := Secondary.Load()
	if cfgMap == nil {
		return 0
	}
	for _, v := range cfgMap.(map[int32]*SecondaryCfg) {
		if v.TaskGroup == group && v.Id == 1 {
			return v.TaskId
		}
	}
	return 0
}

func GetAllDailyTaskIDs() []int32 {
	cfgMap := Daily.Load()
	if cfgMap == nil {
		return nil
	}
	ids := make([]int32, 0)
	for id, _ := range cfgMap.(map[int32]*DailyCfg) {
		idCopy := id
		ids = append(ids, idCopy)
	}
	return ids
}

func GetSideTaskMap() map[int32][]*SecondaryCfg {
	v := sideTaskMap.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]*SecondaryCfg)
}

func GetPassTaskMap() map[int32][]*PassCfg {
	v := passTaskMap.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]*PassCfg)
}

func InitSideTask() {
	cfgMap := Secondary.Load()
	newMap := make(map[int32][]*SecondaryCfg)
	for _, v := range cfgMap.(map[int32]*SecondaryCfg) {
		if v.PreTasks == 0 {
			sideTask := make([]*SecondaryCfg, 0)
			sideTask = append(sideTask, v)
			cfg := v
			for cfg.FollowTasks != 0 {
				cfg = cfgMap.(map[int32]*SecondaryCfg)[cfg.FollowTasks]
				sideTask = append(sideTask, cfg)
			}
			newMap[v.TaskGroup] = sideTask
		}
	}
	sideTaskMap.Store(newMap)
}

func InitPassTask() {
	cfgMap := PassCard.Load()
	newMap := make(map[int32][]*PassCfg)
	for _, v := range cfgMap.(map[int32]*PassCfg) {
		newMap[v.PassId] = append(newMap[v.PassId], v)
	}
	passTaskMap.Store(newMap)
}

func InitActTask() {
	cfgMap := Act.Load()
	newMap := make(map[int32][]*ActCfg)
	if cfgMap != nil {
		for _, v := range cfgMap.(map[int32]*ActCfg) {
			newMap[v.ActId] = append(newMap[v.ActId], v)
		}
	}
	for _, cfgs := range newMap {
		sort.Slice(cfgs, func(i, j int) bool { return cfgs[i].Id < cfgs[j].Id })
	}
	actTaskMap.Store(newMap)
}

func GetPassTask(taskId int32) *PassCfg {
	cfgMap := PassCard.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*PassCfg)[taskId]
}

func GetActTaskCfg(taskId int32) *ActCfg {
	cfgMap := Act.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*ActCfg)[taskId]
}

func GetActTaskCfgsByActID(actId int32) []*ActCfg {
	v := actTaskMap.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32][]*ActCfg)[actId]
}

func (m *ActCfg) GetTaskId() int32 {
	return m.TaskId
}

var _ taskCfgServer = (*ActCfg)(nil)

func (m *PassCfg) GetTaskId() int32 {
	return m.TaskId
}

var _ taskCfgServer = (*PassCfg)(nil)

func GetCoreTaskId(attr, id int32) (int32, error) {
	var cfg taskCfgServer
	if attr == enum.TaskAffiliationMain {
		cfg = GetMainCfg(id)
	} else if attr == enum.TaskAffiliationDaily {
		cfg = GetDailyCfg(id)
	} else if attr == enum.TaskAffiliationSide {
		cfg = GetSecondaryCfg(id)
	} else if attr == enum.TaskAffiliationBounty {
		cfg = GetBountyTaskCfg(id)
	} else if attr == enum.TaskAffiliationPassCard {
		cfg = GetPassTask(id)
	} else if attr == enum.TaskAffiliationTrial {
		cfg = GetTrialTaskCfg(id)
	} else if attr == enum.TaskAffiliationCityAge {
		return id, nil
	} else if attr == enum.TaskAffiliationAct {
		cfg = GetActTaskCfg(id)
	}

	// 防止配置表变更导致 cfg 为 nil 时 panic
	// 注意：即使 cfg 接口变量不为 nil，内部持有的指针值可能为 nil
	// 使用反射检查指针是否为 nil（通用方案，适用于所有配置类型）
	if cfg == nil || (reflect.ValueOf(cfg).Kind() == reflect.Ptr && reflect.ValueOf(cfg).IsNil()) {
		return 0, errors.New(fmt.Sprintf("[GetCoreTaskId] config is nil, attr=%d, id=%d", attr, id))
	}

	return cfg.GetTaskId(), nil
}

func checkTackTypePara() error {
	var err error

	cfgMap := Core.Load()
	if cfgMap == nil {
		return errors.New("[gameConfig] checkTackTypePara Core config is nil")
	}

	for _, coreCfg := range cfgMap.(map[int32]*CoreCfg) {
		switch coreCfg.TaskType {
		case enum.ObjectiveTypeKillAnyMonsterHowMany:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			} else {
				if coreCfg.TaskPara[0] != 0 {
					if coreCfg.TaskPara[0] < 1 || coreCfg.TaskPara[0] > 5 {
						err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara value ID:%d", coreCfg.Id))
					}
				}
			}
		case enum.ObjectiveTypeKillWhatMonsterHowMany:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeWhereKillWhatMonsterHowMany:
			if len(coreCfg.TaskPara) != 2 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeGetTypeOrQualityItemsHowMany:
			if len(coreCfg.TaskPara) != 2 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeGetWhatItemsHowMany:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeAnyHeroReachWhatLevel:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypePassWhatMainLevel:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeHowManyHeroReachWhatLevel:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeTowerChallengePassWhatLevel:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeHowManyHeroReachWhatStar:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeWhatBuildLevelUpWhat:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeWhatDispatchMapUnlockWhatStage:
			if len(coreCfg.TaskPara) != 2 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeLoopBoxSystemLevelReachWhat:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeAccessorySystemLevelReachWhat:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeHowManyHeroReachWhatPotential:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			} else if coreCfg.TaskPara[0] <= 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara value ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeHowManyEquipStrongReachWhatLevel:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeHowManyPetReachWhatLevel:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeJoinAlliance:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeAdventureParticipateHowMany:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeHowManyPetReachWhatStar:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeEquipmentForgeHowMany:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeWearHowManyEquipmentQuality:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeArenaScoreReachWhat:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeAdChestOpenHowMany:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeMainTaskPassWhatNum:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeStoneClassTotalLevelReachWhat:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeGloryArenaChallengeHowMany:
			if len(coreCfg.TaskPara) != 0 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeDungeonParticipateHowMany, enum.ObjectiveTypeDungeonParticipateHowManyCumulative:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeDungeonPassWhatStage:
			if len(coreCfg.TaskPara) != 2 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		case enum.ObjectiveTypeWearHowManyEquipmentLevel:
			if len(coreCfg.TaskPara) != 1 {
				err = errors.New(fmt.Sprintf("[gameConfig] check Core error invalid TaskPara length ID:%d", coreCfg.Id))
			}
		}
	}
	return err
}
