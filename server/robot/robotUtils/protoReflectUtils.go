package robotUtils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/drop/GoServer/server/logic/pb"
	"google.golang.org/protobuf/proto"
)

func BuildMessageWithParams(msg proto.Message, params map[string]interface{}) error {
	v := reflect.ValueOf(msg).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 跳过 protobuf 内部字段
		if strings.HasPrefix(field.Name, "state") || strings.HasPrefix(field.Name, "sizeCache") || strings.HasPrefix(field.Name, "unknownFields") {
			continue
		}

		// 查找对应的参数（支持多种命名格式）
		paramName := strings.ToLower(field.Name)
		paramValue, ok := params[paramName]
		if !ok {
			// 尝试驼峰命名
			camelName := SnakeToCamel(field.Name)
			paramValue, ok = params[camelName]
			if !ok {
				// 尝试下划线命名
				snakeName := CamelToSnake(camelName)
				paramValue, ok = params[snakeName]
			}
		}

		if !ok {
			return fmt.Errorf("field %q doesn't exist in field %q", paramName, field.Name)
		}

		// 设置字段值
		if err := setFieldValue(fieldValue, paramValue); err != nil {
			return fmt.Errorf("设置字段 %s 失败: %v", field.Name, err)
		}
	}

	return nil
}

// 设置字段值
func setFieldValue(field reflect.Value, value interface{}) error {
	if !field.CanSet() {
		return fmt.Errorf("字段不可设置")
	}

	valueType := reflect.TypeOf(value)
	fieldType := field.Type()

	// 类型匹配
	if valueType.AssignableTo(fieldType) {
		field.Set(reflect.ValueOf(value))
		return nil
	}

	// 类型转换
	switch fieldType.Kind() {
	case reflect.Int32, reflect.Int64:
		if intVal, ok := ConvertToInt(value); ok {
			field.SetInt(int64(intVal))
			return nil
		}
	case reflect.Bool:
		if boolVal, ok := value.(bool); ok {
			field.SetBool(boolVal)
			return nil
		}
	case reflect.Slice:
		if sliceVal, ok := value.([]int64); ok && fieldType.Elem().Kind() == reflect.Int64 {
			field.Set(reflect.ValueOf(sliceVal))
			return nil
		}
	}

	return fmt.Errorf("无法转换类型: %v -> %v", valueType, fieldType)
}

func ParseMessageID(idString string) (pb.MESSAGE_ID, error) {
	key := strings.TrimSpace(idString)
	if key == "" {
		return 0, fmt.Errorf("empty message id")
	}

	if n, err := strconv.ParseUint(key, 10, 32); err == nil {
		return pb.MESSAGE_ID(n), nil
	}

	lastDot := strings.LastIndex(key, ".")
	if lastDot >= 0 && lastDot < len(key)-1 {
		key = key[lastDot+1:]
	}

	enumKey := strings.TrimPrefix(strings.ToUpper(key), "MESSAGE_ID_")
	enumKey = strings.ReplaceAll(enumKey, "-", "_")
	if v, ok := pb.MESSAGE_ID_value[enumKey]; ok {
		return pb.MESSAGE_ID(v), nil
	}

	return 0, fmt.Errorf("unknown message id %q", idString)
}
