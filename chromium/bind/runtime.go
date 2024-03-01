package bind

import "github.com/imblowsnow/cgui/chromium/runtime"

func GetRuntimeBinds() []interface{} {
	return []interface{}{
		runtime.WebviewTest{},
	}
}
