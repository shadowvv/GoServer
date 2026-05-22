package task

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/eventService"
)

type TaskEventHandler struct {
	eventBus *eventService.EventBus
}

func NewTaskEventHandler(eventBus *eventService.EventBus) *TaskEventHandler {
	eh := &TaskEventHandler{
		eventBus: eventBus,
	}
	// 绑定 TaskQueue 的回调到当前 handler 的方法
	return eh
}

func (eh *TaskEventHandler) Start() {
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

func (eh *TaskEventHandler) handleEvent(event eventService.GameEvent) {

	if event.GetObjectID() == 0 {
		return
	}
	eventType := event.GetEventType()
	if _, ok := enum.EventToObjectiveTypes[eventType]; ok {
		dispatcher.DispatchInnerMessageTask(enum.INNER_MSG_TYPE_PLAYER, enum.INNER_MSG_EVENT_TASK_PLAYER, event.GetObjectID(), event, 0, 0, nil)
	}
}

var messageSender logicCommon.MessageSenderInterface
var dispatcher *dispatcherService.Dispatcher

func TaskInitServer(eventBus *eventService.EventBus, sender logicCommon.MessageSenderInterface, dispatcherInstance *dispatcherService.Dispatcher) {
	messageSender = sender
	dispatcher = dispatcherInstance
	handler := NewTaskEventHandler(eventBus)
	handler.Start()
}
