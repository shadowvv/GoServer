// File: equipmentInterface.go
// Description: 装备系统服务接口定义
// Author: 木村凉太
// Create Time: 2025.11

package logicCommon

import (
	"github.com/drop/GoServer/server/logic/pb"
)

// EquipmentInterface 装备服务接口
type EquipmentInterface interface {
	// AddEquipment 添加装备（掉落生成）
	AddEquipment(userId int64, equipmentID int32, level int32) (int64, error)

	// EquipEquipment 穿戴装备
	EquipEquipment(userId int64, equipmentOwnID int64, heroOwnID int64) error

	// UnequipEquipment 卸下装备
	UnequipEquipment(userId int64, equipmentOwnID int64) error

	// SwapEquipment 替换装备
	SwapEquipment(userId int64, equipmentOwnID int64, heroOwnID int64) (int64, error)

	// QuickEquip 一键穿戴
	QuickEquip(userId int64, heroOwnIDs []int64) ([]*pb.EquipmentInfo, error)

	// QuickUnequip 一键卸下
	QuickUnequip(userId int64, heroOwnIDs []int64) error

	// LockEquipment 锁定装备
	LockEquipment(userId int64, equipmentOwnID int64) error

	// UnlockEquipment 解锁装备
	UnlockEquipment(userId int64, equipmentOwnID int64) error

	// DecomposeEquipments 批量分解装备
	DecomposeEquipments(userId int64, equipmentOwnIDs []int64) (*pb.EquipmentDecomposeResp, error)

	// GetEquipmentList 获取装备列表
	GetEquipmentList(userId int64) ([]*pb.EquipmentInfo, error)

	// GetEquipmentDetail 获取装备详情
	GetEquipmentDetail(userId int64, equipmentOwnID int64) (*pb.EquipmentDetailInfo, error)

	// StrongEquipment 强化装备
	StrongEquipment(userId int64, equipmentOwnID int64, isUseStone bool) (*pb.EquipmentStrongResp, error)

	// RebirthEquipment 重生装备
	RebirthEquipment(userId int64, equipmentOwnID int64) (*pb.EquipmentRebirthResp, error)
}
