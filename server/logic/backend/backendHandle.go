package backend

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/drop/GoServer/server/logic/inventory"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/mail"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/httpPlatform"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/webProto"

	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/tool"
)

func isPermiss(permiss []int32, id int32) bool {
	for _, v := range permiss {
		if v == id {
			return true
		}
	}
	return false
}

var backendMailIdGenerator *tool.IdGenerator
var backendUserIdGenerator *tool.IdGenerator
var backendHeroIdGenerator *tool.IdGenerator
var backendEquipIdGenerator *tool.IdGenerator
var backendPetIdGenerator *tool.IdGenerator

func InitBackend() {
	backendMailIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_GM_HTTP))
	backendUserIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_USER))
	backendHeroIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_HERO))
	backendEquipIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_EQUIPMENT))
	backendPetIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_ITEM))
}

func convertPbMailItems(items []*mail.MailAttachmentItem) []*mail.MailAttachmentItem {
	if len(items) == 0 {
		return []*mail.MailAttachmentItem{}
	}
	out := make([]*mail.MailAttachmentItem, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		out = append(out, &mail.MailAttachmentItem{
			Type: it.Type,
			ID:   it.ID,
			Num:  it.Num,
		})
	}
	return out
}

func GmUser(req *GmUserReq, resp *GmResp) {
	respData := resp.Data.([]*GmUserData)
	data := easyDB.GmGetEntitiesByWhere(enum.SystemUser, map[string]interface{}{})
	if data == nil {
		resp.Code = 1
		resp.Msg = "user not found"
		return
	}
	for _, value := range data {
		if value["Account"] != nil && value["Permiss"] != nil {
			user := &GmUserData{
				User:    value["Account"].(string),
				Permiss: gameConfig.ParseIntArray(value["Permiss"].(string)),
			}
			respData = append(respData, user)
		}
	}

	resp.Data = respData
}

func EditGmUser(req *GmEditGmUserReq, resp *GmResp) {
	var err error
	switch req.Add {
	case 0:
		err = easyDB.GmUpdateEntityByWhere(enum.SystemUser, map[string]interface{}{"Permiss": tool.Int32SliceTostring(req.Permiss, "|")}, map[string]interface{}{"Account": req.Uid})
	case 1:
		signStr := md5.Sum([]byte(req.Pwd))
		tMd5Pwd := fmt.Sprintf("%x", signStr)
		err = easyDB.GmCreatEntity(enum.SystemUser, map[string]interface{}{"Account": req.Uid, "PassWord": tMd5Pwd, "Permiss": tool.Int32SliceTostring(req.Permiss, "|")})
	case 2:
		signStr := md5.Sum([]byte(req.Pwd))
		tMd5Pwd := fmt.Sprintf("%x", signStr)
		err = easyDB.GmUpdateEntityByWhere(enum.SystemUser, map[string]interface{}{"PassWord": tMd5Pwd}, map[string]interface{}{"Account": req.Uid})
	case 3:
		err = easyDB.GmDeteleEntityByWhere(enum.SystemUser, map[string]interface{}{"Account": req.Uid})
	}
	if err != nil {
		resp.Code = 1
		resp.Msg = err.Error()
	}
}

func GmUserInfo(req *GmUserInfoReq, resp *GmResp) {
	respData := resp.Data.([]*GmUserInfoData)
	var row1 []*model.UserEntity
	var err error
	if req.Type == 0 {
		row1, err = easyDB.GetPlayerEntitiesByWhere[model.UserEntity](map[string]interface{}{"account": req.Account})
		if err != nil {
			resp.Code = 1
			resp.Msg = err.Error()
			return
		}
	} else if req.Type == 1 {
		userId, err := strconv.ParseInt(req.Account, 10, 64)
		if err != nil {
			resp.Code = 5
			resp.Msg = err.Error()
			return
		}
		row1, err = easyDB.GetPlayerEntitiesByWhere[model.UserEntity](map[string]interface{}{"user_id": userId})
		if err != nil {
			resp.Code = 1
			resp.Msg = err.Error()
			return
		}
	} else if req.Type == 2 {
		row1, err = easyDB.GetPlayerEntitiesByWhere[model.UserEntity](map[string]interface{}{"nickname": req.Account})
		if err != nil {
			resp.Code = 1
			resp.Msg = err.Error()
			return
		}
	}

	for _, v := range row1 {
		row2, err := easyDB.GetPlayerEntitiesByWhere[model.PlayerInstanceEntity](map[string]interface{}{"user_id": v.UserId})
		if err != nil {
			resp.Code = 2
			resp.Msg = err.Error()
			return
		}
		userInfo := &GmUserInfoData{
			UserId:              v.UserId,
			Account:             v.Account,
			NickName:            v.Nickname,
			RegistrationTime:    v.RegisterTime,
			RegistrationChannel: v.ChannelId,
			ServerId:            v.ServerId,
			RechargeNum:         v.ChargeCount,
			LastLoginTime:       v.LastLoginTime,
			LastOfflineTime:     v.LastOfflineTime,
		}
		for _, value := range row2 {
			if value.InstanceId == int32(enum.MAIN_INSTANCE_ID) {
				userInfo.MainLevel = value.MaxStageId
			}
			if value.InstanceId == int32(enum.FIVE_VS_FIVE_TOWER_INSTANCE_ID) {
				userInfo.FiveVsFLevel = value.MaxStageId
			}
		}
		row3, err := easyDB.GetPlayerEntityByWhere[model.HeroFormationEntity](map[string]interface{}{"user_id": v.UserId, "formation_type": pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN, "is_active": true})
		if err != nil {
			resp.Code = 3
			resp.Msg = err.Error()
			return
		}
		for _, v := range row3.HeroOwnIDList {
			row4, err := easyDB.GetPlayerEntityByWhere[model.HeroDetailsEntity](map[string]interface{}{"hero_own_id": v})
			if err != nil {
				resp.Code = 4
				resp.Msg = err.Error()
				return
			}
			userInfo.Power += row4.Power
		}
		row4, err := easyDB.GetServerEntityByWhere[model.BanListEntity](map[string]interface{}{"account": v.Account})
		if err != nil {
			if err.Error() != "record not found" {
				resp.Code = 5
				resp.Msg = err.Error()
				return
			}
		}
		if row4 != nil {
			userInfo.BanStatus = 0
			if row4.StartTime < tool.UnixNowMilli() && row4.EndTime > tool.UnixNowMilli() {
				userInfo.BanStatus = 1
			}
			userInfo.BanReason = row4.Reason
		}
		// 查询禁言状态
		muteRecord, err := easyDB.GetServerEntityByWhere[model.UserBanRecordEntity](map[string]interface{}{"account": v.Account})
		if err != nil {
			if err.Error() != "record not found" {
				resp.Code = 6
				resp.Msg = err.Error()
				return
			}
		}
		if muteRecord != nil {
			userInfo.MuteStatus = 0
			if muteRecord.StartTime < tool.UnixNowMilli() && muteRecord.EndTime > tool.UnixNowMilli() {
				userInfo.MuteStatus = 1
			}
			userInfo.MuteReason = muteRecord.Reason
		}
		row5, err := easyDB.GetPlayerEntityByWhere[model.ArchitectureEntity](map[string]interface{}{"user_id": v.UserId, "type": pb.ArchitectureType_ARCHITECTURE_TYPE_MAIN})
		if err != nil {
			resp.Code = 7
			resp.Msg = err.Error()
			return
		}
		userInfo.Level = row5.Level
		respData = append(respData, userInfo)
	}
	resp.Data = respData
}

