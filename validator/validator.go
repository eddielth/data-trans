package validator

import (
	"fmt"
	"reflect"
)

// Validator 表示数据验证器接口
type Validator interface {
	// Validate 验证数据
	Validate(data interface{}) error
}

// RangeValidator 表示范围验证器
type RangeValidator struct {
	Field string
	Min   float64
	Max   float64
}

// Validate 验证数据字段是否在指定范围内
func (rv *RangeValidator) Validate(data interface{}) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("数据必须是结构体类型")
	}

	field := v.FieldByName(rv.Field)
	if !field.IsValid() {
		return fmt.Errorf("字段 %s 不存在", rv.Field)
	}

	var value float64
	switch field.Kind() {
	case reflect.Float32, reflect.Float64:
		value = field.Float()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = float64(field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = float64(field.Uint())
	default:
		return fmt.Errorf("字段 %s 不是数值类型", rv.Field)
	}

	if value < rv.Min || value > rv.Max {
		return fmt.Errorf("字段 %s 的值 %f 不在范围 [%f, %f] 内", rv.Field, value, rv.Min, rv.Max)
	}

	return nil
}
