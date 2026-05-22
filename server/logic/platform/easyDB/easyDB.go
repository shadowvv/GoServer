package easyDB

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/platform/dbPool"
	"gorm.io/gorm"
)

var dbPoolManager *dbPool.DBPoolManager
var playerBarrierMap sync.Map // map[int64]*playerBarrier
var ErrWaitBarrierTimeout = errors.New("wait player barrier timeout")

func SetGameDBPool(dbManager *dbPool.DBPoolManager) {
	dbPoolManager = dbManager
	playerBarrierMap = sync.Map{}
}

func CreatePlayerEntity[T any](t *T) error {
	if err := dbPoolManager.DB.Create(t).Error; err != nil {
		return err
	}
	return nil
}

func GetPlayerEntitiesByWhere[T any](where map[string]interface{}) ([]*T, error) {
	var t []*T
	if err := dbPoolManager.DB.Where(where).Find(&t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func GetPlayerEntitiesByRaw[T any](sql string, values ...interface{}) ([]*T, error) {
	var t []*T
	if err := dbPoolManager.DB.Raw(sql, values...).Scan(&t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func GetPlayerEntityByWhere[T any](where map[string]interface{}) (*T, error) {
	var t T
	if err := dbPoolManager.DB.Where(where).Take(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func GetPlayerEntityByStringID[T any](pkName string, id any) (*T, error) {
	var t T
	if err := dbPoolManager.DB.Where(fmt.Sprintf("%s = ?", pkName), id).Take(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func GetPlayerEntityByID[T any](id any) (*T, error) {
	var t T
	if err := dbPoolManager.DB.Take(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func DirectSavePlayerEntityByWhere[T any](t *T) error {
	return dbPoolManager.DB.Save(t).Error
}

func cloneChangedMap(changed map[string]interface{}) map[string]interface{} {
	if len(changed) == 0 {
		return nil
	}
	snapshot := make(map[string]interface{}, len(changed))
	for k, v := range changed {
		snapshot[k] = v
	}
	return snapshot
}

func UpdatePlayerEntity[T any](t *T, changed map[string]interface{}, playerId int64) {
	if len(changed) == 0 {
		return
	}
	copyT := *t
	changedSnapshot := cloneChangedMap(changed)
	if len(changedSnapshot) == 0 {
		return
	}
	incPlayerBarrier(playerId)
	dbPoolManager.AddPlayerDBTask(enum.DB_POOL_TYPE_PLAYER, playerId, func(db *gorm.DB) error {
		defer decPlayerBarrier(playerId)
		return db.Model(&copyT).Updates(changedSnapshot).Error
	})
}

func UpdatePlayerEntityByRaw(sql string) error {
	return dbPoolManager.DB.Exec(sql).Error
}

func UpdatePlayerBatchEntities[T any](t map[int64]*T, changed map[int64]map[string]interface{}, playerId int64) {
	if len(changed) == 0 {
		return
	}
	entitySnapshot := make(map[int64]T, len(changed))
	changedSnapshot := make(map[int64]map[string]interface{}, len(changed))
	for id, changeInfo := range changed {
		entity := t[id]
		if entity == nil || len(changeInfo) == 0 {
			continue
		}
		entitySnapshot[id] = *entity
		changedSnapshot[id] = cloneChangedMap(changeInfo)
	}
	if len(changedSnapshot) == 0 {
		return
	}

	incPlayerBarrier(playerId)
	dbPoolManager.AddPlayerDBTask(enum.DB_POOL_TYPE_PLAYER, playerId, func(db *gorm.DB) error {
		defer decPlayerBarrier(playerId)
		for id, changeInfo := range changedSnapshot {
			entity := entitySnapshot[id]
			copyT := entity
			if err := db.Model(&copyT).Updates(changeInfo).Error; err != nil {
				continue
			}
		}
		return nil
	})
}

func DeletePlayerEntityByWhere[T any](where map[string]interface{}, playerId int64) error {
	incPlayerBarrier(playerId)
	dbPoolManager.AddPlayerDBTask(enum.DB_POOL_TYPE_PLAYER, playerId, func(db *gorm.DB) error {
		defer decPlayerBarrier(playerId)
		if err := db.Where(where).Delete(new(T)).Error; err != nil {
			return err
		}
		return nil
	})
	return nil
}

type playerBarrier struct {
	cnt     atomic.Int32
	waiters []chan struct{}
	mu      sync.Mutex
}

func getPlayerBarrier(playerId int64) *playerBarrier {
	val, _ := playerBarrierMap.LoadOrStore(playerId, &playerBarrier{})
	return val.(*playerBarrier)
}

func incPlayerBarrier(playerId int64) {
	b := getPlayerBarrier(playerId)
	b.cnt.Add(1)
}

func decPlayerBarrier(playerId int64) {
	val, ok := playerBarrierMap.Load(playerId)
	if !ok {
		return
	}
	b := val.(*playerBarrier)

	if b.cnt.Add(-1) == 0 {
		// 唤醒所有等待者
		b.mu.Lock()
		for _, ch := range b.waiters {
			close(ch)
		}
		b.waiters = nil
		b.mu.Unlock()

		playerBarrierMap.Delete(playerId)
	}
}

func WaitPlayerBarrier(playerId int64, timeout time.Duration) error {
	val, ok := playerBarrierMap.Load(playerId)
	if !ok {
		// 没有 barrier，说明没有未完成 DB
		return nil
	}
	b := val.(*playerBarrier)

	// 再次检查（防 race）
	if b.cnt.Load() == 0 {
		return nil
	}

	ch := make(chan struct{})
	b.mu.Lock()
	b.waiters = append(b.waiters, ch)
	b.mu.Unlock()

	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
		return ErrWaitBarrierTimeout
	}
}

func GetPlayerDB() *gorm.DB {
	return dbPoolManager.DB

}
