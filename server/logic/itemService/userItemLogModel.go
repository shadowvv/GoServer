package itemService

import (
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

// UserItemLog 用户物品流水记录
type UserItemLog struct {
	Uid  int64  `json:"uid"`  // 用户 ID
	T    int64  `json:"t"`    // 添加时间（毫秒时间戳）
	It   int32  `json:"it"`   // 物品类型
	Id   int32  `json:"id"`   // 物品 ID
	Ft   int32  `json:"ft"`   // 来源类型
	Fv   int32  `json:"fv"`   // 来源值
	Ext  int64  `json:"ext"`  // 额外值
	Init string `json:"init"` // 初始值
	Chg  string `json:"chg"`  // 变化值
	Fina string `json:"fina"` // 最终值
}

// ToEntity 转换为数据库实体
func (l *UserItemLog) ToEntity() map[string]interface{} {
	return map[string]interface{}{
		"uid":  l.Uid,
		"t":    l.T,
		"it":   l.It,
		"id":   l.Id,
		"ft":   l.Ft,
		"fv":   l.Fv,
		"ext":  l.Ext,
		"init": l.Init,
		"chg":  l.Chg,
		"fina": l.Fina,
	}
}

// ReportUserItemChange 上报用户物品变更日志
// 参数说明：
//   - uid: 用户 ID
//   - itemType: 物品类型（枚举值，如 enum.ITEM_TYPE_ITEM）
//   - itemId: 物品 ID
//   - fromType: 来源类型（枚举值，如 enum.ItemChangeReason）
//   - fromValue: 来源值（如副本 ID、活动 ID 等）
//   - extra: 额外值（如 VIP 等级、特殊标记等）
//   - initVal: 初始值 JSON（变化前的完整状态）
//   - chgVal: 变化值 JSON（变化的具体数值）
//   - finaVal: 最终值 JSON（变化后的完整状态）
func ReportUserItemChange(uid int64, itemType, itemId, fromType int32, fromValue, extra int64, initVal, chgVal, finaVal string) error {
	log := &UserItemLog{
		Uid:  uid,
		T:    tool.UnixNowMilli(),
		It:   itemType,
		Id:   itemId,
		Ft:   fromType,
		Fv:   int32(fromValue),
		Ext:  extra,
		Init: initVal,
		Chg:  chgVal,
		Fina: finaVal,
	}

	entity := log.ToEntity()
	entity["add_time"] = log.T // 用于生成表名

	return easyDB.LogCreatEntity(entity, easyDB.ITEM_LOG)
}

// ReportUserItemChanges 批量上报用户物品变更日志
func ReportUserItemChanges(logs []*UserItemLog) error {
	if len(logs) == 0 {
		return nil
	}

	entities := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		entity := log.ToEntity()
		entity["add_time"] = log.T
		entities = append(entities, entity)
	}

	return easyDB.LogCreatEntities(entities, easyDB.ITEM_LOG)
}

