package enum

import (
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

type ObjType int32

const (
	ObjType_Unknown ObjType = iota
	ObjType_Register
	ObjType_Login
	ObjType_Recharge
)

var channels = "2"

// 数据上报
type Obj struct {
	UserId   int64   `json:"uid"`
	Account  string  `json:"account"`
	Time     int64   `json:"time"`
	ObjType  ObjType `json:"objType"`
	ObjValue int64   `json:"objValue"`
}

func NewRedisMessage(userId int64, account string, objType ObjType, objValue int64) *Obj {
	return &Obj{
		UserId:   userId,
		Account:  account,
		Time:     time.Now().Unix(),
		ObjType:  objType,
		ObjValue: objValue,
	}
}

func PublishRegister(RDB *redis.Client, userId int64, account string, objValue int64) {
	RDB.Publish(context.Background(), channels, NewRedisMessage(userId, account, ObjType_Register, objValue))
}

func PublishLogin(RDB *redis.Client, userId int64, account string, objValue int64) {
	RDB.Publish(context.Background(), channels, NewRedisMessage(userId, account, ObjType_Login, objValue))
}

func PublishRecharge(RDB *redis.Client, userId int64, account string, objValue int64) {
	RDB.Publish(context.Background(), channels, NewRedisMessage(userId, account, ObjType_Recharge, objValue))
}
