package platform

import (
	"fmt"
	"github.com/drop/GoServer/server/logic/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/service/db"
	"gorm.io/gorm"
	"reflect"
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

func (m *DBPoolManager) initModel() {
	err := m.db.AutoMigrate(&model.UserModel{})
	if err != nil {
		return
	}
}

func (m *DBPoolManager) AddDBPool(poolType enum.DBPoolType, workerNum, workerTaskSize int32) error {
	dbPool := NewDBPool(workerNum, workerTaskSize, db.DB)
	m.dbPools[poolType] = dbPool
	return nil
}

func (m *DBPoolManager) AddDBTask(poolType enum.DBPoolType, playerID int64, task DBTask) {
	dbPool := m.dbPools[poolType]
	dbPool.Submit(playerID, task)
}

func GetByStringID[T any](pkName string, id any) (*T, error) {
	var t T
	if err := dbPoolManager.db.Where(fmt.Sprintf("%s = ?", pkName), id).Take(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func GetByID[T any](id any) (*T, error) {
	var t T

	// 确保反射类型是结构体
	v := reflect.ValueOf(&t)
	if v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Ptr {
		panic("T should not be a pointer type")
	}

	if err := dbPoolManager.db.Take(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func Save[T any](t *T) error {
	dbPoolManager.AddDBTask(enum.DB_POOL_TYPE_LOGIN, 0, func(db *gorm.DB) {
		err := dbPoolManager.db.Save(t).Error
		if err != nil {
			panic(err)
		}
	})
	return nil
}
