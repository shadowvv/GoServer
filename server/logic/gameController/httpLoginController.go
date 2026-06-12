package gameController

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/httpPlatform"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/webProto"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"gorm.io/gorm"
)

func RegisterWebMessage() {
	httpPlatform.RegisterHttpMessage("/serverInfo", handleGameServerInfoRequest)
	httpPlatform.RegisterHttpMessage("/login", handleLogin)
	httpPlatform.RegisterHttpMessage("/announceInfo", handleAnnounceInfoRequest)
	httpPlatform.RegisterHttpMessage("/getChatMessage", handleGetChatMessageRequest)
}

func handleGameServerInfoRequest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	// 防止恶意请求超大 Body
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req webProto.ServerListReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	if req.Account == "" {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := webProto.ServerListResp{}
	servers := httpPlatform.GetServerInfoService().GetAllServerInfo()
	respList := make([]*webProto.ServerInfo, 0)
	respServerId := make(map[int32]bool)

	playerInfos, err := easyDB.GetPlayerEntitiesByWhere[model.UserEntity](
		map[string]interface{}{"account": req.Account},
	)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.ErrorWithZapFields(fmt.Sprintf("[web] db error: %v", err))
		sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
		return
	}

	regList := make([]*webProto.UserSimpleInfo, 0, len(playerInfos))

	for _, v := range playerInfos {
		currentPlayerInfo := httpPlatform.GetServerInfoService().GetServerInfo(v.ServerId)
		if currentPlayerInfo == nil {
			continue
		}
		for _, targetServerInfo := range servers {
			// 只有可见的组和新服可以看到
			if targetServerInfo.CanSeeGroupId == currentPlayerInfo.CanSeeGroupId || (currentPlayerInfo.OpenToNew == 1 && targetServerInfo.OpenToNewWeight == httpPlatform.GetServerInfoService().GetNewServerWeight()) {
				if targetServerInfo.OpenToNew == 1 && targetServerInfo.ServerOpenTime < tool.UnixNowMilli() {
					if _, ok := respServerId[targetServerInfo.ServerId]; !ok {
						respServerId[targetServerInfo.ServerId] = true
						respList = append(respList, &webProto.ServerInfo{
							ID:       targetServerInfo.ServerId,
							Name:     targetServerInfo.ServerNameId,
							AreaID:   targetServerInfo.AreaId,
							Status:   targetServerInfo.Status,
							OpenTime: targetServerInfo.GetServerOpenTime(),
						})
					}
				}
			}
		}
		// 玩家等级与主城等级一致，主城等级从0级开始，但是不能显示玩家等级为0
		if v.Level == 0 {
			v.Level = 1
		}
		regList = append(regList, &webProto.UserSimpleInfo{
			UserId:        v.UserId,
			Nickname:      v.Nickname,
			ServerID:      v.ServerId,
			Frame:         v.HeadFrameId,
			Head:          v.HeadId,
			LastLoginTime: v.LastLoginTime,
			Level:         v.Level,
		})
	}

	resp.Data = &webProto.AllServerInfo{
		List:    respList,
		RegList: regList,
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(resp)
}

