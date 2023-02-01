package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"reflect"
	"regexp"
	"strings"
)

type Method interface {
	Caption(ctx context.Context) string
	Description(ctx context.Context) string
}

type MethodDesc struct {
	Path       string
	Method     Method
	Request    reflect.Type
	Response   reflect.Type
	Func       reflect.Value
	Errors     map[string]string
	Validators map[string][]validateFunc
}

type File io.ReadCloser

type buffer struct {
	b *bytes.Buffer
}

func (b *buffer) Close() error {
	return nil
}

func (b *buffer) Read(p []byte) (int, error) {
	return b.b.Read(p)
}

var (
	reMethodVersion = regexp.MustCompile(`^V\d+$`)
	reErrorsVersion = regexp.MustCompile(`^ErrorsV(\d+)$`)
)

func getMethodPath(m Method, trimPrefix string) (string, error) {
	trimPrefix = strings.TrimSuffix(trimPrefix, "/")

	mType := reflect.TypeOf(m)

	path := mType.PkgPath()
	if path == "" && mType.Kind() == reflect.Ptr {
		path = mType.Elem().PkgPath()
	}

	if !strings.HasPrefix(path, trimPrefix) {
		return "", fmt.Errorf("invalid trim prefix '%s' for '%s'", trimPrefix, path)
	}

	return strings.TrimPrefix(path, trimPrefix), nil
}

func ParseMethodDesc(m Method, trimPrefix string) ([]*MethodDesc, error) {
	return descsFromMethod(m, trimPrefix)
}

func descsFromMethod(m Method, trimPrefix string) ([]*MethodDesc, error) {
	path, err := getMethodPath(m, trimPrefix)
	if err != nil {
		return nil, err
	}

	var res []*MethodDesc

	mType := reflect.TypeOf(m)
	for i := 0; i < mType.NumMethod(); i++ {
		goMethod := mType.Method(i)

		if reMethodVersion.MatchString(goMethod.Name) {
			if goMethod.Type.NumIn() != 3 || goMethod.Type.In(1).String() != "context.Context" {
				return nil, fmt.Errorf("invalid method %s signature, must be (context.Context, <request type>)", goMethod.Name)
			}

			if goMethod.Type.NumOut() != 2 || goMethod.Type.Out(1).String() != "error" {
				return nil, fmt.Errorf("invalid method %s return signature, must be (<response type>), error", goMethod.Name)
			}

			validators := map[string][]validateFunc{}
			if err := getValidators(goMethod.Type.In(2), validators, ""); err != nil {
				return nil, err
			}
			res = append(res, &MethodDesc{
				Path:       path + "/" + strings.ToLower(goMethod.Name),
				Method:     m,
				Func:       goMethod.Func,
				Request:    goMethod.Type.In(2),
				Response:   goMethod.Type.Out(0),
				Errors:     map[string]string{},
				Validators: validators,
			})
		}
	}

	return res, nil
}

func getValidators(t reflect.Type, validatorsMap map[string][]validateFunc, curPath string) error {
	switch t.Kind() {
	case reflect.Ptr:
		return getValidators(t.Elem(), validatorsMap, curPath)

	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldType := field.Type

			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}

			switch fieldType.Kind() {
			case reflect.Struct:
				if err := getValidators(field.Type, validatorsMap, curPath+"/"+field.Name); err != nil {
					return err
				}

			default:
				for _, validator := range validators[field.Type.Kind()] {
					vFunc, err := validator.GetValidateFunc(field)
					if err != nil {
						return err
					}

					if vFunc != nil {
						path := curPath + "/" + field.Name
						validatorsMap[path] = append(validatorsMap[path], vFunc)
					}
				}
			}
		}
	}

	return nil
}

