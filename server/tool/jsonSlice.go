package tool

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type JSONInt64Slice []int64

func (j *JSONInt64Slice) Scan(value interface{}) error {
	if value == nil {
		*j = JSONInt64Slice{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("JSONInt64Slice: 无法扫描类型 %T", value)
	}

	return json.Unmarshal(bytes, j)
}

func (j JSONInt64Slice) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

type JSONInt32Slice []int32

func (j *JSONInt32Slice) Scan(value interface{}) error {
	if value == nil {
		*j = JSONInt32Slice{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("JSONInt32Slice.Scan: 无法处理类型 %T", v)
	}

	return json.Unmarshal(bytes, j)
}

func (j JSONInt32Slice) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func Int32SliceTostring(nums []int32, mid string) string {
	if len(nums) == 0 {
		return ""
	}

	// 创建字符串切片
	strParts := make([]string, len(nums))

	// 遍历 int32 切片，将每个元素转换为字符串
	for i, num := range nums {
		strParts[i] = strconv.FormatInt(int64(num), 10)
	}

	// 使用 strings.Join 将字符串切片用 "|" 连接
	return strings.Join(strParts, mid)
}
