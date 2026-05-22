package model

import (
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

type LoopBoxEntity struct {
	UserID      int64               `gorm:"column:user_Id;primaryKey"` //用户Id
	SystemEx    int32               `gorm:"column:system_ex"`          // 系统经验
	SystemLevel int32               `gorm:"column:system_level"`       // 系统等级
	SystemPoint int32               `gorm:"column:system_point"`       // 系统积分
	LoopId      int32               `gorm:"column:loop_id"`            // 当前循环id
	BoxList     tool.JSONInt32Slice `gorm:"column:box_list;type:json"` // 下标0-4 表示用户拥有1-5某个箱子多少个
}

var _ logicCommon.PlayerModelInterface = (*LoopBoxModel)(nil)

func (u *LoopBoxEntity) TableName() string {
	return "loop_box"
}

type LoopBoxModel struct {
	UserId        int64
	LoopBoxEntity *LoopBoxEntity
	Changed       map[string]interface{}
}

func NewLoopBoxModel(userId int64, entity *LoopBoxEntity) *LoopBoxModel {
	return &LoopBoxModel{
		UserId:        userId,
		LoopBoxEntity: entity,
		Changed:       make(map[string]interface{}),
	}
}

func CreatLoopBoxModel(userId int64) (*LoopBoxModel, error) {
	entity := &LoopBoxEntity{
		UserID:      userId,
		SystemEx:    0,
		SystemLevel: 1,
		SystemPoint: 0,
		LoopId:      1,
		BoxList:     tool.JSONInt32Slice{0, 0, 0, 0, 0},
	}
	err := easyDB.CreatePlayerEntity(entity)
	if err != nil {
		return nil, err
	}
	return NewLoopBoxModel(userId, entity), nil
}

func (l *LoopBoxModel) SaveModelToDB() {
	if l.Changed != nil || len(l.Changed) > 0 {
		easyDB.UpdatePlayerEntity[LoopBoxEntity](l.LoopBoxEntity, l.Changed, l.UserId)
	}
	l.Changed = make(map[string]interface{})
}

func (l *LoopBoxModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {

}

func (l *LoopBoxModel) UpdateSystemEx(ex int32) {
	l.LoopBoxEntity.SystemEx = ex
	l.Changed["system_ex"] = ex
}

func (l *LoopBoxModel) UpdateSystemLevel(level int32) {
	l.LoopBoxEntity.SystemLevel = level
	l.Changed["system_level"] = level
}

func (l *LoopBoxModel) UpdateSystemPoint(point int32) {
	l.LoopBoxEntity.SystemPoint = point
	l.Changed["system_point"] = point
}

func (l *LoopBoxModel) UpdateLoopId(id int32) {
	l.LoopBoxEntity.LoopId = id
	l.Changed["loop_id"] = id
}

func (l *LoopBoxModel) UpdateBoxList(list tool.JSONInt32Slice) {
	l.LoopBoxEntity.BoxList = list
	l.Changed["box_list"] = list
}

func LoadLoopBoxModel(userId int64) (*LoopBoxModel, error) {
	entity := &LoopBoxEntity{}
	row, err := easyDB.GetPlayerEntitiesByWhere[LoopBoxEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return NewLoopBoxModel(userId, entity), err
	}
	if len(row) <= 0 {
		return CreatLoopBoxModel(userId)
	}
	entity = row[0]
	return NewLoopBoxModel(userId, entity), nil
}

func (l *LoopBoxModel) AddLoopBox(boxId, num int32) {
	l.LoopBoxEntity.BoxList[boxId-1] += num
	l.UpdateBoxList(l.LoopBoxEntity.BoxList)
}
