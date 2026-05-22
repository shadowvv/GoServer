package robotUtils

import (
	"strconv"
	"strings"
)

func GenerateAccount(baseAccount string, index int) string {
	if baseNum, err := strconv.Atoi(baseAccount); err == nil {
		return strconv.Itoa(baseNum + index)
	}
	return baseAccount + strconv.Itoa(index)
}

func NormalizeString(module string) string {
	return strings.ToLower(strings.TrimSpace(module))
}

// 将蛇形命名转换为驼峰命名
func SnakeToCamel(s string) string {
	if s == "" {
		return s
	}

	parts := strings.Split(s, "_")
	result := ""
	first := true
	for _, part := range parts {
		if part == "" {
			continue
		}

		if first {
			result += strings.ToLower(part[:1]) + part[1:]
			first = false
			continue
		}

		result += strings.ToUpper(part[:1]) + part[1:]
	}

	return result
}

// 将驼峰命名转换为蛇形命名
func CamelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// 转换为整数
func ConvertToInt(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	}
	return 0, false
}
