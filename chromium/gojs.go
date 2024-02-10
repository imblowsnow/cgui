package chromium

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/chromedp/cdproto/fetch"
)

var bindCls struct {
	// 绑定方法
	Call   string `json:"call"`
	Params string `json:"params"`
}

func OnRequestGoUrl(ev *fetch.EventRequestPaused, executorCtx context.Context, binds []interface{}) {
	var headers []*fetch.HeaderEntry
	headers = append(headers, &fetch.HeaderEntry{Name: "Content-Type", Value: "application/json"})
	headers = append(headers, &fetch.HeaderEntry{Name: "Access-Control-Allow-Origin", Value: "*"})
	headers = append(headers, &fetch.HeaderEntry{Name: "Access-Control-Allow-Methods", Value: "GET, POST, OPTIONS"})
	headers = append(headers, &fetch.HeaderEntry{Name: "Access-Control-Allow-Headers", Value: "Content-Type"})

	// 跨域放行
	if ev.Request.Method == "OPTIONS" {
		fetch.FulfillRequest(ev.RequestID, 200).WithResponseHeaders(headers).Do(executorCtx)
		return
	}
	// ev.Request.PostData 转换为 json对象
	postData := ev.Request.PostData
	err := json.Unmarshal([]byte(postData), &bindCls)
	if err != nil {
		fmt.Println("json.Marshal error:", err)
		return
	}

	// 执行绑定方法
	result := CallBindJs(bindCls.Call, bindCls.Params)

	jsonByte, err := json.Marshal(result)

	fetch.FulfillRequest(ev.RequestID, 200).WithResponsePhrase("OK").WithResponseHeaders(headers).WithBody(base64.StdEncoding.EncodeToString(jsonByte)).Do(executorCtx)
}