func GmGetFormation(req *GmGetFormationReq, resp *GmResp) {
	respData := resp.Data.(*GmGetFormationData)
	heroInfoList := make([]*HeroInfo, 0)

	var err error
	row1, err := easyDB.GetPlayerEntityByWhere[model.HeroFormationEntity](map[string]interface{}{"user_id": req.Uid, "formation_type": pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN, "is_active": true})
	if err != nil {
		resp.Code = 1
		resp.Msg = err.Error()
		return
	}
	for _, v := range row1.HeroOwnIDList {
		row2, err := easyDB.GetPlayerEntityByWhere[model.HeroDetailsEntity](map[string]interface{}{"hero_own_id": v})
		if err != nil {
			resp.Code = 2
			resp.Msg = err.Error()
			return
		}
		heroInfo := &HeroInfo{
			HeroId:    int32(row2.HeroID),
			Class:     row2.EvolutionPath,
			Level:     row2.Level,
			StarLevel: row2.StarLevel,
			Power:     row2.Power,
		}
		for _, v := range row2.EquipmentId {
			if v == 0 {
				continue
			}
			row3, err := easyDB.GetPlayerEntityByWhere[model.EquipmentEntity](map[string]interface{}{"equipment_own_id": v})
			if err != nil {
				resp.Code = 3
				resp.Msg = err.Error()
				return
			}
			cfg := gameConfig.GetEquipmentBaseCfg(row3.EquipmentID)
			if cfg == nil {
				continue
			}
			heroInfo.EquipId = append(heroInfo.EquipId, int32(row3.EquipmentID))
			heroInfo.EquipLevel = append(heroInfo.EquipLevel, int32(row3.Level))
			heroInfo.EquipQuality = append(heroInfo.EquipQuality, cfg.EquipmentQuality)
		}
		row4, err := easyDB.GetPlayerEntityByWhere[model.AccessoryEntity](map[string]interface{}{"user_id": req.Uid, "hero_own_id": v})
		if err != nil {
			if err.Error() != "record not found" {
				resp.Code = 4
				resp.Msg = err.Error()
				return
			}
		}
		if row4 != nil {
			cfg := gameConfig.GetAccessoryBaseCfg(row4.AccessoryId)
			if cfg == nil {
				continue
			}
			heroInfo.AccessoryInfo = &AccessoryInfo{
				AccessoryId:      row4.AccessoryId,
				AccessoryLevel:   row4.AccessoryLevel,
				AccessoryQuality: cfg.Quality,
			}
		}

		heroInfoList = append(heroInfoList, heroInfo)
	}

	respData.HeroInfoList = heroInfoList
}

func GmGetAccessory(req *GmGetAccessoryReq, resp *GmResp) {
	respData := resp.Data.(*GmGetAccessoryData)
	accessoryInfoList := make([]*AccessoryInfo, 0)
	row1, err := easyDB.GetPlayerEntitiesByWhere[model.AccessoryEntity](map[string]interface{}{"user_id": req.UserId})
	if err != nil {
		resp.Code = 1
		resp.Msg = err.Error()
		return
	}
	for _, v := range row1 {
		cfg := gameConfig.GetAccessoryBaseCfg(v.AccessoryId)
		if cfg == nil {
			continue
		}
		accessoryInfo := &AccessoryInfo{
			AccessoryId:      v.AccessoryId,
			AccessoryLevel:   v.AccessoryLevel,
			AccessoryQuality: cfg.Quality,
		}
		accessoryInfoList = append(accessoryInfoList, accessoryInfo)
	}

	respData.AccessoryInfo = accessoryInfoList
}

func GmServerList(req *GmServerListReq, resp *GmResp) {
	respData := resp.Data.([]*GmServerListData)
	row, err := easyDB.GetServerAllEntities[model.GameServerInfoEntity]()
	if err != nil {
		resp.Code = 1
		resp.Msg = err.Error()
		return
	}
	for _, v := range row {
		serverInfo := &GmServerListData{
			ServerNameId:     v.ServerNameId,
			ServerTime:       v.ServerTime,
			ServerLogicId:    v.ServerLogicId,
			MaxRegisterCount: v.MaxRegisterCount,
			OpenToNewWeight:  v.OpenToNewWeight,
			OpenToNew:        v.OpenToNew,
			CanSeeGroupId:    v.CanSeeGroupId,
			Status:           v.Status,

			ServerId:       v.ServerId,
			ServerName:     v.ServerName,
			ServerOpenTime: v.ServerOpenTime,
			AreaId:         v.AreaId,
			AreaName:       v.AreaName,
		}
		accountInfoList, err := easyDB.GetPlayerEntitiesByWhere[model.UserEntity](map[string]interface{}{"server_id": v.ServerId})
		if err != nil {
			resp.Code = 2
			resp.Msg = err.Error()
			return
		}
		now := time.Now()
		todayZero := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UnixMilli()
		for _, v := range accountInfoList {
			if v.LastLoginTime >= todayZero {
				serverInfo.TodayActPlayer++
			}
		}
		serverInfo.RegisterCount = int32(len(accountInfoList))
		respData = append(respData, serverInfo)
	}
	resp.Data = respData
}

