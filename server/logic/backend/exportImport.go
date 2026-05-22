package backend

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlayerExportPayload 结构化导出的玩家全部数据
type PlayerExportPayload struct {
	SourceUserId int64                        `json:"source_user_id"`
	User         *model.UserEntity            `json:"user"`
	Heroes       []*model.HeroDetailsEntity   `json:"heroes"`
	Formations   []*model.HeroFormationEntity `json:"formations"`
	Equipments   []*model.EquipmentEntity     `json:"equipments"`
	Pets         []*model.PetEntity           `json:"pets"`
	Accessories  []*model.AccessoryEntity     `json:"accessories"`
	// 以下子模型只有 user_id 需要替换，使用通用 JSON 方式处理
	SubModels map[string]json.RawMessage `json:"sub_models"`
}

// subModelTableList 只需替换 user_id 的子模型表名列表（必须与 Entity.TableName() 一致）
var subModelTableList = []string{
	"player_static_data",
	"player_story_trigger",
	"task",
	"bounty",
	"player_expedition_data",
	"player_expedition_battlefield_data",
	"player_expedition_slot_data",
	"lottery",
	"loop_box",
	"player_pass_progress",
	"player_pass_vip",
	"player_pass_reward",
	"player_pass_drop_choice",
	"pass_card_task",
	"player_vip_card",
	"player_privilege_reward",
	"player_shop_item_data",
	"player_pass_data",
	"player_Instance_data",
	"player_activity_data",
	"player_sign_data",
	"idle",
	"player_adventure",
	"player_adventure_settle_type",
	"player_adventure_entry",
	"player_arena_data",
	"architecture",
	"stone",
	"lumber",
	"city_furniture",
	"trial",
	"collection",
	"collection_entry",
	"appearance",
	"city_age",
	"player_function_data",
	"player_ad_chest",
	"player_ad_chest_daily",
	"player_token_shop",
	"album_reward_score",
	"hero_album",
	"pet_affinity",
	"pet_affinity_book",
	"pet_recruit",
	"accessory_lucky",
	"task_active_reward",
	"player_inventory",
}

// ExportPlayerStructured 从数据库加载指定玩家的所有数据，序列化为 JSON
func ExportPlayerStructured(userId int64) (string, error) {
	db := easyDB.GetPlayerDB()

	// 1. 查询 UserEntity
	userEntity, err := easyDB.GetPlayerEntityByWhere[model.UserEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return "", fmt.Errorf("query user error: %w", err)
	}

	payload := &PlayerExportPayload{
		SourceUserId: userId,
		User:         userEntity,
		SubModels:    make(map[string]json.RawMessage),
	}

	// 2. 查询核心模型（需要特殊 ID 映射的）
	payload.Heroes, _ = easyDB.GetPlayerEntitiesByWhere[model.HeroDetailsEntity](map[string]interface{}{"user_id": userId})
	payload.Formations, _ = easyDB.GetPlayerEntitiesByWhere[model.HeroFormationEntity](map[string]interface{}{"user_id": userId})
	payload.Equipments, _ = easyDB.GetPlayerEntitiesByWhere[model.EquipmentEntity](map[string]interface{}{"user_id": userId})
	payload.Pets, _ = easyDB.GetPlayerEntitiesByWhere[model.PetEntity](map[string]interface{}{"user_id": userId})
	payload.Accessories, _ = easyDB.GetPlayerEntitiesByWhere[model.AccessoryEntity](map[string]interface{}{"user_id": userId})

	// 3. 查询其他子模型（只需替换 user_id）
	for _, tableName := range subModelTableList {
		var rows []map[string]interface{}
		if err := db.Table(tableName).Where("user_id = ?", userId).Find(&rows).Error; err != nil {
			continue // 表不存在或无数据，跳过
		}
		if len(rows) > 0 {
			data, _ := json.Marshal(rows)
			payload.SubModels[tableName] = data
		}
	}

	// 4. 序列化整个 payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload error: %w", err)
	}
	return string(jsonData), nil
}