// ReportItemChange 统一物品变更日志上报入口
// 根据物品类型自动判断数量获取方式并上报 init/chg/fina 三段日志。
//
// 分类规则：
//   - 进主背包的物品（非装备/英雄/秘罐/饰品等特殊类型）：
//     调用 mainInventoryService.GetItemCount 获取变更前数量，计算结果值。
//   - 英雄（ITEM_TYPE_HERO）、装备（ITEM_TYPE_EQUIP）：
//     具有唯一实体，无统计意义的总量，init/fina 填 "-"，chg 填变化量。
//   - 秘罐（ITEM_TYPE_LOOP_BOX）：
//     不进主背包但有数量，通过 playerModel.LoopBoxModel 按 targetId（boxId）获取当前数量。
//   - 饰品（ITEM_TYPE_ACCESSORY）：
//     不进主背包但有 Num 计数，通过 playerModel.AccessoryModel 按 targetId 获取当前数量。
//   - VIP卡（ITEM_TYPE_VIP_CARD）、通行证（ITEM_TYPE_PASS/PASS_VIP）、
//     广告宝箱（ITEM_TYPE_AD_CHEST）等不具可统计总量的特殊系统：
//     init/fina 填 "-"，chg 填变化量。
//
// 参数说明：
//   - player: 玩家对象
//   - item: 物品配置（含 ID 和 Num）
//   - fromType: 来源类型枚举（enum.ItemChangeReason）
//   - fromValue: 来源附加值（如副本ID、活动ID等，无则传 0）
//   - extra: 额外标记值（无则传 0）
//   - isAdd: true 表示增加，false 表示减少
func ReportItemChange(
	player logicCommon.PlayerInterface,
	item *gameConfig.ItemConfig,
	fromType enum.ItemChangeReason,
	fromValue int64,
	extra int64,
	isAdd bool,
) {
	if fromType == enum.ITEM_CHANGE_REASON_PASS_CARD {
		playerModel := player.(*model.PlayerModel)
		extra = int64(playerModel.PassModel.GetReqPassId())
	}
	if item == nil {
		return
	}
	itemCfg := gameConfig.GetItemCfg(item.ID)
	if itemCfg == nil {
		return
	}

	uid := player.GetUserId()
	itemType := itemCfg.ShowGroup
	chgNum := item.Num
	if !isAdd {
		chgNum = -chgNum
	}
	chgStr := fmt.Sprintf("%d", chgNum)

	var initStr, finaStr string

	switch enum.ItemType(itemType) {
	case enum.ITEM_TYPE_EQUIP:
		// 装备：id 填装备模板 ID（itemCfg.Id）
		_ = ReportUserItemChange(
			uid,
			itemType,
			itemCfg.Id,
			int32(fromType),
			fromValue,
			extra,
			"-",
			chgStr,
			"-",
		)
		return
	case enum.ITEM_TYPE_HERO:
		// 英雄：id 填 hero 配置表 ID（itemCfg.Id）
		_ = ReportUserItemChange(
			uid,
			itemType,
			itemCfg.Id,
			int32(fromType),
			fromValue,
			extra,
			"-",
			chgStr,
			"-",
		)
		return

	case enum.ITEM_TYPE_LOOP_BOX:
		// 秘罐：通过 playerModel.LoopBoxModel 读取当前箱子数量
		if pm, ok := player.(*model.PlayerModel); ok && pm.LoopBoxModel != nil {
			boxId := itemCfg.TargetId
			boxList := pm.LoopBoxModel.LoopBoxEntity.BoxList
			idx := int(boxId) - 1
			var beforeCount int64
			if idx >= 0 && idx < len(boxList) {
				beforeCount = int64(boxList[idx])
			}
			initStr = fmt.Sprintf("%d", beforeCount)
			finaStr = fmt.Sprintf("%d", beforeCount+chgNum)
		} else {
			initStr = "-"
			finaStr = "-"
		}

	case enum.ITEM_TYPE_ACCESSORY:
		// 饰品：通过 playerModel.AccessoryModel 读取当前 Num
		if pm, ok := player.(*model.PlayerModel); ok && pm.AccessoryModel != nil {
			var beforeCount int64
			if entity := pm.AccessoryModel.Entities[itemCfg.TargetId]; entity != nil {
				beforeCount = int64(entity.Num)
			}
			initStr = fmt.Sprintf("%d", beforeCount)
			finaStr = fmt.Sprintf("%d", beforeCount+chgNum)
		} else {
			initStr = "-"
			finaStr = "-"
		}

	case enum.ITEM_TYPE_VIP_CARD, enum.ITEM_TYPE_PASS, enum.ITEM_TYPE_PASS_VIP, enum.ITEM_TYPE_AD_CHEST:
		// 特殊系统道具：不具可统计总量，只记录变化量
		initStr = "-"
		finaStr = "-"

	default:
		// 进主背包的物品：通过 mainInventoryService.GetItemCount 获取变更前数量
		beforeCount, err := mainInventoryService.GetItemCount(uid, item.ID)
		if err != nil {
			beforeCount = 0
		}
		initStr = fmt.Sprintf("%d", beforeCount)
		afterCount := beforeCount + chgNum
		if afterCount < 0 {
			afterCount = 0
		}
		finaStr = fmt.Sprintf("%d", afterCount)
	}

	_ = ReportUserItemChange(
		uid,
		itemType,
		item.ID,
		int32(fromType),
		fromValue,
		extra,
		initStr,
		chgStr,
		finaStr,
	)
}

// GetUserItemLogs 查询用户物品日志
// where 参数示例：
//
//	map[string]interface{}{
//	    "uid": userId,
//	    "it":  itemType,
//	    "st":  startTime, // 可选，开始时间（毫秒）
//	    "et":  endTime,   // 可选，结束时间（毫秒）
//	}
func GetUserItemLogs(where map[string]interface{}) ([]*UserItemLog, error) {
	rows, err := easyDB.LogGetEntitiesByWhere(where)
	if err != nil {
		return nil, err
	}

	result := make([]*UserItemLog, 0, len(rows))
	for _, row := range rows {
		log := &UserItemLog{
			Uid:  row["uid"].(int64),
			T:    row["t"].(int64),
			It:   row["it"].(int32),
			Id:   row["id"].(int32),
			Ft:   row["ft"].(int32),
			Fv:   row["fv"].(int32),
			Ext:  row["ext"].(int64),
			Init: row["init"].(string),
			Chg:  row["chg"].(string),
			Fina: row["fina"].(string),
		}
		result = append(result, log)
	}

	return result, nil
}
