package gameConfig

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/drop/GoServer/server/service/logger"
)

type configLoaderInterface interface {
	loadData() error
	checkData() error
	apply()
}

var configLoaderMap = make(map[string]configLoaderInterface)

func RegisterConfigLoader(name string, loader configLoaderInterface) {
	configLoaderMap[name] = loader
}

func CheckAllConfig() {

	green := "\033[32m"
	red := "\033[31m"

	for name, loader := range configLoaderMap {
		fmt.Print(fmt.Sprintf("%s [gameConfig] load config: %s \n", green, name))
		if err := loader.loadData(); err != nil {
			fmt.Print(fmt.Sprintf("%s [gameConfig] load %s error: %v \n", red, name, err))
			panic(err)
		}
		loader.apply()
	}
	for name, loader := range configLoaderMap {
		fmt.Print(fmt.Sprintf("%s [gameConfig] check config: %s \n", green, name))
		if err := loader.checkData(); err != nil {
			fmt.Print(fmt.Sprintf("%s [gameConfig] check %s error: %v \n", red, name, err))
			panic(err)
		}
		fmt.Print(fmt.Sprintf("%s [gameConfig] load config: %s success\n", green, name))
	}
}

func LoadAllConfig() {
	for name, loader := range configLoaderMap {
		logger.InfoWithSprintf("[gameConfig] load config: %s", name)
		if err := loader.loadData(); err != nil {
			logger.ErrorWithZapFields(fmt.Sprintf("[gameConfig] load %s error: %v", name, err))
			panic(err)
		}
		loader.apply()
	}
	for name, loader := range configLoaderMap {
		if err := loader.checkData(); err != nil {
			logger.ErrorWithZapFields(fmt.Sprintf("[gameConfig] check %s error: %v", name, err))
			panic(err)
		}
		logger.InfoWithSprintf("[gameConfig] load config: %s success", name)
	}

}

func ReloadAllConfig() {
	for name, loader := range configLoaderMap {
		logger.InfoWithSprintf("[gameConfig] reload config: %s", name)
		if loader == nil {
			logger.ErrorWithZapFields(fmt.Sprintf("[gameConfig] reload %s error: not found", name))
			return
		}
		if err := loader.loadData(); err != nil {
			logger.ErrorWithZapFields(fmt.Sprintf("[gameConfig] reload %s error: %v", name, err))
			return
		}
		if err := loader.checkData(); err != nil {
			logger.ErrorWithZapFields(fmt.Sprintf("[gameConfig] check %s error: %v", name, err))
			return
		}
		loader.apply()
	}
}

func ReloadTargetConfig(target string) {
	logger.InfoWithSprintf("[gameConfig] reload config: %s", target)
	loader := configLoaderMap[target]
	if loader == nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[gameConfig] reload %s error: not found", target))
		return
	}
	if err := loader.loadData(); err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[gameConfig] reload %s error: %v", target, err))
		return
	}
	if err := loader.checkData(); err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[gameConfig] check %s error: %v", target, err))
		return
	}
	loader.apply()
}

type ItemConfig struct {
	ID  int32
	Num int64
}

// ItemConfig 解析
func ParseItem(s string) *ItemConfig {
	parts := strings.Split(s, "~")
	if len(parts) != 2 {
		return nil
	}
	id, err1 := strconv.Atoi(parts[0])
	num, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return nil
	}
	return &ItemConfig{ID: int32(id), Num: num}
}

func ParseItemArray(s string) []*ItemConfig {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "|")
	arr := make([]*ItemConfig, 0)
	for _, p := range parts {
		item := ParseItem(strings.TrimSpace(p))
		if item != nil {
			arr = append(arr, item)
		}
	}
	return arr
}

func ParseItemMatrix(s string) [][]*ItemConfig {
	if s == "" {
		return nil
	}
	rows := strings.Split(s, ";")
	matrix := make([][]*ItemConfig, 0)
	for _, r := range rows {
		matrix = append(matrix, ParseItemArray(r))
	}
	return matrix
}

// int
func ParseInt(s string) int32 {
	i, _ := strconv.Atoi(s)
	return int32(i)
}

func ParseFloat(s string) float32 {
	f, _ := strconv.ParseFloat(s, 32)
	return float32(f)
}

func ParseInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// int[]
func ParseIntArray(s string) []int32 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "|")
	arr := make([]int32, 0, len(parts))
	for _, p := range parts {
		i, err := strconv.Atoi(strings.TrimSpace(p))
		if err == nil {
			arr = append(arr, int32(i))
		}
	}
	return arr
}

// int[][]
func ParseIntMatrix(s string) [][]int32 {
	if s == "" {
		return nil
	}
	rows := strings.Split(s, ";")
	matrix := make([][]int32, 0, len(rows))
	for _, r := range rows {
		matrix = append(matrix, ParseIntArray(r))
	}
	return matrix
}

// str[]
func ParseStrArray(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "|")
	arr := make([]string, 0, len(parts))
	for _, p := range parts {
		arr = append(arr, strings.TrimSpace(p))
	}
	return arr
}

// str[][]
func ParseStrMatrix(s string) [][]string {
	if s == "" {
		return nil
	}
	rows := strings.Split(s, ";")
	matrix := make([][]string, 0, len(rows))
	for _, r := range rows {
		matrix = append(matrix, ParseStrArray(r))
	}
	return matrix
}

func ParseTime(timeString string) (int64, error) {
	layout := "2006|01|02|01|01|01"
	t, err := time.ParseInLocation(layout, timeString, time.Local)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

func ParseTimeWithYMD(timeString string) (int64, error) {
	layout := "2006|01|02"
	t, err := time.ParseInLocation(layout, timeString, time.Local)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}
