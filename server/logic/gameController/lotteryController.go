package gameController

import (
	"context"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/hero"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	unlockSvc "github.com/drop/GoServer/server/logic/unlockService"
	"github.com/drop/GoServer/server/logic/vipCard"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("Lotter", &LotteryController{})
}

type LotteryController struct {
}

var _ LogicControllerInterface = (*LotteryController)(nil)

func (l *LotteryController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_REQ, &pb.GetLotteryHistoryDetailReq{}, GetLotteryHistoryDetailHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_LOTTERY_INFO_REQ, &pb.GetLotteryInfoReq{}, GetLotteryInfoHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_LOTTERY_REQ, &pb.LotteryReq{}, LotteryHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_NEW_LUCKY_REWARD_REQ, &pb.NewLuckyRewardReq{}, NewLuckyRewardHandle, enum.FUNCTION_ID_NONE)
}

func GetLotteryHistoryDetailHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("get lottery history detail req", player)

	req, ok := message.(*pb.GetLotteryHistoryDetailReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	lotteryId := req.LotteryId
	cfg := gameConfig.GetSummonPoolCfg(lotteryId)
	if cfg == nil {
		platformLogger.InfoWithUser("lottery req param error", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_LOTTERY_LOTTERY_ID_ERROR)
		return
	}
	if int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)) != 0 && !unlockService.CheckSystemUnlock(int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)), player) {
		platformLogger.InfoWithUser("lottery system not open", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}
	if player.LotteryModel == nil || player.LotteryModel.LotteryHistoryDetail == nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, &pb.GetLotteryHistoryDetailResp{
			HistoryDetails: nil,
		})
		return
	}
	res := make([]*pb.LotteryHistoryDetail, 0)
	for _, v := range player.LotteryModel.LotteryHistoryDetail[lotteryId] {
		res = append(res, &pb.LotteryHistoryDetail{
			Id:     v.Id,
			ItemId: v.ItemId,
			Count:  v.Count,
		})
	}
	resp := &pb.GetLotteryHistoryDetailResp{
		HistoryDetails: res,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, resp)
}

func GetLotteryInfoHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("get lottery info req", player)

	req, ok := message.(*pb.GetLotteryInfoReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	lotteryId := req.LotteryId
	cfg := gameConfig.GetSummonPoolCfg(lotteryId)
	if cfg == nil {
		platformLogger.InfoWithUser("lottery req param error", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_LOTTERY_LOTTERY_ID_ERROR)
		return
	}
	if int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)) != 0 && !unlockService.CheckSystemUnlock(int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)), player) {
		platformLogger.InfoWithUser("lottery system not open", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}
	if player.LotteryModel == nil || player.LotteryModel.LotteryEntities == nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_INFO_RESP, &pb.GetLotteryInfoResp{
			LotteryInfo: &pb.LotteryInfo{
				LotteryCount:         0,
				LastBasicGuaranteNum: 0,
			},
			FirstDropFree: 0,
		})
		return
	}
	lotteryEntity := player.LotteryModel.GetLotteryEntityById(req.LotteryId)
	if lotteryEntity == nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_INFO_RESP, &pb.GetLotteryInfoResp{
			LotteryInfo: &pb.LotteryInfo{
				LotteryCount:         0,
				LastBasicGuaranteNum: 0,
			},
			FirstDropFree: 0,
		})
		return
	}
	if lotteryEntity.FirstDropFree > 0 {
		if !tool.IsSameDayByMilli(lotteryEntity.LastFirstDropFreeTime, tool.UnixNowMilli()) {
			lotteryEntity.FirstDropFree--
			player.LotteryModel.UpdateFirstDropFree(lotteryId, lotteryEntity.FirstDropFree)
		}
	}
	lotteryLuckyEvent := player.LotteryModel.GetLotteryLuckyEvent()
	resp := &pb.GetLotteryInfoResp{
		LotteryInfo: &pb.LotteryInfo{
			LotteryCount:         lotteryEntity.AllCount,
			LastBasicGuaranteNum: lotteryEntity.LastBasicGuaranteeNum,
		},
		FirstDropFree: player.LotteryModel.LotteryEntities[req.LotteryId].FirstDropFree,
		LotteryLuckyEvent: &pb.LotteryLuckyEvent{
			LuckyNum:   lotteryLuckyEvent.LuckyNum,
			CreateTime: lotteryLuckyEvent.CreateTime,
		},
		//NewLuckyCount:
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_INFO_RESP, resp)
}

func LotteryHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("lottery req", player)

	req, ok := message.(*pb.LotteryReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if player.LotteryModel == nil {
		platformLogger.ErrorWithUser("lottery model nil", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_LOTTERY_LOAD_INFO_ERROR)
		return
	}

	lotteryId := req.LotteryId
	cfg := gameConfig.GetSummonPoolCfg(lotteryId)
	if cfg == nil {
		platformLogger.InfoWithUser("lottery req param error", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_LOTTERY_LOTTERY_ID_ERROR)
		return
	}
	if int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)) != 0 && !unlockService.CheckSystemUnlock(int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)), player) {
		platformLogger.InfoWithUser("lottery system not open", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}
	lotteryType := req.LotteryType
	lotteryNum := int32(0)
	lotteryDetail := player.LotteryModel.GetLotteryEntityById(lotteryId)
	if lotteryDetail == nil {
		// 初始化抽奖信息
		newEntity := &model.LotteryEntity{
			UserId:                  player.GetUserId(),
			Id:                      lotteryId,
			AllCount:                0,
			LastBasicGuaranteeNum:   0,
			LastSpecialGuaranteeNum: 0,
			SpecialGuaranteeCount:   0,
			LastOnceEffectiveNum:    0,
			OnceEffectiveCount:      0,
			LastChangeTime:          tool.UnixNowMilli(),
			FirstDropFree:           0,
		}
		if cfg.ActModID != 0 {
			ok, actVersion := player.PlayerActivityModel.CheckActivityOpen(cfg.ActModID)
			if ok {
				newEntity.ActVersion = actVersion
			} else {
				platformLogger.InfoWithUser("lottery req activity not open", player)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
				return
			}
		}
		if err := player.LotteryModel.AddLotteryEntity(newEntity); err != nil {
			platformLogger.ErrorWithUser("add lottery entity error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_LOTTERY_LOAD_INFO_ERROR)
			return
		}
		lotteryDetail = newEntity
	}

	if lotteryType == 1 {
		lotteryNum = 1
	} else if lotteryType == 2 {
		lotteryNum = 10
	} else {
		platformLogger.ErrorWithUser("lottery req param error", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_LOTTERY_TYPE_ERROR)
		return
	}

	res := make([]*pb.ItemBasicInfo, 0)

	//判断背包容量
	if lotteryId == 1 {
		if len(player.HeroDetailsModel.Entities)+int(lotteryNum) > hero.HeroBagMaxNum {
			platformLogger.InfoWithUser("lottery req bag full", player)
			messageSender.SendMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, &pb.LotteryResp{
				Result: pb.LotteryResult_LOTTERY_RESULT_HERO_BAG_FULL,
			})
			return
		}
	}

	// 判断抽奖道具是否足够
	if lotteryNum == 1 && cfg.FirstDropFree-lotteryDetail.FirstDropFree < 0 || lotteryNum == 10 {
		if flag, err := itemService.CheckItemCount(player, &gameConfig.ItemConfig{ID: gameConfig.GetSummonPoolCfg(req.LotteryId).DropToken, Num: int64(lotteryNum)}); !flag || err != nil {
			platformLogger.InfoWithUser("lottery req lack item", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_LOTTERY_ITEM_NOT_ENOUGH)
			return
		}
	}

	// todo 判断卡池时间
	if cfg.ActModID != 0 {
		ok, actVersion := player.PlayerActivityModel.CheckActivityOpen(cfg.ActModID)
		if !ok {
			platformLogger.InfoWithUser("lottery req activity not open", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
			return
		}
		if lotteryDetail.ActVersion != "" && lotteryDetail.ActVersion != actVersion {
			// todo 重置卡池数据
		}
	}

	if cfg.UnlockID != 0 && !unlockService.CheckUnlock(cfg.UnlockID, player) {
		platformLogger.InfoWithUser("lottery req unlock not open", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}
	vipCards, err := vipCard.Service.GetAllFunctionValues(player)
	if err != nil {
		platformLogger.ErrorWithUser("GetVipCardInfoList failed", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	var lossValue int64 = 0
	if _, ok := vipCards[enum.VIP_PRIVILEGE_LOTTERY_GUAR]; ok {
		lossValue = vipCards[enum.VIP_PRIVILEGE_LOTTERY_GUAR]
	}

	for i := int32(0); i < lotteryNum; i++ {
		// 抽奖逻辑
		player.LotteryModel.UpdateAllCount(req.LotteryId, lotteryDetail.AllCount+1)
		// 单次保底全部触发 或 未配置
		if lotteryDetail.OnceEffectiveCount >= int32(len(cfg.Guarantees1)) {
			//循环保底
			if len(cfg.Guarantees2) > 0 {
				if lotteryDetail.AllCount-lotteryDetail.LastSpecialGuaranteeNum >= cfg.Guarantees2[lotteryDetail.SpecialGuaranteeCount].Num {

					dropGroupId := gameConfig.WeightedRandomChoice(cfg.Guarantees2[lotteryDetail.SpecialGuaranteeCount].DropGroupIdList, cfg.Guarantees2Weight[lotteryDetail.SpecialGuaranteeCount])
					itemIdList := gameConfig.DropGroupItems(dropGroupId, nil)

					player.LotteryModel.UpdateLastBasicGuaranteeNum(req.LotteryId, lotteryDetail.AllCount)
					for _, v := range itemIdList {
						itemCfg := gameConfig.GetItemCfg(v.ID)
						if itemCfg != nil {
							if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_HERO) {
								player.LotteryModel.AddLotteryLog(&model.LotteryLog{UserId: lotteryDetail.UserId, Id: req.LotteryId, ItemId: v.ID, Count: lotteryDetail.AllCount, AddTime: tool.UnixNowMilli()})
							}
						}
					}
					player.LotteryModel.UpdateLastSpecialGuaranteeNum(req.LotteryId, lotteryDetail.AllCount)
					player.LotteryModel.CheckAndUpSpecialGuaranteeNum(cfg, lotteryId)
					player.LotteryModel.UpdateLastChengeTime(req.LotteryId, tool.UnixNowMilli())

					for _, itemInfo := range itemIdList {
						res = append(res, &pb.ItemBasicInfo{
							ItemId: itemInfo.ID,
							Count:  itemInfo.Num,
						})
					}

				} else {
					// 普通保底
					if cfg.Guarantees != nil && lotteryDetail.AllCount-lotteryDetail.LastBasicGuaranteeNum >= cfg.Guarantees.Num-int32(lossValue) {
						player.LotteryModel.BasicLotteryGuarantee(cfg, &res, false, true, lotteryId)
					} else {
						// 普通抽取，抽取次数决定卡池
						player.LotteryModel.BasicLottery(cfg, &res, false, true, lotteryId)
					}
				}
			} else {
				if cfg.Guarantees != nil && lotteryDetail.AllCount-lotteryDetail.LastBasicGuaranteeNum >= cfg.Guarantees.Num-int32(lossValue) {
					player.LotteryModel.BasicLotteryGuarantee(cfg, &res, false, false, lotteryId)
				} else {
					// 普通抽取，抽取次数决定卡池
					player.LotteryModel.BasicLottery(cfg, &res, false, false, lotteryId)
				}
			}
		} else {
			// 单次保底未触发
			if lotteryDetail.AllCount-lotteryDetail.LastOnceEffectiveNum >= cfg.Guarantees1[lotteryDetail.OnceEffectiveCount].Num {
				// 触发单次保底
				dropGroupId := gameConfig.WeightedRandomChoice(cfg.Guarantees1[lotteryDetail.OnceEffectiveCount].DropGroupIdList, cfg.Guarantees1Weight[lotteryDetail.OnceEffectiveCount])
				itemIdList := gameConfig.DropGroupItems(dropGroupId, nil)
				player.LotteryModel.UpdateLastBasicGuaranteeNum(req.LotteryId, lotteryDetail.AllCount)
				for _, v := range itemIdList {
					itemCfg := gameConfig.GetItemCfg(v.ID)
					if itemCfg != nil {
						if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_HERO) {
							player.LotteryModel.AddLotteryLog(&model.LotteryLog{UserId: lotteryDetail.UserId, Id: req.LotteryId, ItemId: v.ID, Count: lotteryDetail.AllCount, AddTime: tool.UnixNowMilli()})
						}
					}
				}
				player.LotteryModel.UpdateLastSpecialGuaranteeNum(req.LotteryId, lotteryDetail.AllCount)
				player.LotteryModel.UpdateLastOnceEffectiveNum(req.LotteryId, lotteryDetail.AllCount)
				player.LotteryModel.UpdateOnceEffectiveCount(req.LotteryId, lotteryDetail.OnceEffectiveCount+1)
				player.LotteryModel.UpdateLastChengeTime(req.LotteryId, tool.UnixNowMilli())
				for _, itemInfo := range itemIdList {
					res = append(res, &pb.ItemBasicInfo{
						ItemId: itemInfo.ID,
						Count:  itemInfo.Num,
					})
				}
			} else {
				// 普通保底
				if cfg.Guarantees != nil && lotteryDetail.AllCount-lotteryDetail.LastBasicGuaranteeNum >= cfg.Guarantees.Num-int32(lossValue) {
					player.LotteryModel.BasicLotteryGuarantee(cfg, &res, true, false, lotteryId)
				} else {
					player.LotteryModel.BasicLottery(cfg, &res, true, false, lotteryId)
				}
			}
		}
	}
	if lotteryNum == 1 && lotteryDetail.FirstDropFree < cfg.FirstDropFree {
		player.LotteryModel.UpdateFirstDropFree(lotteryId, lotteryDetail.FirstDropFree+1)
		player.LotteryModel.UpdateLastFirstDropFreeTime(lotteryId, tool.UnixNowMilli())
	} else {
		removeItems := &gameConfig.ItemConfig{ID: gameConfig.GetSummonPoolCfg(req.LotteryId).DropToken, Num: int64(lotteryNum)}
		lotteryLuckyEvent := player.LotteryModel.GetLotteryLuckyEvent()
		// 抽奖类型：英雄卡池          数量：十抽          卡池类型：限定卡池
		if cfg.Gashatype == 1 && lotteryNum == 10 && cfg.ActModID != 0 {
			luckyEventCfgTime := gameConfig.GetConstantCfg(gameConfig.CONSTANT_limitedGachaLuckyEventTime)
			// 事件存在，折数存在，且未过期
			if lotteryLuckyEvent != nil && lotteryLuckyEvent.LuckyNum > 0 && (tool.UnixNowMilli()-lotteryLuckyEvent.CreateTime) < int64(luckyEventCfgTime.Value[0]*1000) {
				removeItems.Num = removeItems.Num * int64(lotteryLuckyEvent.LuckyNum) / 10
			}
			err := player.LotteryModel.CreateLotteryLuckyEvent()
			if err != nil {
				platformLogger.ErrorWithUser("create lottery lucky event error", player, err)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_CREATE_LOTTERY_LUCKY_EVENT_ERROR)
				return
			}
		}
		if cfg.Gashatype == 1 && !lotteryLuckyEvent.IsNewLuckyReward {
			player.LotteryModel.UpdateNewLuckyCount(lotteryLuckyEvent.NewLuckyCount + lotteryNum)
		}
		err := itemService.RemoveItem(player, removeItems, enum.ITEM_CHANGE_REASON_DRAW_LOTTERY)
		if err != nil {
			platformLogger.InfoWithUser("扣除材料失败", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
			return
		}
	}
	items := make([]*gameConfig.ItemConfig, 0)
	for _, v := range res {
		items = append(items, &gameConfig.ItemConfig{
			ID:  v.ItemId,
			Num: v.Count,
		})
	}
	err = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_LOTTERY)
	if err != nil {
		platformLogger.ErrorWithUser("add item error", player, err)
		return
	}
	lotteryLuckyEvent := player.LotteryModel.GetLotteryLuckyEvent()
	resp := &pb.LotteryResp{
		Result:   pb.LotteryResult_LOTTERY_RESULT_SUCCESS,
		InfoList: res,
		LotteryInfo: &pb.LotteryInfo{
			LotteryCount:         lotteryDetail.AllCount,
			LastBasicGuaranteNum: lotteryDetail.LastBasicGuaranteeNum,
		},
		FirstDropFree: player.LotteryModel.LotteryEntities[req.LotteryId].FirstDropFree,
		LotteryLuckyEvent: &pb.LotteryLuckyEvent{
			CreateTime: lotteryLuckyEvent.CreateTime,
			LuckyNum:   lotteryLuckyEvent.LuckyNum,
		},
		NewLuckyCount: lotteryLuckyEvent.NewLuckyCount,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, resp)

	eventBusService.SubmitLuckyLotteryEvent(player.GetUserId(), enum.AllLotteryType[cfg.Gashatype], lotteryNum, items)

	// 上报抽奖日志
	operationLogService.OnUserHeroLottery(player.GetUserId(), lotteryNum)

	// 记录今日抽卡数据到 Redis
	ctx := context.Background()
	if cfg.Gashatype == 2 {
		player.StaticData.UpdateCollectionLotteryDrawCount(player.StaticData.GetCollectionLotteryDrawCount() + lotteryNum)
		err := unlockSvc.DailyCache.RecordCollectionLottery(ctx, player.GetUserId(), lotteryNum)
		if err != nil {
			platformLogger.ErrorWithUser("record collection lottery to redis error", player, err)
		}
	}
	for _, item := range items {
		err := unlockSvc.DailyCache.RecordLottery(ctx, player.GetUserId(), lotteryId, item.ID, int32(item.Num))
		if err != nil {
			platformLogger.ErrorWithUser("record lottery to redis error", player, err)
		}
	}
}

func NewLuckyRewardHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.NewLuckyRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_NEW_LUCKY_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	newLuckyNumCfg := gameConfig.GetConstantCfg(gameConfig.CONSTANT_beginnerBenefitsNumbers)
	if newLuckyNumCfg == nil || len(newLuckyNumCfg.Value) < 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_NEW_LUCKY_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	flag := player.LotteryModel.RewardNewLucky(req.ItemId)
	if !flag {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_NEW_LUCKY_REWARD_RESP, pb.ERROR_CODE_NEW_LUCKY_REWARD_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_NEW_LUCKY_REWARD_RESP, &pb.NewLuckyRewardResp{})
}