func bindErrors(m Method, trimPrefix string, methods map[string]*MethodDesc) error {
	path, err := getMethodPath(m, trimPrefix)
	if err != nil {
		return err
	}

	mType := reflect.TypeOf(m)
	for i := 0; i < mType.NumMethod(); i++ {
		goMethod := mType.Method(i)

		submatch := reErrorsVersion.FindStringSubmatch(goMethod.Name)
		if len(submatch) < 2 {
			continue
		}

		methodPath := path + "/v" + submatch[1]

		errorsVar := goMethod.Func.Call([]reflect.Value{reflect.ValueOf(m)})[0]
		errorsVar = errorsVar.Elem()
		if errorsVar.Kind() != reflect.Ptr || errorsVar.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("errors variable must be a pointer to a structure")
		}
		errorsVar = errorsVar.Elem()

		for i := 0; i < errorsVar.NumField(); i++ {
			ft := errorsVar.Type().Field(i)
			if ft.Type.Name() != "ErrorFunc" || ft.Type.PkgPath() != "github.com/go-qbit/rpc" {
				return fmt.Errorf("error type for %s must be github.com/go-qbit/rpc.ErrorFunc", ft.Name)
			}

			methods[methodPath].Errors[ft.Name] = ft.Tag.Get("desc")

			f := errorsVar.Field(i)
			errFunc := ErrorFunc(func(message string, data ...interface{}) *Error {
				res := &Error{
					Code:    ft.Name,
					Message: message,
				}

				if len(data) > 0 {
					res.Data = data[0]
				}

				return res
			})

			f.Set(reflect.ValueOf(errFunc))
		}
	}

	return nil
}

func (m *MethodDesc) Call(ctx context.Context, r io.Reader, boundary string, maxMemory int64) (interface{}, error) {
	req := reflect.New(m.Request.Elem())

	if boundary != "" {
		reader := multipart.NewReader(r, boundary)
		for {
			p, err := reader.NextPart()
			// This is OK, no more parts
			if err == io.EOF {
				break
			}

			if err != nil {
				return nil, err
			}

			var file File

			if name, ok := checkFileField(p.FormName(), req.Elem().Type()); ok {
				buf := &bytes.Buffer{}
				n, err := io.CopyN(buf, p, maxMemory+1)
				if err != nil && err != io.EOF {
					return nil, err
				}
				file = &buffer{buf}
				if n > maxMemory {
					tmp, err := os.CreateTemp("", "rpc-multipart-")
					if err != nil {
						return nil, err
					}
					_, err = io.Copy(tmp, io.MultiReader(buf, p))
					if err != nil {
						os.Remove(tmp.Name())
						return nil, err
					}
					_, err = tmp.Seek(0, 0)
					if err != nil {
						os.Remove(tmp.Name())
						return nil, err
					}
					file = tmp

				}
				req.Elem().FieldByName(name).Set(reflect.ValueOf(file))
				continue
			}
			if err := json.NewDecoder(p).Decode(req.Interface()); err != nil && err != io.EOF {
				return nil, &Error{Code: "INVALID_JSON", Message: err.Error()}
			}
		}

	} else {
		if err := json.NewDecoder(r).Decode(req.Interface()); err != nil {
			return nil, &Error{Code: "INVALID_JSON", Message: err.Error()}
		}
	}

	if len(m.Validators) > 0 {
		if err := m.validateData(req, ""); err != nil {
			return nil, &Error{Code: "INVALID_JSON", Message: err.Error()}
		}
	}

	res := m.Func.Call([]reflect.Value{reflect.ValueOf(m.Method), reflect.ValueOf(ctx), req})

	if !res[1].IsNil() {
		return nil, res[1].Interface().(error)
	}

	return res[0].Interface(), nil
}

func checkFileField(partName string, t reflect.Type) (string, bool) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if partName == field.Tag.Get("json") && t.Field(i).Type.Implements(reflect.TypeOf((*File)(nil)).Elem()) {
			return t.Field(i).Name, true
		}
	}
	return "", false
}

func (m *MethodDesc) validateData(data reflect.Value, curPath string) error {
	switch data.Type().Kind() {
	case reflect.Ptr:
		if !data.IsNil() {
			return m.validateData(data.Elem(), curPath)
		}

	case reflect.Struct:
		for i := 0; i < data.NumField(); i++ {
			fieldVal := data.Field(i)
			fieldType := data.Type().Field(i)
			ft := fieldType.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}

			if ft.Kind() == reflect.Struct {
				if err := m.validateData(fieldVal, curPath+"/"+fieldType.Name); err != nil {
					return err
				}
				continue
			}

			for _, validate := range m.Validators[curPath+"/"+fieldType.Name] {
				if err := validate(fieldVal.Interface()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
