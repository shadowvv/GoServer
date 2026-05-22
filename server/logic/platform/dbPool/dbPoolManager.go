package dbPool

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DBPoolManager struct {
	dbPools map[enum.DBPoolType]*DBPool
	DB      *gorm.DB
}

func NewDBPoolManager(db *gorm.DB) *DBPoolManager {
	return &DBPoolManager{
		dbPools: make(map[enum.DBPoolType]*DBPool),
		DB:      db,
	}
}

func (m *DBPoolManager) AddDBPool(poolType enum.DBPoolType, workerNum, queueSize int32) {
	m.dbPools[poolType] = NewDBPool(workerNum, queueSize, m.DB)
}

func (m *DBPoolManager) AddPlayerDBTask(poolType enum.DBPoolType, playerID int64, task DBTask) {
	pool := m.dbPools[poolType]
	if pool == nil {
		logger.ErrorWithZapFields("[db] invalid pool type", zap.String("poolType", string(poolType)))
		return
	}
	pool.Submit(playerID, task)
}
