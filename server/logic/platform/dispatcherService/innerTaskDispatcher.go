package dispatcherService

import (
	"fmt"
	"github.com/drop/GoServer/server/service/serviceInterface"

	"sync"

	"github.com/drop/GoServer/server/enum"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
)

type InnerTask struct {
	Id            int64
	MessageId     enum.InnerMessageId
	ReqType       enum.InnerMessageType
	ReqId         int64
	ReqParameter  any
	ReqCallHandle logicCommon.InnerMessageHandler
	RespType      enum.InnerMessageType
	RespId        int64
	RespCallback  serviceInterface.InnerTaskResult
	result        any
	err           error

	done chan struct{}
	once sync.Once
}

var _ serviceInterface.InnerTaskInterface = (*InnerTask)(nil)

func (i *InnerTask) ReqCall(task serviceInterface.InnerTaskInterface) (any, error) {
	return i.ReqCallHandle(task)
}

func (i *InnerTask) Resolve(res any, err error) {
	i.once.Do(func() {
		i.result = res
		i.err = err
		close(i.done)
	})
}

func (i *InnerTask) GetReqId() int64 {
	return i.ReqId
}

func (i *InnerTask) GetRespId() int64 {
	return i.RespId
}

func (i *InnerTask) SetError(err error) {
	i.err = err
}

type InnerTaskFutureManager struct {
	mu       sync.Mutex
	taskMap  map[int64]*InnerTask
	dispatch serviceInterface.DispatchInterface
}

func NewInnerMessageFutureManager(dispatch serviceInterface.DispatchInterface) *InnerTaskFutureManager {
	return &InnerTaskFutureManager{
		taskMap:  make(map[int64]*InnerTask),
		dispatch: dispatch,
	}
}

func (m *InnerTaskFutureManager) AddTask(task *InnerTask) {
	m.mu.Lock()
	m.taskMap[task.Id] = task
	m.mu.Unlock()

	go func() {
		<-task.done
		// schedule response handling on target processor (preserving resp hash)
		m.dispatch.DispatchInnerTaskResp(task, func() {
			// protect RespCallback being nil or panic inside callback
			defer func() {
				if r := recover(); r != nil {
					// log the panic
					platformLogger.ErrorWithUser(fmt.Sprintf("panic in RespCallback: %v", r), nil, nil)
				}
			}()

			// It's better if RespCallback accepts (result, err), but to keep API compatible:
			task.RespCallback(task.result, task.err)
			// Remove task from manager (idempotent)
			m.RemoveTask(task.Id)
		})
	}()
}

func (m *InnerTaskFutureManager) RemoveTask(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.taskMap, id)
}
