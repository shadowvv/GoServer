package db

import (
	"gorm.io/gorm/schema"
	"unicode"
)

type CamelNamingStrategy struct {
	schema.NamingStrategy
}

// 重写 ColumnName 方法
func (CamelNamingStrategy) ColumnName(table, column string) string {
	// 首字母小写，保持驼峰
	if len(column) == 0 {
		return column
	}
	runes := []rune(column)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