func GmUserItemChg(req *GmUserItemChgReq, resp *GmResp) {
	respData := resp.Data.([]*GmUserItemData)
	where := make(map[string]interface{})

	// 只添加有值的字段
	if req.Uid != 0 {
		where["uid"] = req.Uid
	}
	if req.It != 0 {
		where["it"] = req.It
	}
	if req.Id != 0 {
		where["id"] = req.Id
	}
	if req.Ft != 0 {
		where["ft"] = req.Ft
	}
	if req.Fv != 0 {
		where["fv"] = req.Fv
	}
	if req.St != 0 {
		where["st"] = req.St
	}
	if req.Et != 0 {
		where["et"] = req.Et
	}
	rows, err := easyDB.LogGetEntitiesByWhere(where)
	if err != nil {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("get entities error=%s", err.Error())
		return
	}
	for _, v := range rows {
		info := &GmUserItemData{
			Uid:  toInt64(v["uid"]),
			T:    toInt64(v["t"]),
			It:   toInt32(v["it"]),
			Id:   toInt32(v["id"]),
			Ft:   toInt32(v["ft"]),
			Fv:   toInt32(v["fv"]),
			Ext:  toInt64(v["ext"]),
			Init: toString(v["init"]),
			Chg:  toString(v["chg"]),
			Fina: toString(v["fina"]),
		}
		respData = append(respData, info)
	}
	resp.Data = respData
}

// toInt64 安全地将 map 值转换为 int64（兼容 MySQL 驱动返回的多种整型）
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case uint64:
		return int64(val)
	case float64:
		return int64(val)
	}
	return 0
}

// toInt32 安全地将 map 值转换为 int32
func toInt32(v interface{}) int32 {
	return int32(toInt64(v))
}

// toString 安全地将 map 值转换为 string（兼容 MySQL 驱动返回 []byte 的情况）
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	}
	return fmt.Sprintf("%v", v)
}

func GmGetRankList(req *GmGetRankListReq, resp *GmResp) {
	respData := resp.Data.([]*GmGetRankListData)

	// 扫描数据库中所有 common_ 和 activity_ 开头的排行榜表
	allTables := make([]string, 0)

	commonTables, err := easyDB.GetRankTableNames("common_")
	if err != nil {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("scan common rank tables error: %s", err.Error())
		return
	}
	allTables = append(allTables, commonTables...)

	activityTables, err := easyDB.GetRankTableNames("activity_")
	if err != nil {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("scan activity rank tables error: %s", err.Error())
		return
	}
	allTables = append(allTables, activityTables...)

	// 按条件过滤表名
	for _, tableName := range allTables {
		// 解析表名获取信息
		parts := strings.Split(tableName, "_")
		if len(parts) < 3 {
			fmt.Printf("[DEBUG] Skip table %s: invalid parts\n", tableName)
			continue
		}

		var tableRankId, tableActId, tableServerId int32
		var tableDate string

		if parts[0] == "common" {
			// common_{rankId}_{serverId}_({date})
			if len(parts) >= 3 {
				rid, err1 := strconv.Atoi(parts[1])
				sid, err2 := strconv.Atoi(parts[2])
				if err1 != nil || err2 != nil {
					fmt.Printf("[DEBUG] Skip table %s: parse error rid=%v sid=%v\n", tableName, err1, err2)
					continue
				}
				tableRankId = int32(rid)
				tableServerId = int32(sid)
				tableActId = 0
				if len(parts) == 4 {
					tableDate = parts[3]
				}
			}
		} else if parts[0] == "activity" {
			// activity_{actId}_{rankId}_{serverId}_{date}
			if len(parts) >= 5 {
				actId, err1 := strconv.Atoi(parts[1])
				rid, err2 := strconv.Atoi(parts[2])
				sid, err3 := strconv.Atoi(parts[3])
				if err1 != nil || err2 != nil || err3 != nil {
					fmt.Printf("[DEBUG] Skip table %s: parse error\n", tableName)
					continue
				}
				tableActId = int32(actId)
				tableRankId = int32(rid)
				tableServerId = int32(sid)
				tableDate = parts[4]
			}
		} else {
			fmt.Printf("[DEBUG] Skip table %s: unknown prefix %s\n", tableName, parts[0])
			continue
		}

		fmt.Printf("[DEBUG] Table %s: rankId=%d actId=%d serverId=%d date=%s\n", tableName, tableRankId, tableActId, tableServerId, tableDate)

		// 按条件过滤（求并集）
		if req.RankId != 0 && tableRankId != req.RankId {
			fmt.Printf("[DEBUG] Filter by rankId: %d != %d\n", tableRankId, req.RankId)
			continue
		}
		if req.ActId != 0 && tableActId != req.ActId {
			fmt.Printf("[DEBUG] Filter by actId: %d != %d\n", tableActId, req.ActId)
			continue
		}
		if req.ServerId != 0 && tableServerId != req.ServerId {
			fmt.Printf("[DEBUG] Filter by serverId: %d != %d\n", tableServerId, req.ServerId)
			continue
		}
		// 日期过滤：如果指定了日期，只返回匹配的表（活动表优先，但 common 表有日期的也要返回）
		if req.Date != "" && tableDate != req.Date {
			fmt.Printf("[DEBUG] Filter by date: %s != %s\n", tableDate, req.Date)
			continue
		}

		respData = append(respData, &GmGetRankListData{
			TableName: tableName,
			RankId:    tableRankId,
			ActId:     tableActId,
			ServerId:  tableServerId,
			Date:      tableDate,
		})
	}

	resp.Data = respData
}

// checkRankTableExists 检查排行榜表是否存在（有数据）
func checkRankTableExists(tableName string) bool {
	// 直接从数据库查询，避免使用 rankboardService（可能未初始化）
	// 从数据库直接查询
	data, dbErr := easyDB.GetRankBoardData[model.RankBoardInfoEntity](tableName)
	if dbErr != nil {
		// 表不存在或其他错误
		return false
	}

	return len(data) > 0
}