func sendErrorMessage(errorCode pb.ERROR_CODE, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&webProto.WebErrorMessage{
		Code: int32(errorCode),
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	var req webProto.LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	logger.InfoWithSprintf("[web] login req: %v", req)

	if !checkWhiteList(&req) {
		announced := getBlockAnnounce(req.ServerID)
		if announced != nil {
			resp := &webProto.LoginResponse{
				Announce: &webProto.AnnounceInfo{
					ID:         announced.Id,
					Title:      announced.Title,
					Content:    announced.Content,
					Type:       announced.AnnounceType,
					ShowType:   announced.ShowType,
					PicAddress: announced.PicAddress,
					StartTime:  announced.StartTime,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			logger.InfoWithSprintf("[web] login block: %v,account:%s,serverId:%d", announced, req.Account, req.ServerID)
			return
		}
	}

	// 查找账号上次登录的服务器
	accountSimpleEntity, err := easyDB.GetPlayerEntityByWhere[model.AccountSimpleInfoEntity](map[string]interface{}{"account": req.Account})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.InfoWithSprintf("[web] login create accountSimpleInfo: %s", req.Account)
			// 新号没用登录记录
			req.ServerID = httpPlatform.GetServerInfoService().GetDefaultServerId()
			if req.ServerID == 0 {
				sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
				logger.InfoWithSprintf("[web] login no valid serverId: %d,account:%s", req.ServerID, req.Account)
				return
			}
			accountSimpleEntity = &model.AccountSimpleInfoEntity{
				Account:           req.Account,
				LastLoginServerId: req.ServerID,
			}
			err = easyDB.CreatePlayerEntity[model.AccountSimpleInfoEntity](accountSimpleEntity)
			if err != nil {
				logger.ErrorWithZapFields(fmt.Sprintf("[web] login create accountSimpleInfo db error: %v,account:%s,serverId:%d", err, req.Account, req.ServerID))
				sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
				return
			}
		} else {
			// 数据库问题
			logger.ErrorWithZapFields(fmt.Sprintf("[web] login select accountSimpleInfo db error: %v,account:%s,serverId:%d", err, req.Account, req.ServerID))
			sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
			return
		}
	}

	if req.ServerID == 0 {
		// 玩家没用选择服务器，尝试使用上次登录的服务器
		logger.InfoWithSprintf("[web] login get serverId account: %s", req.Account)
		if accountSimpleEntity.LastLoginServerId == 0 {
			req.ServerID = httpPlatform.GetServerInfoService().GetDefaultServerId()
			if req.ServerID == 0 {
				sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
				logger.ErrorBySprintf("[web] login no valid serverId: %d,account:%s", req.ServerID, req.Account)
				return
			}
			accountSimpleEntity.LastLoginServerId = req.ServerID
		} else {
			req.ServerID = accountSimpleEntity.LastLoginServerId
		}
	} else {
		// 玩家选择服务器，判断目标服务器是否有账号
		logger.InfoWithSprintf("[web] login get account with serverId account: %s,serverId:%d", req.Account, req.ServerID)
		_, err = easyDB.GetPlayerEntityByWhere[model.UserEntity](map[string]interface{}{"account": req.Account, "server_id": req.ServerID})
		if err == nil {
			accountSimpleEntity.LastLoginServerId = req.ServerID
		} else {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 账号没有这个服务器的账号,判断目标服务器是否可以进入
				logger.InfoWithSprintf("[web] login no last login serverId account with serverId account: %s,serverId:%d", req.Account, req.ServerID)
				targetServerInfo := httpPlatform.GetServerInfoService().GetServerInfo(req.ServerID)
				if targetServerInfo == nil {
					sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
					logger.InfoWithSprintf("[web] login target serverId not valid serverId: %d,account:%s", req.ServerID, req.Account)
					return
				}
				fromServerInfo := httpPlatform.GetServerInfoService().GetServerInfo(accountSimpleEntity.LastLoginServerId)
				if fromServerInfo == nil {
					sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
					logger.InfoWithSprintf("[web] login from serverId not valid serverId: %d,account:%s,lastLoginServerId:%d", req.ServerID, req.Account, accountSimpleEntity.LastLoginServerId)
					return
				}
				// 可见判断
				if (targetServerInfo.CanSeeGroupId != fromServerInfo.CanSeeGroupId && targetServerInfo.OpenToNewWeight != httpPlatform.GetServerInfoService().GetNewServerWeight()) || targetServerInfo.OpenToNew != 1 || targetServerInfo.ServerOpenTime > tool.UnixNowMilli() {
					sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
					logger.InfoWithSprintf("[web] login target serverId not valid serverId: %d,account:%s,lastLoginServerId:%d", req.ServerID, req.Account, accountSimpleEntity.LastLoginServerId)
					return
				}
			} else {
				// 账号数据库问题
				logger.ErrorWithZapFields(fmt.Sprintf("[web] login select accountSimpleInfo db error: %v,account:%s,serverId:%d", err, req.Account, req.ServerID))
				sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
				return
			}
		}
	}
	err = checkLoginParam(&req)
	if err != nil {
		logger.InfoWithSprintf("[web] invalid param: %v,account:%s,serverId:%d", err, req.Account, req.ServerID)
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	banInfo := checkBanList(&req)
	if banInfo != nil {
		resp := &webProto.LoginResponse{BanInfo: &webProto.BanInfo{
			Reason:  banInfo.Reason,
			EndTime: banInfo.EndTime,
		}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		logger.InfoWithSprintf("[web] login ban: %v,account:%s,serverId:%d", banInfo, req.Account, req.ServerID)
		return
	}

	err = checkServerStatus(&req)
	if err != nil {
		sendErrorMessage(pb.ERROR_CODE_LOGIN_SERVER_IS_MAINTAIN, w)
		logger.InfoWithSprintf("[web] login server status error: %v,account:%s,serverId:%d", err, req.Account, req.ServerID)
		return
	}

	token := generatorToken(req.Account, req.ServerID)
	_, err = dbService.RDB.Set(context.Background(), enum.GetLoginTokenKey(req.Account, token, req.ServerID), true, 30*time.Second).Result()
	if err != nil {
		sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
		logger.ErrorBySprintf("[web] login token generate error: %v,account:%s,serverId:%d", err, req.Account, req.ServerID)
		return
	}

	isAudit := httpPlatform.GetServerInfoService().IsAuditVersion(req.Version)

	var gateWayAddr string
	if isAudit {
		gateWayAddr, err = httpPlatform.GetAuditServerAddr()
		if err != nil {
			sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
			logger.ErrorBySprintf("[web] login get gateway address error: %v,account:%s,serverId:%d", err, req.Account, req.ServerID)
			return
		}
	} else {
		gateWayAddr, err = ServerNodeService.GetGateWayOpenAddress()
		if err != nil {
			sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
			logger.ErrorBySprintf("[web] login get gateway address error: %v,account:%s,serverId:%d", err, req.Account, req.ServerID)
			return
		}
	}

	_ = easyDB.DirectSavePlayerEntityByWhere[model.AccountSimpleInfoEntity](accountSimpleEntity)

	clientCfgInfo, err := easyDB.GetServerEntityByWhere[model.GameClientVersionEntity](map[string]interface{}{"version": req.Version})
	if err != nil {
		sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
		logger.ErrorBySprintf("[web] login get client config error: %v,account:%s,serverId:%d", err, req.Account, req.ServerID)
		return
	}
	resp := &webProto.LoginResponse{WsAddr: gateWayAddr, ServerId: req.ServerID, SessionToken: token, CfgUrl: clientCfgInfo.HotfixConfig, IsAudit: isAudit}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	logger.InfoWithSprintf("[web] login success: %v,account:%s,serverId:%d", resp, req.Account, req.ServerID)
}

func getBlockAnnounce(serverId int32) *model.AnnounceInfoEntity {
	logger.InfoWithSprintf("[web] login get block announce serverId:%d", serverId)
	return httpPlatform.GetServerInfoService().GetBlockAnnounce(serverId)
}

func generatorToken(account string, serverId int32) string {
	return tool.Md5(fmt.Sprintf("%s_%d_%d", account, serverId, tool.UnixNowMilli()))
}

func checkBanList(req *webProto.LoginReq) *model.BanListEntity {
	logger.InfoWithSprintf("[web] login check ban account:%s,serverId:%d", req.Account, req.ServerID)
	return httpPlatform.GetServerInfoService().CheckBanList(req.Account, req.ServerID)
}

// 检测服务器状态
func checkServerStatus(req *webProto.LoginReq) error {
	logger.InfoWithSprintf("[web] login check server status account:%s,serverId:%d", req.Account, req.ServerID)
	serverInfo := httpPlatform.GetServerInfoService().GetServerInfo(req.ServerID)
	if serverInfo == nil {
		return errors.New("invalid server id")
	}
	if serverInfo.ServerOpenTime > tool.UnixNowMilli() {
		return errors.New("server not open")
	}
	if serverInfo.Status != 0 {
		if !checkWhiteList(req) {
			return errors.New("server closed")
		}
	}
	return nil
}

func checkWhiteList(req *webProto.LoginReq) bool {
	return httpPlatform.GetServerInfoService().CheckWhiteList(req.Account)
}

// 检测登录参数
func checkLoginParam(req *webProto.LoginReq) error {
	logger.InfoWithSprintf("[web] login check param: %v,account:%s,serverId:%d", req, req.Account, req.ServerID)
	if req.Account == "" {
		return errors.New("missing code")
	}
	if httpPlatform.GetServerInfoService().GetServerInfo(req.ServerID) == nil {
		return errors.New("invalid server id")
	}
	if req.Channel != nodeConfig.ChannelId {
		return errors.New("missing channel")
	}
	if !httpPlatform.GetServerInfoService().CheckClientVersion(req.Version) {
		return errors.New("invalid client version")
	}
	return nil
}

func handleAnnounceInfoRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	var req webProto.GetAllAnnounceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	err := checkAnnounceParam(&req)
	if err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	announceInfos := httpPlatform.GetServerInfoService().GetAllAnnounceInfo(req.ServerID)

	infos := make([]*webProto.AnnounceInfo, 0, len(announceInfos))
	for _, v := range announceInfos {
		infos = append(infos, &webProto.AnnounceInfo{
			ID:         v.Id,
			Title:      v.Title,
			Content:    v.Content,
			Type:       v.AnnounceType,
			ShowType:   v.ShowType,
			PicAddress: v.PicAddress,
			StartTime:  v.StartTime,
		})
	}

	resp := &webProto.GetAllAnnounceResp{Data: infos}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func checkAnnounceParam(req *webProto.GetAllAnnounceReq) interface{} {
	if req.Account == "" {
		return errors.New("missing code")
	}
	if httpPlatform.GetServerInfoService().GetServerInfo(req.ServerID) == nil {
		return errors.New("invalid server id")
	}
	if req.Channel != nodeConfig.ChannelId {
		return errors.New("missing channel")
	}
	if !httpPlatform.GetServerInfoService().CheckClientVersion(req.Version) {
		return errors.New("invalid client version")
	}
	return nil
}

func handleGetChatMessageRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	var req webProto.GetChatMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	res := make([]*pb.PushReceivedChatMessage, 0)
	ctx := r.Context()

	if req.ChatType == int32(enum.BROADCAST_TYPE_SERVER_ID) || req.ChatType == int32(enum.BROADCAST_TYPE_ALLIANCE) {
		messages, err := dbService.RDB.LRange(ctx, enum.GetChatKey(req.ChatType, req.ToObjectId), 0, -1).Result()
		if err == nil {

			for _, msg := range messages {
				var msgPb *pb.PushReceivedChatMessage
				if err := json.Unmarshal([]byte(msg), &msgPb); err != nil {
					continue
				}
				sendUserId := msgPb.ChatMessage.PlayerInfo.UserId

				userBasicRedisInfo := logicCommon.GetPlayerBasicInfoFromRedis(sendUserId)
				if userBasicRedisInfo == nil {
					logger.ErrorBySprintf("[redis] basic info missing userId=%d", sendUserId)
					continue
				}
				msgPb.ChatMessage.PlayerInfo.ImageId = userBasicRedisInfo.ImageId
				msgPb.ChatMessage.PlayerInfo.TitleId = userBasicRedisInfo.Title
				msgPb.ChatMessage.PlayerInfo.BubbleId = userBasicRedisInfo.BubbleId
				msgPb.ChatMessage.PlayerInfo.HeadFrameId = userBasicRedisInfo.FrameId
				msgPb.ChatMessage.PlayerInfo.HeadId = userBasicRedisInfo.HeadId
				res = append(res, msgPb)
			}
		}
	}
	resp := &webProto.GetChatMessageResp{MsgList: res}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
