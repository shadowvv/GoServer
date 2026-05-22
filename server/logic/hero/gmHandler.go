// File: gmHandler.go
// Description: 英雄系统GM命令处理器
// Author: 木村凉太
// Create Time: 2026.02

package hero

import (
	"fmt"
	"github.com/drop/GoServer/server/logic/model"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"

	"github.com/drop/GoServer/server/logic/pb"
)

var _ logicCommon.GMCommandHandler = (*HeroGMHandler)(nil)

// HeroGMHandler 英雄系统GM命令处理器
type HeroGMHandler struct {
	sessionManager logicCommon.SessionManagerInterface
}

// NewHeroGMHandler 创建英雄系统GM命令处理器
func NewHeroGMHandler(manager logicCommon.SessionManagerInterface) *HeroGMHandler {
	return &HeroGMHandler{
		sessionManager: manager,
	}
}

// GetSupportedCommands 返回支持的命令类型
func (h *HeroGMHandler) GetSupportedCommands() []pb.GMCommandType {
	return []pb.GMCommandType{
		pb.GMCommandType_GM_CMD_HERO_LEVEL_UP,
		pb.GMCommandType_GM_CMD_HERO_STAR_UP,
		pb.GMCommandType_GM_CMD_ADD_HERO,
	}
}

// HandleCommand 处理GM命令
func (h *HeroGMHandler) HandleCommand(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	cmdType := req.GetCmdType()

	switch cmdType {
	case pb.GMCommandType_GM_CMD_HERO_LEVEL_UP:
		return h.handleHeroLevelUp(req, userId)
	case pb.GMCommandType_GM_CMD_HERO_STAR_UP:
		return h.handleHeroStarUp(req, userId)
	case pb.GMCommandType_GM_CMD_ADD_HERO:
		return h.handleAddHero(req, userId)
	default:
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_CMD,
			Message: fmt.Sprintf("英雄系统不支持的命令类型: %d", cmdType),
		}
	}
}

// handleHeroLevelUp 处理英雄升级命令
func (h *HeroGMHandler) handleHeroLevelUp(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	heroOwnId := req.GetHeroOwnId()
	heroLevel := req.GetHeroLevel()

	if heroOwnId <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "英雄唯一ID必须大于0",
		}
	}

	if heroLevel <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "英雄等级必须大于0",
		}
	}

	// 获取玩家模型
	p := h.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家不存在",
		}
	}
	player := p.(*model.PlayerModel)

	// 确保英雄模型已加载
	if player.HeroDetailsModel == nil || player.HeroDetailsModel.Entities == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: "英雄数据未加载",
		}
	}

	// 查找英雄
	heroDetail := player.HeroDetailsModel.GetHero(heroOwnId)
	if heroDetail == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: fmt.Sprintf("英雄不存在: HeroOwnId=%d", heroOwnId),
		}
	}

	// 更新英雄等级
	player.HeroDetailsModel.UpdateLevel(heroOwnId, heroLevel)

	// 保存到数据库
	player.HeroDetailsModel.SaveModelToDB()

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: fmt.Sprintf("英雄升级成功: HeroOwnId=%d, Level=%d", heroOwnId, heroLevel),
		ExtraData: map[string]string{
			"hero_own_id": fmt.Sprintf("%d", heroOwnId),
			"hero_level":  fmt.Sprintf("%d", heroLevel),
			"user_id":     fmt.Sprintf("%d", userId),
		},
	}
}

// handleHeroStarUp 处理英雄升星命令
func (h *HeroGMHandler) handleHeroStarUp(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	heroOwnId := req.GetHeroOwnId()
	heroStar := req.GetHeroStar()

	if heroOwnId <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "英雄唯一ID必须大于0",
		}
	}

	if heroStar <= 0 {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "英雄星级必须大于0",
		}
	}

	// 获取玩家模型
	p := h.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家不存在",
		}
	}
	player := p.(*model.PlayerModel)

	// 确保英雄模型已加载
	if player.HeroDetailsModel == nil || player.HeroDetailsModel.Entities == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: "英雄数据未加载",
		}
	}

	// 查找英雄
	heroDetail := player.HeroDetailsModel.GetHero(heroOwnId)
	if heroDetail == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: fmt.Sprintf("英雄不存在: HeroOwnId=%d", heroOwnId),
		}
	}

	// 更新英雄星级
	player.HeroDetailsModel.UpdateStarLevel(heroOwnId, heroStar)

	// 更新图鉴历史最高星级（如果新星级更高）
	if player.HeroAlbumModel != nil && player.HeroAlbumModel.Entities != nil {
		album := player.HeroAlbumModel.GetAlbum(heroDetail.HeroID)
		if album != nil && heroStar > album.HistoryMaxStar {
			player.HeroAlbumModel.UpdateHistoryMaxStar(heroDetail.HeroID, heroStar)
		}
	}

	// 保存到数据库
	player.HeroDetailsModel.SaveModelToDB()
	if player.HeroAlbumModel != nil {
		player.HeroAlbumModel.SaveModelToDB()
	}

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: fmt.Sprintf("英雄升星成功: HeroOwnId=%d, Star=%d", heroOwnId, heroStar),
		ExtraData: map[string]string{
			"hero_own_id": fmt.Sprintf("%d", heroOwnId),
			"hero_star":   fmt.Sprintf("%d", heroStar),
			"user_id":     fmt.Sprintf("%d", userId),
		},
	}
}

func (h *HeroGMHandler) handleAddHero(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	heroId := req.GetHeroId()
	cfg := gameConfig.GetHeroBaseCfg(int32(heroId))
	if cfg == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: fmt.Sprintf("英雄配置不存在: HeroId=%d", heroId),
		}
	}

	p := h.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "玩家不存在",
		}
	}
	player := p.(*model.PlayerModel)

	detail, err := AddHeroDetail(player, heroId)
	if err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("添加英雄失败: HeroId=%d, Error=%s", heroId, err.Error()),
		}
	}
	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: fmt.Sprintf("添加英雄成功: HeroId=%d, HeroOwnId=%d", heroId, detail.HeroOwnID),
		ExtraData: map[string]string{
			"hero_id":     fmt.Sprintf("%d", heroId),
			"hero_own_id": fmt.Sprintf("%d", detail.HeroOwnID),
			"user_id":     fmt.Sprintf("%d", userId),
		},
	}
}