func GmGetRank(req *GmGetRankReq, resp *GmResp) {
	respData := resp.Data.([]*GmGetRankData)

	if req.TableName == "" {
		resp.Code = 1
		resp.Msg = "table_name is required"
		return
	}

	// 直接按表名查询数据
	data, err := easyDB.GetRankBoardData[model.RankBoardInfoEntity](req.TableName)
	if err != nil {
		resp.Code = 2
		resp.Msg = fmt.Sprintf("get rank data error: %s", err.Error())
		return
	}

	for _, info := range data {
		rankData := &GmGetRankData{
			Rank:         info.Rank,
			UserId:       info.Id,
			Score:        info.Score,
			ThumbUpCount: info.ThumbUpCount,
			EnterTime:    info.EnterTime,
			UpdateTime:   info.UpdateTime,
		}
		// 查询用户昵称
		userInfo, userErr := easyDB.GetPlayerEntityByWhere[model.UserEntity](map[string]interface{}{"user_id": info.Id})
		if userErr == nil && userInfo != nil {
			rankData.NickName = userInfo.Nickname
		}
		respData = append(respData, rankData)
	}

	resp.Data = respData
}

func GmGetTalk(req *GmGetTalkReq, resp *GmResp) {
	respData := resp.Data.(*webProto.GetChatMessageResp)
	if req.ServerId == 0 {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("serverId is null")
		return
	}
	res := make([]*pb.PushReceivedChatMessage, 0)
	ctx := context.Background()

	messages, err := dbService.RDB.LRange(ctx, enum.GetChatKey(int32(enum.BROADCAST_TYPE_SERVER_ID), int64(req.ServerId)), 0, -1).Result()
	if err == nil {
		for _, msg := range messages {
			var msgPb *pb.PushReceivedChatMessage
			if err := json.Unmarshal([]byte(msg), &msgPb); err != nil {
				continue
			}
			if req.KeyWords != "" {
				if strings.Contains(strings.ToLower(msgPb.ChatMessage.MessageContent), req.KeyWords) {
					res = append(res, msgPb)
				}
			} else {
				res = append(res, msgPb)
			}
		}
	} else {
		resp.Code = 2
		resp.Msg = err.Error()
		return
	}

	respData.MsgList = res
}

func GmUserMail(req *GmUserMailReq, resp *GmResp) {
	respData := resp.Data.([]*GmUserMailData)

	where := make(map[string]interface{})
	where["user_id"] = req.Uid
	row1, err := easyDB.GetPlayerEntitiesByWhere[mail.MailEntity](where)
	if err != nil {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("get entities error=%s", err.Error())
		return
	}
	for _, v := range row1 {
		if (req.Sts != 0 && v.SendTime < req.Sts) || (req.Ste != 0 && v.SendTime > req.Ste) {
			continue
		}
		if v.ExpireTime == 0 && (req.Ets != 0 || req.Ete != 0) {
			continue
		} else if (req.Ets != 0 && v.ExpireTime < req.Ets) || (req.Ete != 0 && v.ExpireTime > req.Ete) {
			continue
		}
		mailInfo := &GmUserMailData{
			Mid:   v.MailID,
			Mtp:   v.MailType,
			CfgId: v.TemplateID,
			Title: v.Title,
			Ms:    v.Status,
			Ct:    v.SendTime,
			Ot:    v.ExpireTime,
			IsExpired: func() int32 {
				if tool.UnixNowMilli() > v.ExpireTime {
					return 1
				}
				return 0
			}(),
			Del: func() int32 {
				if v.Status == 3 {
					return 1
				}
				return 0
			}(),
		}
		var attachments []*mail.MailAttachmentItem
		if v.Attachments != "" {
			if err := json.Unmarshal([]byte(v.Attachments), &attachments); err != nil {
				// 记录错误或返回错误
				resp.Code = 2
				resp.Msg = err.Error()
				return
			}
		}
		mailInfo.Items = attachments
		respData = append(respData, mailInfo)
	}
	resp.Data = respData
}

func GmGamePublic(req *GmGamePublicReq, resp *GmResp) {
	respData := resp.Data.([]*GmGamePublicData)
	row, err := easyDB.GetServerAllEntities[model.AnnounceInfoEntity]()
	if err != nil {
		resp.Code = 1
		resp.Msg = err.Error()
		return
	}
	for _, v := range row {
		serverInfo := &GmGamePublicData{
			Id:           v.Id,
			AnnounceType: v.AnnounceType,
			Title:        v.Title,
			Content:      v.Content,
			ShowType:     v.ShowType,
			PicAddress:   v.PicAddress,
			ServerIds:    v.ServerIds,
			Unlocks:      v.Unlocks,
			UnlockStop:   v.UnlockStop,
			StartTime:    v.StartTime,
			EndTime:      v.EndTime,
			Valid:        v.Valid,
			ExtraInfo:    v.ExtraInfo,
		}
		respData = append(respData, serverInfo)
	}
	resp.Data = respData
}

func GmUserOrder(req *GmUserOrderReq, resp *GmResp) {
	respData := resp.Data.([]*GmUserOrderData)
	where := make(map[string]interface{})
	orderStatusMap := make(map[int32]bool)
	for _, value := range req.Msl {
		orderStatusMap[value] = true
	}

	if req.Uid != 0 {
		if req.T == 0 {
			where["user_id"] = req.Uid
		} else {
			where["order_id"] = req.Uid
		}
	}

	rows, err := easyDB.GetServerEntitiesByWhere[model.RechargeOrderEntity](where)
	if err != nil {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("get entities error=%s", err.Error())
		return
	}

	for _, v := range rows {
		// 检查时间区间过滤条件
		if req.Cts != 0 && v.CreateTime < req.Cts {
			continue
		}
		if req.Cte != 0 && v.CreateTime > req.Cte {
			continue
		}
		if req.Pts != 0 && v.PayTime < req.Pts {
			continue
		}
		if req.Pte != 0 && v.PayTime > req.Pte {
			continue
		}
		if len(orderStatusMap) > 0 && !orderStatusMap[v.Status] {
			continue
		}
		orderInfo := &GmUserOrderData{
			Oid:   v.OrderId,    // 订单ID转为字符串
			Uid:   v.UserId,     // 用户ID
			Goods: v.ShopItemId, // 商品ID
			P:     v.Price,      // 价格
			Ct:    v.CreateTime, // 创建时间（毫秒时间戳）
			S:     v.Status,     // 订单状态
			Ot:    v.PayTime,    // 支付时间（毫秒时间戳）
			Pt:    v.PayType,    // 支付方式（暂时用状态代替，可根据实际需求调整）
			Bz:    v.ExtraInfo,  // 备注
		}
		respData = append(respData, orderInfo)
	}
	resp.Data = respData
}

