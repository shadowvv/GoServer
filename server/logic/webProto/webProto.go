package webProto

import "github.com/drop/GoServer/server/logic/pb"

type LoginReq struct {
	Account  string `json:"account"`  // 登录account
	Channel  int32  `json:"channel"`  // 登录渠道
	Version  string `json:"version"`  // 登录版本
	ServerID int32  `json:"server"`   // 登录服务器(0 表示最新服务器)
	Language uint16 `json:"lang"`     // 语言ID(0 简体中文  1 英文  2 日文  3 韩文  4 繁体中文)
	DeviceID string `json:"deviceId"` // 设备ID
	AppID    string `json:"appId"`    // 登录的AppID
	Sign     string `json:"sign"`     // 签名字段
}

type LoginResponse struct {
	WsAddr       string        `json:"wsAddr"`
	SessionToken string        `json:"sessionToken"`
	ServerId     int32         `json:"serverId"`
	Announce     *AnnounceInfo `json:"announce"`
	BanInfo      *BanInfo      `json:"banInfo"`
	CfgUrl       string        `json:"cfgUrl"`
	IsAudit      bool          `json:"isAudit"`
}

type BanInfo struct {
	EndTime int64 `json:"endTime"` // 封禁时间
	Reason  int32 `json:"reason"`  // 封禁原因
}

type AnnounceInfo struct {
	ID         int32  `json:"id"`
	Type       int32  `json:"type"`
	ShowType   int32  `json:"showType"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	PicAddress string `json:"picAddress"`
	StartTime  int64  `json:"startTime"`
}

type GetAllAnnounceReq struct {
	Account  string `json:"account"` // 登录account
	ServerID int32  `json:"server"`
	Channel  int32  `json:"channel"` // 登录渠道
	Version  string `json:"version"` // 登录版本
	Language uint16 `json:"lang"`    // 语言ID(0 简体中文  1 英文  2 日文  3 韩文  4 繁体中文)
}

type GetAllAnnounceResp struct {
	Data []*AnnounceInfo `json:"data"` // 返回数据
}

type ServerListReq struct {
	Account string `json:"account"` // 登录code
}

type ServerListResp struct {
	Code int            `json:"code"` // 错误类型
	Data *AllServerInfo `json:"data"` // 返回数据
}

type AllServerInfo struct {
	List    []*ServerInfo     `json:"serverInfo"`
	RegList []*UserSimpleInfo `json:"userInfo"`
}

type ServerInfo struct {
	ID       int32 `json:"id"`     // 服务器ID
	Name     int32 `json:"name"`   // 服务器名称
	AreaID   int32 `json:"areaId"` // 大区ID
	Status   int32 `json:"status"` // 服务器状态
	OpenTime int64 `json:"time"`   // 开启时间
}

type UserSimpleInfo struct {
	UserId        int64  `json:"userId"`
	Nickname      string `json:"nickname"`
	Level         int32  `json:"level"`
	Head          int32  `json:"head"`
	Frame         int32  `json:"frame"`
	ServerID      int32  `json:"id"`
	LastLoginTime int64  `json:"lastLoginTime"`
	TitleId       int32  `json:"titleId"`
	Exp           int64  `json:"exp"`
}

type GetChatMessageReq struct {
	ChatType   int32 `json:"chatType"`   // 聊天类型 0 私聊 1 全服通知 2 世界 3 工会
	ToObjectId int64 `json:"toObjectId"` // 世界/工会/私聊对象ID
}

type GetChatMessageResp struct {
	MsgList []*pb.PushReceivedChatMessage `json:"msgList"`
}

type WebErrorMessage struct {
	Code int32 `json:"code"`
}

type ConsumeProductReq struct {
	PlayerId  int64        `json:"playerId"`
	OrderInfo []*OrderInfo `json:"orderInfo"` // 订单ID
}

type OrderInfo struct {
	ProductId string `json:"productId"` // 商品ID
	Token     string `json:"token"`     // 订单状态
	OrderId   string `json:"orderId"`   // 订单ID
}

type ConsumeProductResp struct {
	Code int32 `json:"code"` // 错误类型
}
