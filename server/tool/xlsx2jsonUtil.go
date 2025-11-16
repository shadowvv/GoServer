package tool

import (
	"github.com/drop/GoServer/server/logic/model"
	"strconv"
	"strings"
)

// Item 解析
func ParseItem(s string) *model.Item {
	parts := strings.Split(s, "|")
	if len(parts) != 2 {
		return nil
	}
	id, err1 := strconv.Atoi(parts[0])
	num, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return nil
	}
	return &model.Item{ID: int32(id), Num: int32(num)}
}

func ParseItemArray(s string) []*model.Item {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	arr := make([]*model.Item, 0)
	for _, p := range parts {
		item := ParseItem(strings.TrimSpace(p))
		if item != nil {
			arr = append(arr, item)
		}
	}
	return arr
}

func ParseItemMatrix(s string) [][]*model.Item {
	if s == "" {
		return nil
	}
	rows := strings.Split(s, ";")
	matrix := make([][]*model.Item, 0)
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

// int[]
func ParseIntArray(s string) []int32 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
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
	parts := strings.Split(s, ",")
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