func GmGetClientVersion(req *GmGetClientVersionReq, resp *GmResp) {
	respData := resp.Data.([]*GmClientVersionData)
	rows, err := easyDB.GetServerAllEntities[model.GameClientVersionEntity]()
	if err != nil {
		resp.Code = 1
		resp.Msg = err.Error()
		return
	}
	for _, v := range rows {
		respData = append(respData, &GmClientVersionData{
			Version:      v.Version,
			HotFixConfig: v.HotfixConfig,
			Examine:      v.Examine,
		})
	}
	resp.Data = respData
}

// 以下函数为：修改数据库操作
func GmSendMail(req *GmSendMailReq, resp *GmResp) {
	if req == nil || req.Info == nil {
		resp.Code = 1
		resp.Msg = "invalid request"
		return
	}

	db := easyDB.GetPlayerDB()
	if db == nil {
		resp.Code = 1
		resp.Msg = "db not initialized"
		return
	}

	now := tool.UnixNow()
	expireTime := int64(0)
	if req.Info.Ot > 0 {
		expireTime = req.Info.Ot / 1000
	} else if req.Info.ExpireDays > 0 {
		expireTime = now + int64(req.Info.ExpireDays)*24*3600
	}
	if expireTime > 0 && expireTime <= now {
		resp.Code = 1
		resp.Msg = "expire time must be in the future"
		return
	}

	items := convertPbMailItems(req.Info.Items)

	// 服务邮件：全服(1)、区服(2)、联盟(3)
	type mp struct {
		SID int32
		AID int64
	}
	var ps []mp
	switch req.SendType {
	case 0:
		for _, userID := range req.Ul {
			if userID <= 0 {
				continue
			}
			entity := mail.MailToEntity(&mail.Mail{
				MailID:       backendMailIdGenerator.NextId(),
				UserID:       userID,
				MailType:     req.Info.Mtp,
				Title:        req.Info.Title,
				Content:      req.Info.Content,
				SenderName:   req.Info.Sn,
				SenderAvatar: req.Info.Sa,
				TemplateID:   req.Info.CfgId,
				Status:       mail.MailStatusUnread,
				IsConvenient: req.Info.IsConvenient,
				Items:        items,
				ExpireTime:   expireTime,
				SendTime:     now,
			})
			if err := db.Create(entity).Error; err != nil {
				resp.Code = 1
				resp.Msg = err.Error()
				return
			}
			_ = mail.NotifyMailRefresh(userID)
		}
		resp.Code = 0
		resp.Msg = "ok"
		resp.Data = map[string]any{"sent": len(req.Ul)}
		return
	case 2:
		if len(req.ServerIds) == 0 {
			resp.Code = 1
			resp.Msg = "serverIds is required for sendType=2"
			return
		}
		for _, sid := range req.ServerIds {
			ps = append(ps, mp{sid, 0})
		}
	case 3:
		if len(req.AllianceIds) == 0 {
			resp.Code = 1
			resp.Msg = "allianceIds is required for sendType=3"
			return
		}
		for _, aid := range req.AllianceIds {
			ps = append(ps, mp{0, aid})
		}
	default:
		ps = append(ps, mp{0, 0})
	}

	var ids []int64
	for _, p := range ps {
		id := backendMailIdGenerator.NextId()
		entity := mail.ServerMailToEntity(&mail.ServerMail{
			ServerMailID: id,
			MailType:     req.Info.Mtp,
			Title:        req.Info.Title,
			Content:      req.Info.Content,
			TemplateID:   req.Info.CfgId,
			ServerID:     p.SID,
			AllianceID:   p.AID,
			SenderAvatar: req.Info.Sa,
			UnlockList:   []int32{},
			IsConvenient: req.Info.IsConvenient,
			Items:        items,
			SendTime:     now,
			ExpireTime:   expireTime,
			Status:       mail.ServerMailStatusSent,
			CreatedBy:    "backend",
		})
		if err := db.Create(entity).Error; err != nil {
			resp.Code = 1
			resp.Msg = err.Error()
			return
		}
		ids = append(ids, id)
	}
	_ = mail.NotifyServerMailRefresh()
	resp.Code = 0
	resp.Msg = "ok"
	resp.Data = map[string]any{"server_mail_ids": ids}
}

func GmEditServer(req *GmEditServerReq, resp *GmResp) {
	serverEntity := &model.GameServerInfoEntity{
		ServerId:         req.Info.ServerId,
		ServerName:       req.Info.ServerName,
		ServerNameId:     req.Info.ServerId,
		ServerOpenTime:   req.Info.ServerOpenTime,
		ServerTime:       req.Info.ServerTime,
		ServerLogicId:    req.Info.ServerId,
		AreaId:           req.Info.AreaId,
		AreaName:         req.Info.AreaName,
		MaxRegisterCount: req.Info.MaxRegisterCount,
		OpenToNewWeight:  req.Info.OpenToNewWeight,
		OpenToNew:        req.Info.OpenToNew,
		CanSeeGroupId:    req.Info.CanSeeGroupId,
		Status:           req.Info.Status,
	}

	// 查询是否已存在
	_, err := easyDB.GetServerEntityByWhere[model.GameServerInfoEntity](map[string]interface{}{"server_id": req.Info.ServerId})
	if err != nil {
		// 不存在则创建
		if req.Info.CanSeeGroupId == 0 {
			serverEntity.CanSeeGroupId = serverEntity.ServerId
		}
		if req.Info.OpenToNewWeight == 0 {
			serverEntity.OpenToNewWeight = serverEntity.ServerId
		}
		if req.Info.ServerName == "" {
			serverEntity.ServerName = fmt.Sprintf("服务器%d", serverEntity.ServerId)
		}
		createErr := httpPlatform.GetServerInfoService().AddServerInfo(serverEntity)
		if createErr != nil {
			resp.Code = 1
			resp.Msg = createErr.Error()
			return
		}
	} else {
		// 存在则更新
		updateErr := httpPlatform.GetServerInfoService().UpdateServerInfo(serverEntity)
		if updateErr != nil {
			resp.Code = 2
			resp.Msg = updateErr.Error()
			return
		}
	}
	row, err := easyDB.GetServerEntityByWhere[model.GameServerInfoEntity](map[string]interface{}{"server_id": req.Info.ServerId})
	if err != nil {
		resp.Code = 1
		resp.Msg = err.Error()
		return
	}
	serverEntity.ServerLogicId = row.ServerId
	serverEntity.ServerNameId = row.ServerId
	updateErr := httpPlatform.GetServerInfoService().UpdateServerInfo(serverEntity)
	if updateErr != nil {
		resp.Code = 2
		resp.Msg = updateErr.Error()
		return
	}
}

