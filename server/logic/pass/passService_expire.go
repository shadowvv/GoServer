// File: passService_expire.go
// Description: 通行证过期未领取奖励发邮件
// Author: 木村凉太
// Create Time: 2026.02

package pass

import (
	"errors"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/mail"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"go.uber.org/zap"
)

const (
	passExpireMailTitle      = "通行证奖励补发"
	passExpireMailContent    = "您的通行证已结束，以下未领取的奖励已通过邮件发放，请查收。"
	passExpireMailExpireDays = 7
)

// ComputeUnclaimedRewardsAndMarkSent 计算未领取奖励、标记为已发并落库，返回邮件附件列表（多选奖励取首个选项）
func (p *PassService) ComputeUnclaimedRewardsAndMarkSent(passModel *model.PassModel, passId int32) ([]*logicCommon.MailAttachmentItem, error) {
	if passModel == nil {
		return nil, errors.New("pass model is nil")
	}
	basePassCfg := gameConfig.GetBasePassCfg(passId)
	if basePassCfg == nil {
		return nil, errors.New("pass config not found")
	}
	progress, _ := passModel.GetOrCreateProgress(passId)
	currentLevel := progress.Level
	allRewardCfgs := gameConfig.GetAllPassRewardCfgByPassId(passId)
	if len(allRewardCfgs) == 0 {
		return nil, nil
	}

	var items []*logicCommon.MailAttachmentItem
	for _, rewardCfg := range allRewardCfgs {
		if rewardCfg.Level > currentLevel {
			continue
		}
		for rewardLevel := int32(0); rewardLevel <= 2; rewardLevel++ {
			if passModel.HasReceivedReward(passId, rewardCfg.Level, rewardLevel) {
				continue
			}
			if rewardLevel > 0 && !passModel.HasVipLevel(passId, rewardLevel) {
				continue
			}

			var dropId int32
			switch rewardLevel {
			case 0:
				dropId = rewardCfg.DropId1
			case 1:
				if len(rewardCfg.DropId2) == 0 {
					continue
				}
				dropId = rewardCfg.DropId2[0]
			case 2:
				if len(rewardCfg.DropId3) == 0 {
					continue
				}
				dropId = rewardCfg.DropId3[0]
			default:
				continue
			}
			if dropId <= 0 {
				continue
			}

			dropCfg := gameConfig.GetDropCfg(dropId)
			if dropCfg == nil {
				continue
			}
			dropItemCount := gameConfig.GetDropItemCount(dropId)
			var chosenItem *gameConfig.ItemConfig
			if dropItemCount == 1 {
				drops := gameConfig.Drop(dropId)
				if len(drops) > 0 && drops[0] != nil {
					chosenItem = drops[0]
				}
			} else if dropItemCount > 1 {
				existing := passModel.GetDropChoice(passId, rewardCfg.Level, rewardLevel, dropId)
				if existing != nil {
					chosenItem = &gameConfig.ItemConfig{ID: existing.ChosenItemId, Num: 1}
				} else {
					if len(dropCfg.FixedItem) > 0 && dropCfg.FixedItem[0] != nil {
						chosenItem = dropCfg.FixedItem[0]
					} else if len(dropCfg.Groups) > 0 && len(dropCfg.Groups[0].Items) > 0 && dropCfg.Groups[0].Items[0] != nil {
						chosenItem = dropCfg.Groups[0].Items[0]
					}
					if chosenItem != nil {
						passModel.SetDropChoice(passId, rewardCfg.Level, rewardLevel, dropId, chosenItem.ID)
					}
				}
			}
			if chosenItem == nil || chosenItem.ID <= 0 {
				continue
			}

			passModel.AddRewardRecord(passId, rewardCfg.Level, rewardLevel)
			items = append(items, &logicCommon.MailAttachmentItem{
				Type: mail.AttachmentItemTypeItem,
				ID:   chosenItem.ID,
				Num:  int32(chosenItem.Num),
			})
		}
	}
	if len(items) > 0 {
		passModel.SaveModelToDB()
	}
	return items, nil
}

// ProcessExpiredPassMailsForPlayer 玩家登录时检测：该玩家所在服已结束的通行证活动中，若有未领取奖励则补发邮件
// TODO:后期优化
func (p *PassService) ProcessExpiredPassMailsForPlayer(player logicCommon.PlayerInterface) {
	if p.MailService == nil || player == nil {
		return
	}
	userId := player.GetUserId()
	serverId := player.GetUserServerId()
	now := tool.UnixNowMilli()
	allOpen, err := easyDB.GetServerAllEntities[model.ServerOpenActivityEntity]()
	if err != nil {
		logger.ErrorWithZapFields("[Pass] get server open activity failed", zap.Error(err))
		return
	}

	passModel := p.getOrLoadPassModel(player)
	if passModel == nil {
		return
	}

	for _, open := range allOpen {
		if open == nil || open.OpenServerId != serverId || open.EndTime >= now {
			continue
		}
		var passIds []int32
		for passId, cfg := range gameConfig.GetAllBasePassCfg() {
			if cfg != nil && cfg.ActId == open.ActivityId {
				passIds = append(passIds, passId)
			}
		}
		for _, passId := range passIds {
			if _, ok := passModel.ProgressMap[passId]; !ok {
				continue
			}
			items, err := p.ComputeUnclaimedRewardsAndMarkSent(passModel, passId)
			if err != nil || len(items) == 0 {
				continue
			}
			expireAt := tool.UnixNow() + int64(passExpireMailExpireDays)*24*3600
			mailObj := &logicCommon.Mail{
				UserID:       userId,
				MailType:     mail.MailTypeOfficial,
				Title:        passExpireMailTitle,
				Content:      passExpireMailContent,
				SenderID:     0,
				SenderName:   "系统",
				Items:        items,
				ExpireTime:   expireAt,
				SendTime:     tool.UnixNow(),
				IsConvenient: true,
			}
			_, err = p.MailService.SendMailToUserId(userId, mailObj)
			if err != nil {
				logger.ErrorWithZapFields("[Pass] send expire mail failed", zap.Int64("user_id", userId), zap.Int32("pass_id", passId), zap.Error(err))
			}
		}
	}
}
