package xlsx2json

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"os"
	"strings"
)

type FieldInfo struct {
	Name    string
	Comment string
	Type    string
	Scope   string
	Index   int
}

type SheetData struct {
	Fields []FieldInfo
	Rows   []map[string]string // 原始字符串
}

// ParseExcel 读取 Excel 文件，返回 sheet 数据
func ParseExcel(file string) (map[string]SheetData, error) {
	keyName := file[strings.LastIndex(file, "\\")+1:]
	if strings.Contains(keyName, "-") {
		keyName = keyName[:strings.Index(keyName, "-")]
	} else if strings.Contains(keyName, "_") {
		keyName = keyName[strings.Index(keyName, "_")+1 : strings.Index(keyName, ".")]
	} else {
		keyName = keyName[:strings.Index(keyName, ".")]
	}

	// 只读模式打开
	fp, err := os.OpenFile(file, os.O_RDONLY, 0)
	if err != nil {
		fmt.Println("只读打开失败:", err)
		return nil, err
	}

	f, err := excelize.OpenReader(fp)
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
			if name == "#" {
				continue
			}
			if name == "" {
				break
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
			if scope == "c" || scope == "#" {
				continue
			}
			fields = append(fields, FieldInfo{Name: name, Comment: comment, Type: typ, Scope: scope, Index: i})
		}
		var dataRows []map[string]string
		for _, row := range rows[4:] {
			item := make(map[string]string)
			for j := 0; j < len(fields); j++ {
				field := fields[j]
				if field.Index < len(row) {
					item[field.Name] = strings.TrimSpace(row[field.Index])
				} else {
					item[field.Name] = ""
				}
			}
			dataRows = append(dataRows, item)
		}
		result[keyName] = SheetData{Fields: fields, Rows: dataRows}
		break
	}
	return result, nil
}
