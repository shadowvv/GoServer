package logicCommon

import (
	"github.com/drop/GoServer/server/service/serviceInterface"
)

type UserBaseInterface interface {
	GetUserId() int64
	GetUserAccount() string
	GetUserServerId() int32
	GetSession() serviceInterface.SessionInterface
	GetNodeId() int32
	GetSceneId() int32
}

type PlayerInterface interface {
	UserBaseInterface
	Heartbeat(milli int64) error
	SavePlayerToDB()
	GetLevel() int32
}

type PlayerModelInterface interface {
	SaveModelToDB()
	Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool)
}

type PlayerHeartbeatServiceInterface interface {
	Heartbeat(player PlayerInterface, currentTime int64)
}

type HeroAttrInterface interface {
	GetHeroAttr(heroId int64, attrId int32) int64
	GetBuffAttr(heroId int64, attrId int32) int64
	// GetChangedHeroOwnIDs 返回本次有变化的英雄OwnID列表和全局脏标记
	// - heroOwnIDs: 具体变化的英雄ID列表（为空表示无特定英雄变化）
	// - allDirty: true 表示影响全部英雄（如建筑、石像、饰品全局属性等）
	GetChangedHeroOwnIDs() (heroOwnIDs []int64, allDirty bool)
}

type GatewayPlayerInfo struct {
	Account  string
	ServerId int32
	UserId   int64
	NodeId   int32
	Level    int32
	Session  serviceInterface.SessionInterface
}

var _ UserBaseInterface = (*GatewayPlayerInfo)(nil)

func (g *GatewayPlayerInfo) GetUserId() int64 {
	return g.UserId
}

func (g *GatewayPlayerInfo) GetUserAccount() string {
	return g.Account
}

func (g *GatewayPlayerInfo) GetUserServerId() int32 {
	return g.ServerId
}

func (g *GatewayPlayerInfo) GetSession() serviceInterface.SessionInterface {
	return g.Session
}

func (g *GatewayPlayerInfo) GetNodeId() int32 {
	return g.NodeId
}

// 网关不关心玩家所在具体场景
func (g *GatewayPlayerInfo) GetSceneId() int32 {
	return 0
}
