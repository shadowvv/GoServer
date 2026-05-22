package dbPool

import "gorm.io/gorm"

// DBTask 是一个数据库任务，支持返回错误。
type DBTask func(db *gorm.DB) error
