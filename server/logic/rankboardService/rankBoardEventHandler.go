package rankboardService

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/eventService"
)

type RankBoardEventHandler struct {
	eventBus        *eventService.EventBus
	activityService logicCommon.GameActivityServiceInterface
	dispatcher      *dispatcherService.Dispatcher
}

func NewRankBoardEventHandler(eventBus *eventService.EventBus, activityService logicCommon.GameActivityServiceInterface, dispatcher *dispatcherService.Dispatcher) *RankBoardEventHandler {
	eh := &RankBoardEventHandler{
		eventBus:        eventBus,
		activityService: activityService,
		dispatcher:      dispatcher,
	}
	// 绑定 TaskQueue 的回调到当前 handler 的方法
	return eh
}

func (eh *RankBoardEventHandler) Start() {
	// 订阅所有事件
	eventCh := eh.eventBus.SubscribeAll()
	go func() {
		for {
			select {
			case event, ok := <-eventCh:
				if !ok {
					return
				}
				eh.handleEvent(event)
			}
		}
	}()
	//log.Println("任务事件处理器启动")
}

func (eh *RankBoardEventHandler) handleEvent(event eventService.GameEvent) {

	if event.GetObjectID() == 0 {
		return
	}
	eventType := event.GetEventType()
	if _, ok := enum.EventToObjectiveTypes[eventType]; ok {
		switch eventType {
		case enum.EventTypePassInstance:
			eh.dispatcher.DispatchInnerMessageTask(enum.INNER_MSG_TYPE_PLAYER, enum.INNER_MSG_EVENT_UPDATE_RANKBOARD, event.GetObjectID(), event, 0, 0, nil)
		case enum.EventTypeBuildLevelUp:
			eh.dispatcher.DispatchInnerMessageTask(enum.INNER_MSG_TYPE_PLAYER, enum.INNER_MSG_EVENT_UPDATE_RANKBOARD, event.GetObjectID(), event, 0, 0, nil)
		}
	}
}

func InitEventHandler(eventBus *eventService.EventBus, activityService logicCommon.GameActivityServiceInterface, dispatcher *dispatcherService.Dispatcher) {
	handler := NewRankBoardEventHandler(eventBus, activityService, dispatcher)
	handler.Start()
}