// ImportPlayerStructured 解析结构化 JSON，执行 ID 重映射，导入到目标玩家
func ImportPlayerStructured(jsonStr string, targetAccount string, targetServerId int32, createTime int64) (newUserId int64, oldUserId int64, err error) {
	// 1. 反序列化
	var payload PlayerExportPayload
	if err = json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		return 0, 0, fmt.Errorf("unmarshal payload error: %w", err)
	}
	oldUserId = payload.SourceUserId

	// 2. 创建 ID 重映射器
	remapper := NewIdRemapper()
	newUserId = remapper.NewUserId

	// 3. 验证目标玩家存在
	targetUser, err := easyDB.GetPlayerEntityByWhere[model.UserEntity](map[string]interface{}{
		"account":   targetAccount,
		"server_id": targetServerId,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("target player not found (account=%s, server_id=%d): %w", targetAccount, targetServerId, err)
	}
	oldUserId = targetUser.UserId

	// 4. 对核心模型执行 ID 重映射
	// 4.1 Heroes
	for _, h := range payload.Heroes {
		h.UserID = newUserId
		h.HeroOwnID = remapper.RemapHeroOwnId(h.HeroOwnID)
		if len(h.EquipmentId) > 0 {
			h.EquipmentId = tool.JSONInt64Slice(remapper.RemapEquipOwnIdList([]int64(h.EquipmentId)))
		}
	}

	// 4.2 Formations
	for _, f := range payload.Formations {
		f.UserID = newUserId
		if len(f.HeroOwnIDList) > 0 {
			f.HeroOwnIDList = tool.JSONInt64Slice(remapper.RemapHeroOwnIdList([]int64(f.HeroOwnIDList)))
		}
	}

	// 4.3 Equipments
	for _, e := range payload.Equipments {
		e.UserID = newUserId
		e.EquipmentOwnID = remapper.RemapEquipOwnId(e.EquipmentOwnID)
		e.HeroOwnID = remapper.RemapHeroOwnId(e.HeroOwnID)
	}

	// 4.4 Pets
	for _, p := range payload.Pets {
		p.UserID = newUserId
		p.PetOwnID = remapper.RemapPetOwnId(p.PetOwnID)
		p.HeroOwnId = remapper.RemapHeroOwnId(p.HeroOwnId)
	}

	// 4.5 Accessories
	for _, a := range payload.Accessories {
		a.UserId = newUserId
		a.HeroOwnId = remapper.RemapHeroOwnId(a.HeroOwnId)
	}

	// 5. 在事务中执行数据库操作
	db := easyDB.GetPlayerDB()
	txErr := db.Transaction(func(tx *gorm.DB) error {
		// Step 1: UPDATE UserEntity，修改 user_id 为新值
		updateFields := map[string]interface{}{
			"user_id": newUserId,
		}
		// 同时覆盖导出数据中的属性字段
		if payload.User != nil {
			updateFields["nickname"] = payload.User.Nickname
			updateFields["head_id"] = payload.User.HeadId
			updateFields["head_frame_id"] = payload.User.HeadFrameId
			updateFields["title_id"] = payload.User.TitleId
			updateFields["level"] = payload.User.Level
			updateFields["vip"] = payload.User.Vip
			updateFields["charge_count"] = payload.User.ChargeCount
			updateFields["last_charge_time"] = payload.User.LastChargeTime
		}
		// 如果传入了创号时间，使用传入的值；否则保留原数据
		if createTime > 0 {
			updateFields["register_time"] = createTime
		}
		if err := tx.Model(&model.UserEntity{}).
			Where("account = ? AND server_id = ?", targetAccount, targetServerId).
			Updates(updateFields).Error; err != nil {
			return fmt.Errorf("update user entity error: %w", err)
		}

		// Step 2: INSERT 核心模型（批量插入，每批100条）
		const batchSize = 100
		if len(payload.Heroes) > 0 {
			if err := tx.CreateInBatches(payload.Heroes, batchSize).Error; err != nil {
				return fmt.Errorf("batch insert heroes error: %w", err)
			}
		}
		if len(payload.Formations) > 0 {
			if err := tx.CreateInBatches(payload.Formations, batchSize).Error; err != nil {
				return fmt.Errorf("batch insert formations error: %w", err)
			}
		}
		if len(payload.Equipments) > 0 {
			if err := tx.CreateInBatches(payload.Equipments, batchSize).Error; err != nil {
				return fmt.Errorf("batch insert equipments error: %w", err)
			}
		}
		if len(payload.Pets) > 0 {
			if err := tx.CreateInBatches(payload.Pets, batchSize).Error; err != nil {
				return fmt.Errorf("batch insert pets error: %w", err)
			}
		}
		if len(payload.Accessories) > 0 {
			if err := tx.CreateInBatches(payload.Accessories, batchSize).Error; err != nil {
				return fmt.Errorf("batch insert accessories error: %w", err)
			}
		}

		// Step 3: INSERT 其他子模型（批量插入，替换 user_id，对 UUID 主键生成新值）
		for tableName, rawData := range payload.SubModels {
			var rows []map[string]interface{}
			if err := json.Unmarshal(rawData, &rows); err != nil {
				continue
			}
			for _, row := range rows {
				// 删除所有大小写变体的 user_id，避免 "Column specified twice" 错误
				for k := range row {
					if strings.EqualFold(k, "user_id") {
						delete(row, k)
					}
				}
				row["user_id"] = newUserId
				// 对 unique_id 主键字段生成新 UUID，避免与源玩家数据冲突
				for k := range row {
					if strings.EqualFold(k, "unique_id") {
						delete(row, k)
						row["unique_id"] = uuid.New().String()
						break
					}
				}
				// 对子模型中引用的英雄/装备/宠物 OwnID 做重映射
				remapSubModelOwnIds(tableName, row, remapper)
			}
			if len(rows) > 0 {
				if err := tx.Table(tableName).CreateInBatches(rows, batchSize).Error; err != nil {
					return fmt.Errorf("batch insert %s error: %w", tableName, err)
				}
			}
		}

		return nil
	})

	if txErr != nil {
		return 0, oldUserId, txErr
	}
	return newUserId, oldUserId, nil
}

