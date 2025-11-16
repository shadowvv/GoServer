package xlsx2goloader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/drop/GoServer/server/tool/xlsx2json"
)

// GenerateGoFile 从 Excel 文件生成 Go 文件
func GenerateGoFile(excelFile, outJson, outGo, clientScope string) error {
	// 从 Excel 文件读取字段信息和 Sheet 数据
	sheetMap, err := xlsx2json.ParseExcel(excelFile)
	if err != nil {
		return err
	}

	// Loader 名称
	prefix := strings.TrimSuffix(filepath.Base(outJson), filepath.Ext(outJson))
	loaderName := toCamel(prefix) + "CfgLoader"

	var builder strings.Builder
	builder.WriteString("package gameConfig\n\n")
	builder.WriteString("import (\n\t\"errors\"\n\t\"fmt\"\n\t\"sync/atomic\"\n\t\"github.com/drop/GoServer/server/logic/model\"\n\t\"github.com/drop/GoServer/server/service/fileLoader\"\n\t\"github.com/drop/GoServer/server/tool\"\n)\n\n")

	// ---- init 和 Loader struct ----
	builder.WriteString(fmt.Sprintf("func init() {\n\tRegisterConfigLoader(\"%s\", &%s{})\n}\n\n", prefix, loaderName))
	builder.WriteString(fmt.Sprintf("type %s struct {\n", loaderName))
	i := 1
	sheetKeys := make([]string, 0, len(sheetMap))
	for key := range sheetMap {
		sheetKeys = append(sheetKeys, key)
	}
	sort.Strings(sheetKeys)
	for _, key := range sheetKeys {
		builder.WriteString(fmt.Sprintf("\ttemp%d map[int32]*%sCfg\n", i, toCamel(key)))
		i++
	}
	builder.WriteString("}\n\n")
	builder.WriteString(fmt.Sprintf("var _ configLoaderInterface = (*%s)(nil)\n\n", loaderName))

	// ---- loadData 方法 ----
	builder.WriteString(fmt.Sprintf("func (s *%s) loadData() error {\n", loaderName))
	builder.WriteString("\tvar rawData map[string]map[string]map[string]string\n")
	jsonPath := filepath.ToSlash(outJson)
	builder.WriteString(fmt.Sprintf("\tif err := fileLoader.LoadJSON(`%s`, &rawData); err != nil {\n\t\treturn err\n\t}\n\n", jsonPath))

	i = 1
	for _, key := range sheetKeys {
		builder.WriteString(fmt.Sprintf("\ts.temp%d = make(map[int32]*%sCfg)\n", i, toCamel(key)))
		builder.WriteString(fmt.Sprintf("\tfor _, row := range rawData[\"%s\"] {\n", key))
		builder.WriteString(fmt.Sprintf("\t\tvar v %sCfg\n", toCamel(key)))
		for _, f := range sheetMap[key].Fields {
			if f.Scope == "#" || (clientScope == "c" && f.Scope == "s") || (clientScope == "s" && f.Scope == "c") {
				continue
			}
			fieldName := toCamel(f.Name)
			builder.WriteString(fmt.Sprintf("\t\tv.%s = row[\"%s\"]\n", fieldName, f.Name))
			switch f.Type {
			case "int":
				builder.WriteString(fmt.Sprintf("\t\tv.%s = tool.ParseInt(v.%s)\n", fieldName, fieldName))
			case "int[]":
				builder.WriteString(fmt.Sprintf("\t\tv.%s = tool.ParseIntArray(v.%s)\n", fieldName, fieldName))
			case "int[][]":
				builder.WriteString(fmt.Sprintf("\t\tv.%s = tool.ParseIntMatrix(v.%s)\n", fieldName, fieldName))
			case "str[]":
				builder.WriteString(fmt.Sprintf("\t\tv.%s = tool.ParseStrArray(v.%s)\n", fieldName, fieldName))
			case "str[][]":
				builder.WriteString(fmt.Sprintf("\t\tv.%s = tool.ParseStrMatrix(v.%s)\n", fieldName, fieldName))
			case "item":
				builder.WriteString(fmt.Sprintf("\t\tv.%s = tool.ParseItem(v.%s)\n", fieldName, fieldName))
			case "item[]":
				builder.WriteString(fmt.Sprintf("\t\tv.%s = tool.ParseItemArray(v.%s)\n", fieldName, fieldName))
			case "item[][]":
				builder.WriteString(fmt.Sprintf("\t\tv.%s = tool.ParseItemMatrix(v.%s)\n", fieldName, fieldName))
			}
		}
		builder.WriteString("\t\tif v.ID <= 0 {\n\t\t\tcontinue\n\t\t}\n")
		builder.WriteString(fmt.Sprintf("\t\ts.temp%d[v.ID] = &v\n", i))
		builder.WriteString("\t}\n\n")
		i++
	}
	builder.WriteString("\treturn nil\n}\n\n")

	// ---- checkData ----
	builder.WriteString(fmt.Sprintf("func (s *%s) checkData() error {\n", loaderName))
	i = 1
	for _, key := range sheetKeys {
		builder.WriteString(fmt.Sprintf("\tfor id, v := range s.temp%d {\n", i))
		builder.WriteString("\t\tif v.ID <= 0 {\n")
		builder.WriteString(fmt.Sprintf("\t\t\treturn errors.New(fmt.Sprintf(\"[gameConfig] load %s error invalid ID:%%d\", id))\n", key))
		builder.WriteString("\t\t}\n\t}\n")
		i++
	}
	builder.WriteString("\treturn nil\n}\n\n")

	// ---- apply ----
	builder.WriteString(fmt.Sprintf("func (s *%s) apply() {\n", loaderName))
	i = 1
	for _, key := range sheetKeys {
		builder.WriteString(fmt.Sprintf("\t%s.Store(s.temp%d)\n", key, i))
		builder.WriteString(fmt.Sprintf("\ts.temp%d = nil\n", i))
		i++
	}
	builder.WriteString("}\n\n")

	// ---- atomic 全局变量 ----
	for _, key := range sheetKeys {
		builder.WriteString(fmt.Sprintf("var %s atomic.Value\n", key))
	}
	builder.WriteString("\n")

	// ---- struct 定义 ----
	for _, key := range sheetKeys {
		builder.WriteString(fmt.Sprintf("// %sCfg 配置\n", key))
		builder.WriteString(fmt.Sprintf("type %sCfg struct {\n", toCamel(key)))
		for _, f := range sheetMap[key].Fields {
			if f.Scope == "#" || (clientScope == "c" && f.Scope == "s") || (clientScope == "s" && f.Scope == "c") {
				continue
			}
			builder.WriteString(fmt.Sprintf("\t// %s\n", f.Comment))
			builder.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", toCamel(f.Name), mapType(f.Type), f.Name))
		}
		builder.WriteString("}\n\n")
	}

	// ---- Getter ----
	for _, key := range sheetKeys {
		builder.WriteString(fmt.Sprintf("func Get%sCfg(id int32) *%sCfg {\n", toCamel(key), toCamel(key)))
		builder.WriteString(fmt.Sprintf("\tcfgMap := %s.Load()\n", key))
		builder.WriteString("\tif cfgMap == nil {\n\t\treturn nil\n\t}\n")
		builder.WriteString(fmt.Sprintf("\treturn cfgMap.(map[int32]*%sCfg)[id]\n", toCamel(key)))
		builder.WriteString("}\n\n")
	}

	return os.WriteFile(outGo, []byte(builder.String()), 0644)
}

// 驼峰转换
func toCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// Excel 类型映射到 Go 类型
func mapType(excelType string) string {
	switch excelType {
	case "int":
		return "int32"
	case "str":
		return "string"
	case "int[]":
		return "[]int32"
	case "str[]":
		return "[]string"
	case "int[][]":
		return "[][]int32"
	case "str[][]":
		return "[][]string"
	case "item":
		return "*model.Item"
	case "item[]":
		return "[]*model.Item"
	case "item[][]":
		return "[][]*model.Item"
	default:
		return "string"
	}
}
