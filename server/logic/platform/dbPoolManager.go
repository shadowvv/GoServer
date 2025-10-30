package platform

import (
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/service/db"
	"gorm.io/gorm"
)

type DBPoolManager struct {
	dbPools map[enum.DBPoolType]*DBPool
	db      *gorm.DB
}

func NewDBPoolManager(db *gorm.DB) *DBPoolManager {
	return &DBPoolManager{
		dbPools: make(map[enum.DBPoolType]*DBPool),
		db:      db,
	}
}

func (m *DBPoolManager) AddDBPool(poolType enum.DBPoolType, workerNum, workerTaskSize int32) error {
	dbPool := NewDBPool(workerNum, workerTaskSize, db.DB)
	m.dbPools[poolType] = dbPool
	return nil
}
