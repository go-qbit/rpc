package rpc

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"github.com/go-qbit/rpc/openapi"
)

type validator interface {
	ToSwaggerSchema(f reflect.StructField, schema *openapi.Schema) error
	GetValidateFunc(f reflect.StructField) (validateFunc, error)
}

type validateFunc func(v interface{}) error

type vMinimumInt struct{}

func (v vMinimumInt) GetValue(f reflect.StructField) (interface{}, error) {
	if t, exists := f.Tag.Lookup("minimum"); exists {
		v, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	return nil, nil
}

func (v vMinimumInt) ToSwaggerSchema(f reflect.StructField, schema *openapi.Schema) error {
	val, err := v.GetValue(f)
	if err != nil {
		return err
	}

	if val != nil {
		schema.Minimum = val
	}

	return nil
}

func (v vMinimumInt) GetValidateFunc(f reflect.StructField) (validateFunc, error) {
	targetVal, err := v.GetValue(f)
	if err != nil {
		return nil, err
	}

	if targetVal == nil {
		return nil, nil
	}

	return func(v interface{}) error {
		var intVal int64
		switch v := v.(type) {
		case int:
			intVal = int64(v)
		case int8:
			intVal = int64(v)
		case int16:
			intVal = int64(v)
		case int32:
			intVal = int64(v)
		case int64:
			intVal = v
		default:
			panic("Unknown int type")
		}

		if intVal < targetVal.(int64) {
			name := f.Name
			if tag, exists := f.Tag.Lookup("json"); exists {
				name = tag
			}
			return fmt.Errorf("%s=%d is less than requred minimum %d", name, intVal, targetVal.(int64))
		}

		return nil
	}, nil
}

type vMinimumUint struct{}

func (v vMinimumUint) GetValue(f reflect.StructField) (interface{}, error) {
	if t, exists := f.Tag.Lookup("minimum"); exists {
		v, err := strconv.ParseUint(t, 10, 64)
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	return nil, nil
}

func (v vMinimumUint) ToSwaggerSchema(f reflect.StructField, schema *openapi.Schema) error {
	val, err := v.GetValue(f)
	if err != nil {
		return err
	}

	if val != nil {
		schema.Minimum = val
	}

	return nil
}

func (v vMinimumUint) GetValidateFunc(f reflect.StructField) (validateFunc, error) {
	targetVal, err := v.GetValue(f)
	if err != nil {
		return nil, err
	}

	if targetVal == nil {
		return nil, nil
	}

	return func(v interface{}) error {
		var uintVal uint64
		switch v := v.(type) {
		case uint:
			uintVal = uint64(v)
		case uint8:
			uintVal = uint64(v)
		case uint16:
			uintVal = uint64(v)
		case uint32:
			uintVal = uint64(v)
		case uint64:
			uintVal = v
		default:
			panic("Unknown int type")
		}

		if uintVal < targetVal.(uint64) {
			name := f.Name
			if tag, exists := f.Tag.Lookup("json"); exists {
				name = tag
			}
			return fmt.Errorf("%s=%d is less than requred minimum %d", name, uintVal, targetVal.(uint64))
		}

		return nil
	}, nil
}

type vMaximumInt struct{}

func (v vMaximumInt) GetValue(f reflect.StructField) (interface{}, error) {
	if t, exists := f.Tag.Lookup("maximum"); exists {
		v, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	return nil, nil
}

func (v vMaximumInt) ToSwaggerSchema(f reflect.StructField, schema *openapi.Schema) error {
	val, err := v.GetValue(f)
	if err != nil {
		return err
	}

	if val != nil {
		schema.Minimum = val
	}

	return nil
}

func (v vMaximumInt) GetValidateFunc(f reflect.StructField) (validateFunc, error) {
	targetVal, err := v.GetValue(f)
	if err != nil {
		return nil, err
	}

	if targetVal == nil {
		return nil, nil
	}

	return func(v interface{}) error {
		var intVal int64
		switch v := v.(type) {
		case int:
			intVal = int64(v)
		case int8:
			intVal = int64(v)
		case int16:
			intVal = int64(v)
		case int32:
			intVal = int64(v)
		case int64:
			intVal = v
		default:
			panic("Unknown int type")
		}

		if intVal > targetVal.(int64) {
			name := f.Name
			if tag, exists := f.Tag.Lookup("json"); exists {
				name = tag
			}
			return fmt.Errorf("%s=%d is greater than requred maximum %d", name, intVal, targetVal.(int64))
		}

		return nil
	}, nil
}

type vPattern struct{}

func (v vPattern) GetValue(f reflect.StructField) (string, error) {
	if t, exists := f.Tag.Lookup("pattern"); exists {
		if _, err := regexp.Compile(t); err != nil {
			return "", err
		}
		return t, nil
	}

	return "", nil
}

func (v vPattern) ToSwaggerSchema(f reflect.StructField, schema *openapi.Schema) error {
	val, err := v.GetValue(f)
	if err != nil {
		return err
	}

	if val != "" {
		schema.Pattern = val
	}

	return nil
}

func (v vPattern) GetValidateFunc(f reflect.StructField) (validateFunc, error) {
	pattern, err := v.GetValue(f)
	if err != nil {
		return nil, err
	}

	if pattern == "" {
		return nil, nil
	}

	re := regexp.MustCompile(pattern)
	return func(v interface{}) error {
		val := v.(string)
		if !re.MatchString(val) {
			name := f.Name
			if tag, exists := f.Tag.Lookup("json"); exists {
				name = tag
			}
			if len(val) > 25 {
				val = val[:22] + "..."
			}
			return fmt.Errorf("%s=%s does not match the pattern %s", name, val, pattern)
		}

		return nil
	}, nil
}

var (
	intValidators  = []validator{vMinimumInt{}, vMaximumInt{}}
	uintValidators = []validator{vMinimumUint{}, vMinimumUint{}}

	validators = map[reflect.Kind][]validator{
		reflect.Int:   intValidators,
		reflect.Int8:  intValidators,
		reflect.Int16: intValidators,
		reflect.Int32: intValidators,
		reflect.Int64: intValidators,

		reflect.Uint:   uintValidators,
		reflect.Uint8:  uintValidators,
		reflect.Uint16: uintValidators,
		reflect.Uint32: uintValidators,
		reflect.Uint64: uintValidators,

		reflect.String: []validator{vPattern{}},
	}
)
