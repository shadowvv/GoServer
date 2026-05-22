package logicScene

import (
	"errors"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

type SceneSelector struct {
	sm       *SceneManager
	strategy SelectStrategy
}

func NewSceneSelector(sm *SceneManager, strat SelectStrategy) *SceneSelector {
	sceneSelector := &SceneSelector{
		sm:       sm,
		strategy: strat,
	}
	return sceneSelector
}

func (ss *SceneSelector) Assign(sceneTemplateId int32, userID int64, sceneMaxPlayerNum int32) (*Scene, error) {
	existing := ss.sm.GetSceneListBySceneId(sceneTemplateId)

	if s := ss.strategy.Select(existing, userID, sceneMaxPlayerNum); s != nil {
		return s, nil
	}
	logger.ErrorBySprintf("[sceneSelector] no available scene templateId:%d userId:%d", sceneTemplateId, userID)
	return nil, errors.New("no available scene")
}

type SelectStrategy interface {
	Select(existing []*Scene, userID int64, sceneMaxPlayerNum int32) *Scene
}

type MinLoadStrategy struct{}

func (m *MinLoadStrategy) Select(existing []*Scene, userID int64, sceneMaxPlayerNum int32) *Scene {
	var best *Scene
	num := int32(0)
	for _, s := range existing {
		if s.playerNum.Load() >= sceneMaxPlayerNum {
			continue
		}
		if best == nil || s.playerNum.Load() < best.playerNum.Load() {
			best = s
		}
		num++
	}
	if best == nil || num >= sceneMaxPlayerNum {
		randIndex := tool.RandInt(0, len(existing)-1)
		return existing[randIndex]
	}
	return best
}
