package easyDB

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var backendDB *gorm.DB

func SetBackendDB(db *gorm.DB) {
	backendDB = db
}

func GmCreatEntity(tableName string, entity map[string]interface{}) error {
	if tableName == "" {
		return fmt.Errorf("table is null")
	}

	if entity == nil || len(entity) == 0 {
		return fmt.Errorf("gm entity is null")
	}

	// 使用 Create 插入数据
	result := backendDB.Table(tableName).Create(entity)

	if result.Error != nil {
		return fmt.Errorf("gm creat is null:%v", result.Error)
	}

	// 可选：检查影响行数
	if result.RowsAffected == 0 {
		return fmt.Errorf("gm creat error")
	}

	return nil
}

func GmGetEntityByWhere(tableName string, where map[string]interface{}) map[string]interface{} {
	var res map[string]interface{}

	// 使用GORM的安全查询
	query := backendDB.Table(tableName)

	// 构建WHERE条件
	for key, value := range where {
		query = query.Where(key+" = ?", value)
	}

	// 查询并限制一条
	err := query.Limit(1).Take(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // 没找到记录
		}
		return nil
	}

	return res
}

func GmGetEntitiesByWhere(tableName string, where map[string]interface{}) []map[string]interface{} {
	var res []map[string]interface{}

	// 使用GORM的安全查询
	query := backendDB.Table(tableName)

	// 构建WHERE条件
	for key, value := range where {
		query = query.Where(key+" = ?", value)
	}

	// 查询并限制一条
	err := query.Find(&res).Error
	if err != nil || len(res) == 0 {
		return res
	}

	return res
}

func GmUpdateEntityByWhere(tableName string, entity map[string]interface{}, where map[string]interface{}) error {
	// 参数校验
	if tableName == "" {
		return fmt.Errorf("gm update table is null")
	}

	if entity == nil || len(entity) == 0 {
		return fmt.Errorf("gm update entity is null")
	}

	if where == nil || len(where) == 0 {
		return fmt.Errorf("gm update where is null")
	}

	// 构建更新查询
	query := backendDB.Table(tableName)

	// 添加WHERE条件
	if len(where) > 0 {
		for key, value := range where {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	// 执行更新
	result := query.Updates(entity)

	if result.Error != nil {
		return fmt.Errorf("gm update error")
	}

	return nil
}

func GmDeteleEntityByWhere(tableName string, where map[string]interface{}) error {
	if tableName == "" {
		return fmt.Errorf("gm del table is null")
	}

	if where == nil || len(where) == 0 {
		return fmt.Errorf("gm del where is null")
	}

	// 构建更新查询
	query := backendDB.Table(tableName)

	// 添加WHERE条件
	if len(where) > 0 {
		for key, value := range where {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}
	result := query.Delete(nil)

	if result.Error != nil {
		return fmt.Errorf("gm del fail: %w", result.Error)
	}

	return nil
}
