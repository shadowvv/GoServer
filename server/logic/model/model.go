package model

import (
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/eventService"
)

var unlockService logicCommon.UnlockServiceInterface
var messageSender logicCommon.MessageSenderInterface
var itemService logicCommon.ItemService
var activityService logicCommon.GameActivityServiceInterface
var rpcMessageSender logicCommon.RpcMessageSenderInterface
var eventServer *eventService.EventBus
var mailServer logicCommon.MailServiceInterface
var playerHeartbeatService logicCommon.PlayerHeartbeatServiceInterface

func InitModel(unlock logicCommon.UnlockServiceInterface, mailService logicCommon.MailServiceInterface, eventBus *eventService.EventBus, sender logicCommon.MessageSenderInterface, item logicCommon.ItemService, activity logicCommon.GameActivityServiceInterface, rpcSender logicCommon.RpcMessageSenderInterface) {
	unlockService = unlock
	messageSender = sender
	eventServer = eventBus
	itemService = item
	mailServer = mailService
	activityService = activity
	rpcMessageSender = rpcSender
}

func InitPlayerHeartbeatService(service logicCommon.PlayerHeartbeatServiceInterface) {
	playerHeartbeatService = service
}
