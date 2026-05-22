package logicScene

import (
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"sync"
)

type SceneManager struct {
	sceneMu sync.RWMutex
	scenes  map[int32]*Scene

	playerMu         sync.RWMutex
	playerId2SceneId map[int64]int32
}

func NewSceneManager() *SceneManager {
	sm := &SceneManager{
		scenes:           make(map[int32]*Scene),
		playerId2SceneId: make(map[int64]int32),
	}
	return sm
}

func (sm *SceneManager) AddScene(s *Scene) {
	sm.sceneMu.Lock()
	defer sm.sceneMu.Unlock()

	sm.scenes[s.SceneID] = s
}

func (sm *SceneManager) GetScene(id int32) (*Scene, bool) {
	sm.sceneMu.RLock()
	defer sm.sceneMu.RUnlock()
	s, ok := sm.scenes[id]
	return s, ok
}

func (sm *SceneManager) GetSceneListBySceneId(sceneId int32) []*Scene {
	sm.sceneMu.RLock()
	defer sm.sceneMu.RUnlock()

	existing := make([]*Scene, 0)
	for _, s := range sm.scenes {
		if s.TemplateID == sceneId {
			existing = append(existing, s)
		}
	}
	return existing
}

func (sm *SceneManager) RemoveScene(id int32) {
	sm.sceneMu.Lock()
	defer sm.sceneMu.Unlock()

	delete(sm.scenes, id)
}

func (sm *SceneManager) GetPlayerSceneId(playerId int64) (int32, bool) {
	sm.playerMu.RLock()
	defer sm.playerMu.RUnlock()

	sceneId, ok := sm.playerId2SceneId[playerId]
	return sceneId, ok
}

func (sm *SceneManager) UpdatePlayerSceneId(playerId int64, sceneId int32) {
	sm.playerMu.Lock()
	defer sm.playerMu.Unlock()

	sm.playerId2SceneId[playerId] = sceneId
}

func (sm *SceneManager) RemovePlayerSceneId(playerId int64) {
	sm.playerMu.Lock()
	defer sm.playerMu.Unlock()

	delete(sm.playerId2SceneId, playerId)
}

func (sm *SceneManager) GetPlayerScene(playerId int64) (*Scene, bool) {
	sceneId, ok := sm.GetPlayerSceneId(playerId)
	if !ok {
		return nil, false
	}
	return sm.GetScene(sceneId)
}

func (sm *SceneManager) InitScene(sceneProcessConfig *nodeConfig.MessageProcessConfig, dispatcher *dispatcherService.Dispatcher) {
	for _, instance := range gameConfig.GetAllInstance() {
		for i := int32(0); i < sceneProcessConfig.RoutineNum; i++ {
			scene := NewScene(instance.Id*100+i, instance.Id, sceneProcessConfig)
			dispatcher.RegisterSceneProcessor(scene.SceneID, scene)
			sm.AddScene(scene)
			scene.Start()
		}
	}
}
