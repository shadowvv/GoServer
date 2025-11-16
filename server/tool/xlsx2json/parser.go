package xlsx2json

import (
	"github.com/xuri/excelize/v2"
	"strings"
)

type FieldInfo struct {
	Name    string
	Comment string
	Type    string
	Scope   string
}

type SheetData struct {
	Fields []FieldInfo
	Rows   []map[string]string // 原始字符串
}

// ParseExcel 读取 Excel 文件，返回 sheet 数据
func ParseExcel(file string) (map[string]SheetData, error) {
	f, err := excelize.OpenFile(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	result := make(map[string]SheetData)
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) < 4 {
			continue
		}
		headers := rows[0]  // 字段名
		comments := rows[1] // 注释
		scopes := rows[2]   // 使用范围
		types := rows[3]    // 类型
		var fields []FieldInfo
		for i := range headers {
			if i >= len(headers) {
				break
			}
			name := strings.TrimSpace(headers[i])
			if name == "" || name == "#" {
				continue
			}
			comment := ""
			if i < len(comments) {
				comment = strings.TrimSpace(comments[i])
			}
			typ := "str"
			if i < len(types) {
				typ = strings.TrimSpace(types[i])
			}
			scope := "cs"
			if i < len(scopes) {
				scope = strings.TrimSpace(scopes[i])
			}
			if scope == "c" {
				continue
			}
			fields = append(fields, FieldInfo{Name: name, Comment: comment, Type: typ, Scope: scope})
		}
		var dataRows []map[string]string
		for _, row := range rows[4:] {
			item := make(map[string]string)
			for j, field := range fields {
				if j < len(row) {
					item[field.Name] = strings.TrimSpace(row[j])
				} else {
					item[field.Name] = ""
				}
			}
			dataRows = append(dataRows, item)
		}
		result[sheet] = SheetData{Fields: fields, Rows: dataRows}
	}
	return result, nil
}
