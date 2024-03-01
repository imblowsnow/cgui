package bind

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime"
	"strings"
)

func Bind(ctx context.Context, binds []interface{}) error {
	bindItems := buildBindItems(binds)

	for _, bindItem := range bindItems {
		err := Expose(ctx, bindItem.GetFullName(), bindItem.call)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildBindItems(binds []interface{}) []BindItem {
	var bindItems []BindItem

	for _, bind := range binds {
		// 反射获取类名
		t := reflect.TypeOf(bind)

		if t.Kind() == reflect.Struct {
			tmpbindItems := buildBindItemByStruct(bind)
			bindItems = append(bindItems, tmpbindItems...)
		} else if t.Kind() == reflect.Func {
			tmpbindItem := buildBindItemByFunc(bind)
			bindItems = append(bindItems, tmpbindItem)
		}
	}

	return bindItems
}
func buildBindItemByStruct(bind interface{}) []BindItem {
	t := reflect.TypeOf(bind)
	jspath := strings.ReplaceAll(t.PkgPath(), "/", ".")
	if strings.Contains(jspath, "github.com.imblowsnow.cgui.chromium.") {
		jspath = strings.ReplaceAll(jspath, "github.com.imblowsnow.cgui.chromium.", "")
	}
	var bindItems []BindItem
	// 反射获取方法名
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)

		bindItems = append(bindItems, BindItem{
			MethodName: method.Name,
			StructName: t.Name(),
			Path:       jspath,
			call: func(args string) (string, error) {
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
			},
		})
	}
	return bindItems
}

func buildBindItemByFunc(bind interface{}) BindItem {
	bindValue := reflect.ValueOf(bind)
	// 获取函数的指针
	funcPtr := bindValue.Pointer()

	// 使用反射获取函数名称
	funcName := runtime.FuncForPC(funcPtr).Name()
	// 分割函数名称 .
	list := strings.Split(funcName, ".")
	// 弹出最后一个元素
	funcName = list[len(list)-1]
	path := strings.Join(list[:len(list)-1], ".")

	bindItem := BindItem{
		MethodName: funcName,
		Path:       path,
		call: func(args string) (string, error) {
			values := bindValue.Call([]reflect.Value{reflect.ValueOf(args)})
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
		},
	}
	return bindItem
}
