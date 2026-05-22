package easyDB

import (
	"fmt"

	"gorm.io/gorm"
)

var rankDB *gorm.DB

func SetRankDB(db *gorm.DB) {
	rankDB = db
}

func CreateRankTable(tableName string) error {
	sql := fmt.Sprintf("CREATE TABLE `%s` LIKE `%s`", tableName, "rankBoardTemplate")
	return rankDB.Exec(sql).Error
}

func GetAllRankTable[T any](tableName string) map[string][]*T {
	allCommonRank := make(map[string][]*T)
	//获取所有以 tableName 为前缀的排行榜
	var rankTableList []string

	// 优先使用 rankDB，如果 rankDB 未初始化（如 backend 服务），则使用 gameDB
	db := rankDB
	if db == nil {
		if dbPoolManager == nil || dbPoolManager.DB == nil {
			return allCommonRank
		}
		db = dbPoolManager.DB
	}

	db.Raw("SHOW TABLES LIKE '" + tableName + "%'").Scan(&rankTableList)
	for _, table := range rankTableList {
		data, err := GetRankBoardData[T](table)
		if err != nil {
			continue
		}
		allCommonRank[table] = data
	}
	return allCommonRank
}

func GetRankBoardData[T any](tableName string) ([]*T, error) {
	var data []*T

	// 优先使用 rankDB，如果 rankDB 未初始化（如 backend 服务），则使用 gameDB
	db := rankDB
	if db == nil {
		// backend 服务中 rankDB 未初始化，回退到 gameDB
		if dbPoolManager == nil || dbPoolManager.DB == nil {
			return nil, fmt.Errorf("neither rankDB nor gameDB is initialized")
		}
		db = dbPoolManager.DB
	}

	err := db.Table(tableName).Find(&data).Error
	return data, err
}

func SaveRankBoardData[T any](t *T) error {
	return rankDB.Save(t).Error
}

func SaveRankBoardToDB[T any](rankId string, list []*T) error {
	if len(list) == 0 {
		return nil
	}
	return rankDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(fmt.Sprintf("DELETE FROM `%s`", rankId)).Error; err != nil {
			return err
		}
		if err := tx.Table(rankId).CreateInBatches(list, 500).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetRankTableNames 扫描数据库中所有以指定前缀开头的排行榜表名
func GetRankTableNames(prefix string) ([]string, error) {
	db := rankDB
	if db == nil {
		if dbPoolManager == nil || dbPoolManager.DB == nil {
			return nil, fmt.Errorf("neither rankDB nor gameDB is initialized")
		}
		db = dbPoolManager.DB
	}
	var tables []string
	err := db.Raw("SHOW TABLES LIKE '" + prefix + "%'").Scan(&tables).Error
	return tables, err
}

func GetRankDataByRaw[T any](sql string, values ...interface{}) ([]*T, error) {
	var data []*T
	err := rankDB.Raw(sql, values...).Scan(&data).Error
	return data, err
}

func RunRankRawSql(sql string, values ...interface{}) error {
	return rankDB.Exec(sql, values...).Error
}

func RunRankRawSqlWithRowsAffected(sql string, values ...interface{}) (int64, error) {
	result := rankDB.Exec(sql, values...)
	return result.RowsAffected, result.Error
}

func CreateRankRows[T any](tableName string, rows []*T) error {
	if len(rows) == 0 {
		return nil
	}
	return rankDB.Table(tableName).CreateInBatches(rows, 500).Error
}