// remapSubModelOwnIds 对通用 SubModels 中引用的英雄/装备/宠物 OwnID 字段做重映射
// 由于这些子模型走 map[string]interface{} 通道，需要按表名硬编码字段处理
// 目前涉及：lumber.hero_own_ids（JSON int64 数组，派驻英雄 OwnID 列表）
func remapSubModelOwnIds(tableName string, row map[string]interface{}, remapper *IdRemapper) {
	switch tableName {
	case "lumber":
		for k, v := range row {
			if strings.EqualFold(k, "hero_own_ids") {
				row[k] = remapJSONInt64Slice(v, remapper.RemapHeroOwnIdList)
			}
		}
	}
}

// remapJSONInt64Slice 解析存储为 JSON 的 int64 数组并执行重映射后重新序列化
// 兼容 JSON 反序列化后可能出现的 string、[]byte、[]interface{} 三种形态
func remapJSONInt64Slice(v interface{}, mapper func([]int64) []int64) interface{} {
	var ids []int64
	switch val := v.(type) {
	case nil:
		return v
	case string:
		if val == "" || val == "[]" {
			return v
		}
		if err := json.Unmarshal([]byte(val), &ids); err != nil {
			return v
		}
	case []byte:
		if len(val) == 0 || string(val) == "[]" {
			return v
		}
		if err := json.Unmarshal(val, &ids); err != nil {
			return v
		}
	case []interface{}:
		ids = make([]int64, 0, len(val))
		for _, x := range val {
			switch n := x.(type) {
			case float64:
				ids = append(ids, int64(n))
			case int64:
				ids = append(ids, n)
			case json.Number:
				if i, err := n.Int64(); err == nil {
					ids = append(ids, i)
				}
			}
		}
	default:
		return v
	}
	newIds := mapper(ids)
	b, err := json.Marshal(newIds)
	if err != nil {
		return v
	}
	return string(b)
}
