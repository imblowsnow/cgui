package handler

import (
	"encoding/json"
	"github.com/imblowsnow/cgui/chromium/bind"
	"strings"
)

var bindCls struct {
	// 绑定方法
	Call   string `json:"call"`
	Params string `json:"params"`
}

func BindHandler(event *FetchRequestEvent) {
	if strings.HasSuffix(event.Event.Request.URL, "/sub-jstogo") {
		ev := event.Event

		event.AddResponseHeader("Content-Type", "application/json")
		event.AddResponseHeader("Access-Control-Allow-Origin", "*")
		event.AddResponseHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		event.AddResponseHeader("Access-Control-Allow-Headers", "Content-Type")

		// 跨域放行
		if ev.Request.Method == "OPTIONS" {
			return
		}
		// ev.Request.PostData 转换为 json对象
		postData := ev.Request.PostData
		err := json.Unmarshal([]byte(postData), &bindCls)
		if err != nil {
			return
		}

		// 执行绑定方法
		result := bind.CallBindJs(bindCls.Call, bindCls.Params)

		jsonByte, err := json.Marshal(result)

		event.SetBody(jsonByte)

		return

	}

	event.Next()
}
