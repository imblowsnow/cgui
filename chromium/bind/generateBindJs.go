package bind

import (
	"reflect"
	"strings"
)

var bindMap = make(map[string]func(params string) any)

func GenerateBindJs(binds []interface{}) string {

	// 生成绑定js
	var js = "console.log(\"bind js\");\nwindow.go = window.go || {};\n"
	js += `window.go._call = function (call,params){
    return fetch("http://127.0.0.1/sub-jstogo", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            "call": call,
            "params": JSON.stringify(params)
        })
    }).then(response => {
        return response.json()
    });
}` + "\n"
	for _, bind := range binds {
		// 反射获取类名
		t := reflect.TypeOf(bind)

		arr := strings.Split(t.PkgPath()+"/"+t.Name(), "/")
		for i := 0; i < len(arr); i++ {
			name := strings.Join(arr[:i+1], ".")
			js += "window.go." + name + " = window.go." + name + " || {};\n"
		}
		jspath := strings.ReplaceAll(t.PkgPath()+"/"+t.Name(), "/", ".")

		// 反射获取方法名
		for i := 0; i < t.NumMethod(); i++ {
			method := t.Method(i)
			bindStr := `window.go.` + jspath + `.` + method.Name + ` = function(params) { return window.go._call("` + jspath + `.` + method.Name + `",params); }`
			js += bindStr + "\n"

			bindMap[jspath+"."+method.Name] = func(params string) any {
				values := method.Func.Call([]reflect.Value{reflect.ValueOf(bind), reflect.ValueOf(params)})
				if len(values) == 0 {
					return nil
				}
				return values[0].Interface()
			}
		}
	}
	return js
}

func CallBindJs(call string, params string) any {
	if call == "status" {
		return "ok"
	}
	if bindMap[call] == nil {
		return ""
	}
	return bindMap[call](params)
}