func GmEditGamePublic(req *GmEditGamePublicReq, resp *GmResp) {
	if req == nil || req.Data == nil {
		resp.Code = 1
		resp.Msg = "invalid request: data is nil"
		return
	}

	publicEnity := &model.AnnounceInfoEntity{
		Id:           req.Data.Id,
		AnnounceType: req.Data.AnnounceType,
		Title:        req.Data.Title,
		Content:      req.Data.Content,
		ShowType:     req.Data.ShowType,
		PicAddress:   req.Data.PicAddress,
		ServerIds:    req.Data.ServerIds,
		Unlocks:      req.Data.Unlocks,
		UnlockStop:   req.Data.UnlockStop,
		StartTime:    req.Data.StartTime,
		EndTime:      req.Data.EndTime,
		Valid:        req.Data.Valid,
		ExtraInfo:    req.Data.ExtraInfo, // 数据库没有这个字段，暂时注释
	}

	// 查询是否已存在
	_, err := easyDB.GetServerEntityByWhere[model.AnnounceInfoEntity](map[string]interface{}{"id": req.Data.Id})
	if err != nil {
		// 不存在则创建
		createErr := httpPlatform.GetServerInfoService().AddAnnounceInfo(publicEnity)
		if createErr != nil {
			resp.Code = 1
			resp.Msg = createErr.Error()
			return
		}
	} else {
		// 存在则更新
		updateErr := httpPlatform.GetServerInfoService().UpdateAnnounceInfo(publicEnity)
		if updateErr != nil {
			resp.Code = 2
			resp.Msg = updateErr.Error()
			return
		}
	}
	rpcController.SendOperationToGateway(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_ANNOUNCE_INFO_UPDATE, 0)
}

func GmEditClientVersion(req *GmEditClientVersionReq, resp *GmResp) {
	// 判断是否有文件上传（通过 UploadFile 字段判断）
	if req.ClientVersionList.UploadFile != nil {
		// 保存上传的文件
		file := req.ClientVersionList.UploadFile
		saveDir := "../../gameClientHotFixCfg"
		if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
			resp.Code = -1
			resp.Msg = fmt.Sprintf("create dir error: %s", err.Error())
			return
		}

		filePath := filepath.Join(saveDir, file.Filename)
		out, err := os.Create(filePath)
		if err != nil {
			resp.Code = -1
			resp.Msg = fmt.Sprintf("create file error: %s", err.Error())
			return
		}
		defer out.Close()

		// 打开上传的文件流
		src, err := file.Open()
		if err != nil {
			resp.Code = -1
			resp.Msg = fmt.Sprintf("open file error: %s", err.Error())
			return
		}
		defer src.Close()

		if _, err := io.Copy(out, src); err != nil {
			resp.Code = -1
			resp.Msg = fmt.Sprintf("save file error: %s", err.Error())
			return
		}

		// 获取服务器 IP
		serverIP, err := tool.GetLocalIP()
		if err != nil || serverIP == "" {
			serverIP = "127.0.0.1"
		}

		// 拼接 URL
		req.ClientVersionList.HotFixConfig = fmt.Sprintf("http://%s/gameClientHotFixCfg/%s", serverIP, file.Filename)
	}

	clientVersionEntity := &model.GameClientVersionEntity{
		Version:      req.ClientVersionList.Version,
		HotfixConfig: req.ClientVersionList.HotFixConfig,
		Examine:      req.ClientVersionList.Examine,
	}

	// 查询是否已存在
	existingEntity, err := easyDB.GetServerEntityByWhere[model.GameClientVersionEntity](map[string]interface{}{"version": req.ClientVersionList.Version})
	if err != nil {
		// 不存在则创建
		createErr := httpPlatform.GetServerInfoService().AddClientVersion(clientVersionEntity)
		if createErr != nil {
			resp.Code = 1
			resp.Msg = createErr.Error()
		}
	} else {
		// 存在则更新（修改查询到的 entity 字段）
		existingEntity.HotfixConfig = req.ClientVersionList.HotFixConfig
		existingEntity.Examine = req.ClientVersionList.Examine
		updateErr := httpPlatform.GetServerInfoService().UpdateClientVersion(existingEntity)
		if updateErr != nil {
			resp.Code = 2
			resp.Msg = updateErr.Error()
		}
	}
}

func GmGetUserInventory(req *GmGetUserInventoryReq, resp *GmResp) {
	respData := resp.Data.([]*GmInventoryData)

	// 从数据库查询玩家背包数据
	rows, err := easyDB.GetPlayerEntitiesByWhere[inventory.PlayerInventoryEntity](map[string]interface{}{"user_id": req.Uid})
	if err != nil {
		resp.Code = 1
		resp.Msg = err.Error()
		return
	}

	for _, v := range rows {
		itemCfg := gameConfig.GetItemCfg(v.ItemId)
		if itemCfg == nil {
			continue
		}
		if itemCfg.Type == 1 || itemCfg.Type == 2 || itemCfg.Type == 3 {
			respData = append(respData, &GmInventoryData{
				ItemId:  v.ItemId,
				ItemNum: v.ItemNum,
			})
		}
	}
	resp.Data = respData
}

