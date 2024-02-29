package bind

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime"
	"strings"
)

func Bind(ctx context.Context, binds []interface{}) error {

	for _, bind := range binds {
		// 反射获取类名
		t := reflect.TypeOf(bind)

		// 判断是否是结构体
		var err error
		if t.Kind() == reflect.Struct {
			err = bindStruct(ctx, bind)
		} else if t.Kind() == reflect.Func {
			err = bindFunc(ctx, bind)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func bindStruct(ctx context.Context, bind interface{}) error {
	t := reflect.TypeOf(bind)
	jspath := strings.ReplaceAll(t.PkgPath()+"/"+t.Name(), "/", ".")
	// 反射获取方法名
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)

		err := Expose(ctx, jspath, func(args string) (string, error) {
			values := method.Func.Call([]reflect.Value{reflect.ValueOf(bind), reflect.ValueOf(args)})
			if len(values) == 0 {
				return "nil", nil
			}
			// 判断类型
			if values[0].Kind() == reflect.String {
				return values[0].String(), nil
			}
			jsonBytes, err := json.Marshal(values[0].Interface())
			if err != nil {
				return "", err
			}
			return string(jsonBytes), nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func bindFunc(ctx context.Context, bind interface{}) error {
	funcName := runtime.FuncForPC(reflect.ValueOf(bind).Pointer()).Name()
	jspath := strings.ReplaceAll(funcName, "/", ".")
	err := Expose(ctx, jspath, func(args string) (string, error) {
		values := reflect.ValueOf(bind).Call([]reflect.Value{reflect.ValueOf(args)})
		if len(values) == 0 {
			return "", nil
		}
		// 判断类型
		if values[0].Kind() == reflect.String {
			return values[0].String(), nil
		}
		jsonBytes, err := json.Marshal(values[0].Interface())
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	})
	if err != nil {
		return err
	}
	return nil
}
