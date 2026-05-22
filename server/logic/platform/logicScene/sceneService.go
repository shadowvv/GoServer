package logicScene

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

var sessionMgr *logicSessionManager.GameSessionManager
var playerProcessorConfig *nodeConfig.MessageProcessConfig

type SceneService struct {
	sm       *SceneManager
	selector *SceneSelector

	sceneMaxPlayerNum int32
}

func NewSceneService(gameSessionManager *logicSessionManager.GameSessionManager, dispatcher *dispatcherService.Dispatcher, sceneProcessConfig *nodeConfig.MessageProcessConfig, sceneMaxPlayerNum int32, playerProcessConfig *nodeConfig.MessageProcessConfig) *SceneService {
	sessionMgr = gameSessionManager
	playerProcessorConfig = playerProcessConfig
	manager := NewSceneManager()
	service := &SceneService{
		sm:                manager,
		selector:          NewSceneSelector(manager, &MinLoadStrategy{}),
		sceneMaxPlayerNum: sceneMaxPlayerNum,
	}
	manager.InitScene(sceneProcessConfig, dispatcher)
	return service
}

func (s *SceneService) EnterScene(scenePlayer *ScenePlayer, instanceId int32) error {
	scene, err := s.selector.Assign(instanceId, scenePlayer.UserID, s.sceneMaxPlayerNum)
	if err != nil {
		return err
	}
	scene.EnterScene(scenePlayer)
	s.sm.UpdatePlayerSceneId(scenePlayer.UserID, scene.SceneID)
	return nil
}
func (s *SceneService) PlayerSceneLoadOver(playerId int64) error {
	scene, ok := s.sm.GetPlayerScene(playerId)
	if !ok {
		return errors.New("no scene")
	}
	scene.PlayerLoadOver(playerId)
	return nil
}

func (s *SceneService) SceneLoadOver(id int64) {

}

func (s *SceneService) LeaveScene(userID int64) (*ScenePlayer, error) {
	scene, ok := s.sm.GetPlayerScene(userID)
	if !ok {
		return nil, errors.New("scene not found")
	}
	s.sm.RemovePlayerSceneId(userID)
	return scene.RemovePlayer(userID), nil
}

func (s *SceneService) PutPlayerMessage(playerId int64, msgId int32, msg proto.Message, handler logicCommon.PlayerMessageHandler, function enum.FunctionIdEnum, isSceneTransferMessage bool) {
	sceneId, ok := s.sm.GetPlayerSceneId(playerId)
	if !ok {
		logger.ErrorWithZapFields(fmt.Sprintf("[scene] PushPlayerMessage playerId:%d not found", playerId))
		return
	}
	scene, ok := s.sm.GetScene(sceneId)
	if !ok {
		logger.ErrorWithZapFields(fmt.Sprintf("[scene] PushPlayerMessage sceneId:%d not found", sceneId))
		return
	}
	scene.PushPlayerMessage(playerId, msgId, msg, handler, function, isSceneTransferMessage)
}

func (s *SceneService) PutSceneInnerMessageTask(task serviceInterface.InnerTaskInterface) {
	scene, ok := s.sm.GetScene(int32(task.GetReqId()))
	if !ok {
		logger.ErrorWithZapFields(fmt.Sprintf("[scene] PushInnerTask sceneId:%d not found", task.GetReqId()))
		return
	}
	scene.PutSceneInnerTask(task)
}

func (s *SceneService) PutPlayerInnerMessageTask(task serviceInterface.InnerTaskInterface) {
	scene, ok := s.sm.GetPlayerScene(task.GetReqId())
	if !ok {
		logger.ErrorWithZapFields(fmt.Sprintf("[scene] PushInnerTask sceneId:%d not found", task.GetReqId()))
		return
	}
	scene.PutSceneInnerTask(task)
}

func (s *SceneService) PutSceneInnerMessageResp(task serviceInterface.InnerTaskInterface, respHandle func()) {
	scene, ok := s.sm.GetScene(int32(task.GetReqId()))
	if !ok {
		logger.ErrorWithZapFields(fmt.Sprintf("[scene] PushInnerTaskResp sceneId:%d not found", task.GetReqId()))
		return
	}
	scene.PutSceneInnerResp(respHandle)
}

func (s *SceneService) PutPlayerInnerMessageResp(task serviceInterface.InnerTaskInterface, respHandle func()) {
	scene, ok := s.sm.GetPlayerScene(task.GetReqId())
	if !ok {
		logger.ErrorWithZapFields(fmt.Sprintf("[scene] PushInnerTaskResp sceneId:%d not found", task.GetReqId()))
		return
	}
	scene.PutSceneInnerResp(respHandle)
}