func GmGetServerActivityConfig(req *GmGetServerActivityConfigReq, resp *GmResp) {
	respData := resp.Data.([]*GmServerActivityConfigData)

	// 从数据库查询服务器活动配置
	rows, err := easyDB.GetServerAllEntities[model.ServerActivityConfigEntity]()
	if err != nil {
		resp.Code = 1
		resp.Msg = err.Error()
		return
	}

	for _, v := range rows {
		respData = append(respData, &GmServerActivityConfigData{
			Id:             v.Id,
			ServerType:     v.ServerType,
			ServerUnit:     v.ServerUnit,
			UnlockId:       v.UnlockId,
			AttendUnlockId: v.AttendUnlockId,
			EventOpen:      v.EventOpen,
			EventEnd:       v.EventEnd,
			WeekOpen:       v.WeekOpen,
			MonthOpen:      v.MonthOpen,
			Duration:       v.Duration,
			SettleTime:     v.SettleTime,
			IfFirst:        v.IfFirst,
			NextId:         v.NextId,
			Cd:             v.Cd,
			OpenLoopNum:    v.OpenLoopNum,
			IfBlockServer:  v.IfBlockServer,
			IfBlock:        v.IfBlock,
		})
	}
	resp.Data = respData
}

func GmEditServerActivityConfig(req *GmEditServerActivityConfigReq, resp *GmResp) {
	if req == nil || req.Data == nil {
		resp.Code = 1
		resp.Msg = "invalid request: data is nil"
		return
	}

	entity := &model.ServerActivityConfigEntity{
		Id:             req.Data.Id,
		ServerType:     req.Data.ServerType,
		ServerUnit:     req.Data.ServerUnit,
		UnlockId:       req.Data.UnlockId,
		AttendUnlockId: req.Data.AttendUnlockId,
		EventOpen:      req.Data.EventOpen,
		EventEnd:       req.Data.EventEnd,
		WeekOpen:       req.Data.WeekOpen,
		MonthOpen:      req.Data.MonthOpen,
		Duration:       req.Data.Duration,
		SettleTime:     req.Data.SettleTime,
		IfFirst:        req.Data.IfFirst,
		NextId:         req.Data.NextId,
		Cd:             req.Data.Cd,
		OpenLoopNum:    req.Data.OpenLoopNum,
		IfBlockServer:  req.Data.IfBlockServer,
		IfBlock:        req.Data.IfBlock,
	}

	// 先查后判
	_, err := easyDB.GetServerEntityByWhere[model.ServerActivityConfigEntity](map[string]interface{}{"id": req.Data.Id})
	if err != nil {
		// 不存在，创建新记录
		if createErr := easyDB.CreateServerEntity(entity); createErr != nil {
			resp.Code = 2
			resp.Msg = createErr.Error()
			return
		}
	} else {
		// 存在，更新
		if updateErr := easyDB.SaveSeverEntity(entity); updateErr != nil {
			resp.Code = 3
			resp.Msg = updateErr.Error()
			return
		}
	}
}

func GmEditBanUser(req *GmEditBanUserReq, resp *GmResp) {
	if req.BanInfo.Account == "" {
		resp.Code = 1
		resp.Msg = "account is required"
		return
	}

	entity := &model.BanListEntity{
		Account:   req.BanInfo.Account,
		ServerId:  req.BanInfo.ServerId,
		Reason:    req.BanInfo.Reason,
		StartTime: req.BanInfo.StartTime,
		EndTime:   req.BanInfo.EndTime,
	}

	// 先查后判（BanList表在server库）
	existingEntity, err := easyDB.GetServerEntityByWhere[model.BanListEntity](map[string]interface{}{"account": req.BanInfo.Account})
	if err != nil {
		// 不存在，创建新记录
		if createErr := easyDB.CreateServerEntity(entity); createErr != nil {
			resp.Code = 2
			resp.Msg = createErr.Error()
			return
		}
	} else {
		// 存在，更新
		existingEntity.ServerId = req.BanInfo.ServerId
		existingEntity.Reason = req.BanInfo.Reason
		existingEntity.StartTime = req.BanInfo.StartTime
		existingEntity.EndTime = req.BanInfo.EndTime
		if updateErr := easyDB.SaveSeverEntity(existingEntity); updateErr != nil {
			resp.Code = 3
			resp.Msg = updateErr.Error()
			return
		}
	}

	resp.Data = &GmBanUserData{
		Account:   req.BanInfo.Account,
		ServerId:  req.BanInfo.ServerId,
		Reason:    req.BanInfo.Reason,
		StartTime: req.BanInfo.StartTime,
		EndTime:   req.BanInfo.EndTime,
	}
}

func GmEditUserChat(req *GmEditUserChatReq, resp *GmResp) {
	if req.UserChatData.Account == "" {
		resp.Code = 1
		resp.Msg = "account is required"
		return
	}

	entity := &model.UserBanRecordEntity{
		Account:   req.UserChatData.Account,
		ServerId:  req.UserChatData.ServerId,
		Reason:    req.UserChatData.Reason,
		StartTime: req.UserChatData.StartTime,
		EndTime:   req.UserChatData.EndTime,
	}

	// 先查后判（UserBanRecordEntity表在server库）
	existingEntity, err := easyDB.GetServerEntityByWhere[model.UserBanRecordEntity](map[string]interface{}{"account": req.UserChatData.Account, "server_id": req.UserChatData.ServerId})
	if err != nil {
		// 不存在，创建新记录
		if createErr := easyDB.CreateServerEntity(entity); createErr != nil {
			resp.Code = 2
			resp.Msg = createErr.Error()
			return
		}
	} else {
		// 存在，更新
		existingEntity.ServerId = req.UserChatData.ServerId
		existingEntity.Reason = req.UserChatData.Reason
		existingEntity.StartTime = req.UserChatData.StartTime
		existingEntity.EndTime = req.UserChatData.EndTime
		if updateErr := easyDB.SaveSeverEntity(existingEntity); updateErr != nil {
			resp.Code = 3
			resp.Msg = updateErr.Error()
			return
		}
	}

	resp.Data = &GmEditUserChatData{
		Account:   req.UserChatData.Account,
		ServerId:  req.UserChatData.ServerId,
		Reason:    req.UserChatData.Reason,
		StartTime: req.UserChatData.StartTime,
		EndTime:   req.UserChatData.EndTime,
	}
}

