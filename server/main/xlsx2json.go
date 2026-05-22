package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/tool/xlsx2goloader"
	"github.com/drop/GoServer/server/tool/xlsx2json"
)

// 控制台颜色
var (
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	reset  = "\033[0m"
)

func main() {
	cfgPath := "serverTool/cfg.conf"
	cfg, err := xlsx2json.LoadConfig(cfgPath)
	if err != nil {
		fmt.Printf("%s❌ 读取配置失败: %v%s\n", red, err, reset)
		os.Exit(1)
	}

	files, _ := filepath.Glob(filepath.Join(cfg.ExcelDir, "*.xlsx"))
	var realFiles []string
	for _, f := range files {
		if strings.Contains(filepath.Base(f), "~$") {
			continue // 忽略 Excel 临时文件
		}
		realFiles = append(realFiles, f)
	}

	if len(realFiles) == 0 {
		fmt.Printf("%s⚠️ 未找到 Excel 文件%s\n", yellow, reset)
		return
	}

	// 分组
	dashGroup := make(map[string][]string)
	underscoreGroup := make(map[string][]string)
	ungrouped := make([]string, 0)

	for _, f := range realFiles {
		base := filepath.Base(f)
		if strings.Contains(base, "-") {
			prefix := xlsx2json.GetDashPrefix(f)
			dashGroup[prefix] = append(dashGroup[prefix], f)
		} else if strings.Contains(base, "_") {
			prefix := xlsx2json.GetUnderscorePrefix(f)
			underscoreGroup[prefix] = append(underscoreGroup[prefix], f)
		} else {
			ungrouped = append(ungrouped, f)
		}
	}

	processGroup := func(prefix string, list []string, typeStr string) {
		outJson := filepath.Join(cfg.JsonDir, prefix+".json")
		outGo := filepath.Join(cfg.GoDir, prefix+".go")

		fmt.Printf("%s[%s]%s 导出 %d 个文件 → JSON: %s\n", green, prefix, reset, len(list), outJson)
		if strings.Contains(typeStr, "-") {
			if err := xlsx2json.MergeAndWriteJSONByDash(list, outJson); err != nil {
				fmt.Printf("%s❌ JSON 导出失败: %v%s\n", red, err, reset)
				return
			}
		} else {
			if err := xlsx2json.MergeAndWriteJSONByUnderscore(list, outJson); err != nil {
				fmt.Printf("%s❌ JSON 导出失败: %v%s\n", red, err, reset)
				return
			}
		}

		// 传入第一个 Excel 文件路径，让 GenerateGoFile 能读取字段
		fmt.Printf("%s[%s]%s 生成 Go 文件 → %s\n", green, prefix, reset, outGo)
		if err := xlsx2goloader.GenerateGoFile(list, outJson, outGo, "cs"); err != nil {
			fmt.Printf("%s❌ Go 文件生成失败: %v%s\n", red, err, reset)
		}
	}

	// 中线分组处理
	if len(dashGroup) > 0 {
		fmt.Printf("%s📁 使用中线分组，找到 %d 个前缀组%s\n", yellow, len(dashGroup), reset)
		for prefix, list := range dashGroup {
			processGroup(prefix, list, "-")
		}
	}

	// 下划线分组处理
	if len(underscoreGroup) > 0 {
		fmt.Printf("%s📁 使用下划线分组，找到 %d 个前缀组%s\n", yellow, len(underscoreGroup), reset)
		for prefix, list := range underscoreGroup {
			processGroup(prefix, list, "_")
		}
	}

	// 未分组文件处理
	if len(ungrouped) > 0 {
		fmt.Printf("%s📁 未分组文件，导出 %d 个%s\n", yellow, len(ungrouped), reset)
		for _, f := range ungrouped {
			prefix := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
			processGroup(prefix, []string{f}, "")
		}
	}

	fmt.Printf("\n%s🎉 全部导出完成！地区：%s%s\n", green, cfg.Field, reset)

	fmt.Printf("\n%s 开始检测数据\n", green)
	// 检测配置文件数据
	gameConfig.CheckAllConfig()
	fmt.Printf("\n%s 开始检测数据完成\n", green)
}
