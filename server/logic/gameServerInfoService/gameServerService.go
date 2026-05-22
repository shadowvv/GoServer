package gameServerInfoService

import (
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

type GameServerInfoService struct {
	serverInfoModel   *model.GameServerInfoModel
	announceInfoModel *model.AnnounceInfoModel
}

func NewGameServerInfoService() *GameServerInfoService {
	gameServerInfoEntities, err := easyDB.GetServerAllEntities[model.GameServerInfoEntity]()
	if err != nil {
		panic(err)
	}
	announceInfoEntities, err := easyDB.GetServerEntitiesByCustomCondition[model.AnnounceInfoEntity]("end_time > ?", tool.UnixNowMilli())
	if err != nil {
		panic(err)
	}

	service := &GameServerInfoService{
		serverInfoModel:   model.NewGameServerInfoModel(gameServerInfoEntities),
		announceInfoModel: model.NewAnnounceInfoModel(announceInfoEntities),
	}
	return service
}

func (s *GameServerInfoService) GetServerInfo(serverId int32) *model.GameServerInfoEntity {
	return s.serverInfoModel.GetGameServerInfo(serverId)
}

func (s *GameServerInfoService) GetAllServerInfo() map[int32]*model.GameServerInfoEntity {
	return s.serverInfoModel.GetAllServerInfo()
}

func (s *GameServerInfoService) GetAllOpenServerInfo() map[int32]*model.GameServerInfoEntity {
	return s.serverInfoModel.GetAllOpenServerInfo()
}

func (s *GameServerInfoService) CheckBanList(account string, serverId int32) *model.BanListEntity {
	banEntity, err := easyDB.GetServerEntityByWhere[model.BanListEntity](map[string]interface{}{"account": account, "server_id": serverId})
	if err != nil {
		return nil
	}
	return banEntity
}

func (s *GameServerInfoService) CheckClientVersion(version string) bool {
	_, err := easyDB.GetServerEntityByWhere[model.GameClientVersionEntity](map[string]interface{}{"version": version})
	if err != nil {
		return false
	}
	return true
}

func (s *GameServerInfoService) CheckWhiteList(account string) bool {
	_, err := easyDB.GetServerEntityByWhere[model.WhiteListEntity](map[string]interface{}{"account": account})
	if err != nil {
		return false
	}
	return true
}

func (s *GameServerInfoService) GetBlockAnnounce(serverId int32) *model.AnnounceInfoEntity {
	return s.announceInfoModel.GetBlockAnnounceInfo(serverId)
}

func (s *GameServerInfoService) GetAllAnnounceInfo(serverId int32) []*model.AnnounceInfoEntity {
	return s.announceInfoModel.GetAllAnnounceInfo(serverId)
}

func (s *GameServerInfoService) GetDefaultServerId() int32 {
	return s.serverInfoModel.GetDefaultServerId()
}

func (s *GameServerInfoService) GetNewServerWeight() int32 {
	return s.serverInfoModel.MaxNewWeight
}

func (s *GameServerInfoService) AddAnnounceInfo(entity *model.AnnounceInfoEntity) error {
	return s.announceInfoModel.AddAnnounceInfo(entity)
}

func (s *GameServerInfoService) UpdateAnnounceInfo(entity *model.AnnounceInfoEntity) error {
	return s.announceInfoModel.UpdateAnnounceInfo(entity)
}

func (s *GameServerInfoService) AddServerInfo(entity *model.GameServerInfoEntity) error {
	return s.serverInfoModel.AddServerInfo(entity)
}

func (s *GameServerInfoService) UpdateServerInfo(entity *model.GameServerInfoEntity) error {
	return s.serverInfoModel.UpdateServerInfo(entity)
}

func (s *GameServerInfoService) AddClientVersion(entity *model.GameClientVersionEntity) error {
	err := easyDB.SaveSeverEntity(entity)
	if err != nil {
		return err
	}
	return nil
}

func (s *GameServerInfoService) UpdateClientVersion(entity *model.GameClientVersionEntity) error {
	err := easyDB.SaveSeverEntity(entity)
	if err != nil {
		return err
	}
	return nil
}

func (s *GameServerInfoService) ReloadAnnounce() {
	announceInfoEntities, err := easyDB.GetServerEntitiesByCustomCondition[model.AnnounceInfoEntity]("end_time > ?", tool.UnixNowMilli())
	if err != nil {
		logger.ErrorBySprintf("announceInfoModel reload error:%v", err)
		return
	}
	s.announceInfoModel.ReloadAnnounce(announceInfoEntities)
}

func (s *GameServerInfoService) ReloadServerInfo() {
	serverInfoEntities, err := easyDB.GetServerAllEntities[model.GameServerInfoEntity]()
	if err != nil {
		logger.ErrorBySprintf("serverInfoModel reload error:%v", err)
		return
	}
	s.serverInfoModel.ReloadServerInfo(serverInfoEntities)
}

func (s *GameServerInfoService) IsAuditVersion(version string) bool {
	versionEntity, err := easyDB.GetServerEntityByWhere[model.GameClientVersionEntity](map[string]interface{}{"version": version})
	if err != nil {
		return false
	}
	return versionEntity.Examine == 1
}

func (s *GameServerInfoService) ResetOnlinePlayerNum() {
	s.serverInfoModel.ResetOnlinePlayerNum()
}
