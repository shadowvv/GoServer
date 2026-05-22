package easyDB

import (
	"errors"

	"gorm.io/gorm"
)

var serverDB *gorm.DB

var ErrDuplicatedKey = errors.New("duplicated key")

func SetServerDB(db *gorm.DB) {
	serverDB = db
}

func CreateServerEntity[T any](t *T) error {
	if err := serverDB.Create(t).Error; err != nil {
		if isDuplicatedKeyError(err) {
			return ErrDuplicatedKey
		}
		return err
	}
	return nil
}

func isDuplicatedKeyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	errStr := err.Error()
	return contains(errStr, "Duplicate entry") && contains(errStr, "PRIMARY")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func GetServerAllEntities[T any]() ([]*T, error) {
	var t []*T
	if err := serverDB.Find(&t).Error; err != nil {
		return t, err
	}
	return t, nil
}

func GetServerEntityByWhere[T any](where map[string]interface{}) (*T, error) {
	var t T
	err := serverDB.Where(where).Take(&t).Error
	if err != nil {
		return &t, err
	}
	return &t, nil
}

func GetServerEntitiesByWhere[T any](where map[string]interface{}) ([]*T, error) {
	var t []*T
	err := serverDB.Where(where).Find(&t).Error
	if err != nil {
		return t, err
	}
	return t, nil
}

func GetServerEntitiesByCustomCondition[T any](conditions string, args ...interface{}) ([]*T, error) {
	var t []*T

	if err := serverDB.Where(conditions, args...).Find(&t).Error; err != nil {
		return t, err
	}
	return t, nil
}

func UpdateServerEntity[T any](t *T, changed map[string]interface{}) error {
	if len(changed) == 0 {
		return nil
	}
	return serverDB.Model(t).Updates(changed).Error
}

func UpdateServerEntities[t any](entity map[string]interface{}, where map[string]interface{}) {
	serverDB.Where(where).Find(&entity)
	return
}

func SaveSeverEntity[T any](entity *T) error {
	return serverDB.Save(entity).Error
}
