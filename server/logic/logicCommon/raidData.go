package logicCommon

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
)

type PlayerInstanceRaid struct {
	PlayerId          int64
	BattleId          int64
	InstanceID        enum.InstanceId
	CurrentStageId    int32
	CurrentSubStageId int32
	RandomSeed        int64
	BattleEndTime     int64
	SubStageInfo      map[int32]*SubStageData
	SubStageIds       []int32
	MonsterTemplates  map[int64]*MonsterTemplate
	StageInfo         *InstanceStageInfo
	AdventureUniqueId string
	FormationType     int32

	IsOver bool

	TargetTd int64
	IsRobot  bool
}

type SubStageData struct {
	SubStageId  int32
	RoomID      int32
	MonsterInfo map[int32]*MonsterInfo
	ComboSkills []int32
}

type MonsterInfo struct {
	Id           int32
	SpawnId      string
	WaveId       int32
	WaveSequence int32
	MonsterId    int32
	DropItems    []*gameConfig.ItemConfig
	IsDead       int32
}

type MonsterTemplate struct {
	MonsterId   int64
	UnitId      int32
	AtkSpeed    int32
	MoveSpeed   int32
	PatrolRange int32
	AggroRange  int32
	AttackRange int32
	BasicSkill  int32
	AttrInfo    map[int32]int64
	SkillId     []int32
}

type InstanceStageInfo struct {
	IsCycle        int32          `json:"IsCycle"`
	KillMonsterId  []int32        `json:"KillMonsterId"`
	KillMonsterMap map[int32]bool `json:"-"`
}

func NewInstanceStageInfo() *InstanceStageInfo {
	return &InstanceStageInfo{
		IsCycle:        0,
		KillMonsterId:  make([]int32, 0),
		KillMonsterMap: make(map[int32]bool),
	}
}
