package xlsx2json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 合并 SheetData
func mergeSheetData(dst *SheetData, src SheetData) {
	// 字段只取第一份（假定所有 Excel 格式一致）
	if len(dst.Fields) == 0 {
		dst.Fields = src.Fields
	}
	// 合并 Rows
	dst.Rows = append(dst.Rows, src.Rows...)
}

// MergeAndWriteJSONByDash 合并同前缀的 Excel 文件（如 drop-*）并导出结构化 JSON
func MergeAndWriteJSONByDash(files []string, outPath string) error {
	if len(files) == 0 {
		return fmt.Errorf("没有需要导出的文件")
	}

	// 取前缀，例如 drop-test1.xlsx / drop-test2.xlsx → drop
	baseName := filepath.Base(files[0])
	prefix := strings.Split(baseName, "-")[0]

	final := make(map[string]map[string]map[string]string)
	final[prefix] = make(map[string]map[string]string)

	for _, f := range files {
		data, err := ParseExcel(f)
		if err != nil {
			fmt.Println("❌ 解析失败:", f, err)
			continue
		}

		for _, sheetData := range data {
			for _, row := range sheetData.Rows {
				idVal, ok := row["id"]
				if !ok {
					fmt.Println("⚠️ 找不到 id 字段:", f)
					continue
				}
				// row 是 map[string]string（或 interface{}→string）

				// 复制一份 map
				cleanRow := make(map[string]string)
				for k, v := range row {
					cleanRow[k] = v
				}

				final[prefix][idVal] = cleanRow
			}
		}
	}

	jsonBytes, err := json.MarshalIndent(final, "", " ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(outPath, jsonBytes, 0644)
}

// MergeAndWriteJSONByUnderscore 处理 _ 前缀（与上面相同逻辑）
func MergeAndWriteJSONByUnderscore(files []string, outPath string) error {
	final := make(map[string]map[string]map[string]string)

	for _, file := range files {
		data, err := ParseExcel(file)
		if err != nil {
			return err
		}

		// 文件名 hero_test1 → 子 key = test1
		_, short := filepath.Split(file)
		name := strings.TrimSuffix(short, filepath.Ext(short))
		parts := strings.Split(name, "-")
		if len(parts) < 2 {
			parts = strings.Split(name, "_")
		}
		childKey := parts[len(parts)-1] // test1

		// 解析多个 sheet
		childMap := make(map[string]map[string]string)

		for _, sheetData := range data {
			for _, row := range sheetData.Rows {
				id := row["id"]
				if id == "" {
					continue
				}
				childMap[id] = row
			}
		}

		final[childKey] = childMap
	}

	// 写入 JSON 文件
	b, err := json.MarshalIndent(final, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outPath, b, 0644)
}

// 前缀解析
func GetDashPrefix(filename string) string {
	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if idx := strings.Index(name, "-"); idx != -1 {
		return name[:idx]
	}
	return name
}

func GetUnderscorePrefix(filename string) string {
	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if idx := strings.Index(name, "_"); idx != -1 {
		return name[:idx]
	}
	return name
}
