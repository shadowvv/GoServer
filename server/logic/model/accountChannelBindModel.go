package model

import (
	"errors"

	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"gorm.io/gorm"
)

// AccountChannelBindEntity 账号渠道绑定记录（玩家维度）
// primary key: user_id + channel
// claim_status: 0 未领取，1 已绑定，2 已领取；必须按 0→1→2 流转
type AccountChannelBindEntity struct {
	UserId      int64  `gorm:"column:user_id;primaryKey"`
	Channel     string `gorm:"column:channel;primaryKey;size:64"`
	ClaimStatus int32  `gorm:"column:claim_status"`
}

func (AccountChannelBindEntity) TableName() string {
	return "account_channel_bind"
}

func GetAllChannelBind(userId int64) []string {
	entities, err := easyDB.GetPlayerEntitiesByWhere[AccountChannelBindEntity](map[string]interface{}{
		"user_id": userId,
	})
	if err != nil {
		return []string{}
	}
	res := make([]string, 0, len(entities))
	for _, e := range entities {
		if e.Channel != "" && e.ClaimStatus == 2 {
			res = append(res, e.Channel)
		}
	}
	return res
}

// GetChannelBind 获取玩家某个渠道的绑定记录
func GetChannelBind(userId int64, channel string) (*AccountChannelBindEntity, error) {
	if channel == "" {
		return nil, nil
	}
	entity, err := easyDB.GetPlayerEntityByWhere[AccountChannelBindEntity](map[string]interface{}{
		"user_id": userId,
		"channel": channel,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return entity, nil
}

// BindChannelIfNotExists 记录绑定（未绑定则插入，已绑定则忽略）
func BindChannelIfNotExists(userId int64, channel string) error {
	if channel == "" {
		return nil
	}
	entity, err := GetChannelBind(userId, channel)
	if err != nil || entity != nil {
		return err
	}
	return easyDB.CreatePlayerEntity(&AccountChannelBindEntity{
		UserId:      userId,
		Channel:     channel,
		ClaimStatus: 0,
	})
}

// TryMarkAsBound 0→1：将“未领取”标记为“已绑定”，仅当当前为 0 时执行
// 返回值：true 表示本次成功执行 0→1
func TryMarkAsBound(userId int64, channel string) (ok bool, err error) {
	entity, err := GetChannelBind(userId, channel)
	if err != nil || entity == nil {
		return false, err
	}
	if entity.ClaimStatus != 0 {
		return false, nil
	}
	entity.ClaimStatus = 1
	easyDB.UpdatePlayerEntity(entity, map[string]interface{}{"claim_status": int32(1)}, userId)
	return true, nil
}

// TryMarkChannelClaimed 1→2：将“已绑定”标记为“已领取”，仅当当前为 1 时执行
// 返回值：claimed == true 表示本次成功执行 1→2，可发奖
func TryMarkChannelClaimed(userId int64, channel string) (claimed bool, err error) {
	entity, err := GetChannelBind(userId, channel)
	if err != nil || entity == nil {
		return false, err
	}
	if entity.ClaimStatus != 1 {
		return false, nil
	}
	entity.ClaimStatus = 2
	easyDB.UpdatePlayerEntity(entity, map[string]interface{}{"claim_status": int32(2)}, userId)
	return true, nil
}