func GmGetUserLogList(req *GmGetUserLogListReq, resp *GmResp) {
	respData := resp.Data.([]*GmUserLogData)

	where := make(map[string]interface{})
	where["user_id"] = req.UserId

	if req.OperationType != -1 {
		where["operation_type"] = req.OperationType
	}
	if req.St != 0 {
		where["st"] = req.St
	}
	if req.Et != 0 {
		where["et"] = req.Et
	}

	entities, err := easyDB.OperLogGetEntitiesByWhere(where, 1000)
	if err != nil {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("get user log list error=%s", err.Error())
		return
	}

	for _, entity := range entities {
		data := &GmUserLogData{}
		if v, ok := entity["user_id"]; ok {
			data.UserId = toInt64(v)
		}
		if v, ok := entity["add_time"]; ok {
			data.AddTime = toInt64(v)
		}
		if v, ok := entity["operation_type"]; ok {
			data.OperationType = toInt32(v)
		}
		if v, ok := entity["param1"]; ok {
			data.Param1 = toInt32(v)
		}
		if v, ok := entity["param2"]; ok {
			data.Param2 = toInt32(v)
		}
		if v, ok := entity["param3"]; ok {
			data.Param3 = toInt32(v)
		}
		if v, ok := entity["param4"]; ok {
			data.Param4 = toInt32(v)
		}
		respData = append(respData, data)
	}
	resp.Data = respData
}

func GmExportPlayer(req *GmExportPlayerReq, resp *GmResp) {
	respData := resp.Data.(*GmExportPlayerData)

	if req.UserId <= 0 {
		resp.Code = 1
		resp.Msg = "user_id is required"
		return
	}

	// 使用结构化导出
	jsonStr, err := ExportPlayerStructured(req.UserId)
	if err != nil {
		resp.Code = 2
		resp.Msg = fmt.Sprintf("export player error: %s", err.Error())
		return
	}

	respData.Json = jsonStr
	resp.Data = respData
}

func GmImportPlayer(req *GmImportPlayerReq, resp *GmResp) {
	respData := resp.Data.(*GmImportPlayerData)

	// 新版：结构化 JSON 导入
	if req.Json != "" {
		if req.TargetAccount == "" || req.TargetServerId <= 0 {
			resp.Code = 1
			resp.Msg = "target_account and target_server_id are required"
			return
		}
		newUserId, oldUserId, err := ImportPlayerStructured(req.Json, req.TargetAccount, req.TargetServerId, req.CreateTime)
		if err != nil {
			resp.Code = 2
			resp.Msg = fmt.Sprintf("import error: %s", err.Error())
			return
		}
		respData.UserId = newUserId
		respData.OldUserId = oldUserId
		respData.Msg = "import success"
		resp.Data = respData
		return
	}

	// 兼容旧版：直接执行 SQL
	if req.Sql == "" {
		resp.Code = 1
		resp.Msg = "json or sql is required"
		return
	}

	db := easyDB.GetPlayerDB()
	tx := db.Begin()
	if tx.Error != nil {
		resp.Code = 2
		resp.Msg = fmt.Sprintf("begin transaction error: %s", tx.Error.Error())
		return
	}

	lines := strings.Split(req.Sql, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if err := tx.Exec(line).Error; err != nil {
			tx.Rollback()
			resp.Code = 3
			resp.Msg = fmt.Sprintf("exec sql error: %s", err.Error())
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		resp.Code = 4
		resp.Msg = fmt.Sprintf("commit error: %s", err.Error())
		return
	}

	respData.UserId = 0
	resp.Data = respData
}

func GmGetThroughput(req *GmGetThroughputReq, resp *GmResp) {
	respData := resp.Data.([]*GmThroughputItem)
	ctx := context.Background()

	// 使用 SCAN 模糟查询所有 throughput:* 的 key
	var cursor uint64
	var allKeys []string
	for {
		keys, nextCursor, err := dbService.RDB.Scan(ctx, cursor, "throughput:*", 100).Result()
		if err != nil {
			resp.Code = 1
			resp.Msg = fmt.Sprintf("scan redis error: %s", err.Error())
			return
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	// 遍历每个 key，获取对应的值并解析 handled:received
	for _, key := range allKeys {
		val, err := dbService.RDB.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		// 格式: "handled:received"
		parts := strings.SplitN(val, ":", 2)
		if len(parts) != 2 {
			continue
		}
		handled, err1 := strconv.ParseInt(parts[0], 10, 64)
		received, err2 := strconv.ParseInt(parts[1], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		respData = append(respData, &GmThroughputItem{
			Key:      key,
			Handled:  handled,
			Received: received,
		})
	}

	resp.Data = respData
}

func GmGetServerActivityOpen(req *GmGetServerActivityOpenReq, resp *GmResp) {
	respData := resp.Data.(*GmGetServerActivityOpenData)

	if req.ServerId <= 0 {
		resp.Code = 1
		resp.Msg = "server_id is required and must be greater than 0"
		return
	}

	// 从数据库查询服务器开启的所有活动
	activities, err := easyDB.GetServerEntitiesByWhere[model.ServerOpenActivityEntity](map[string]interface{}{
		"open_server_id": req.ServerId,
	})
	if err != nil {
		resp.Code = 2
		resp.Msg = fmt.Sprintf("query activity error: %s", err.Error())
		respData.ServerId = req.ServerId
		respData.Activities = make([]*ServerOpenActivityItem, 0)
		return
	}

	// 转换为响应格式
	respData.ServerId = req.ServerId
	respData.Activities = make([]*ServerOpenActivityItem, 0, len(activities))
	for _, activity := range activities {
		respData.Activities = append(respData.Activities, &ServerOpenActivityItem{
			ActivityId:   activity.ActivityId,
			Version:      activity.Version,
			OpenServerId: activity.OpenServerId,
			OpenTime:     activity.OpenTime,
			SettleTime:   activity.SettleTime,
			EndTime:      activity.EndTime,
			OpenCount:    activity.OpenCount,
		})
	}
}

// GmKickPlayer 踢人操作
func GmKickPlayer(req *GmKickPlayerReq, resp *GmResp) {
	if req.Type == 1 {
		rpcController.SendOperationToGateway(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_KICK_PLAYER_OUT, int64(req.Param))
	} else if req.Type == 2 {
		rpcController.SendOperationToGateway(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_KICK_SERVER_PLAYER_OFFLINE, int64(req.Param))
	} else {
		rpcController.SendOperationToGateway(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_KICK_ALL_PLAYER_OFFLINE, 0)
	}
}
