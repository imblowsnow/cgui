package handler

import (
	"context"
	"github.com/chromedp/cdproto/fetch"
)

func DefaultHandler(ev *fetch.EventRequestPaused, ctx context.Context, body []byte) {
	// 默认处理
	if body == nil || len(body) == 0 {
		fetch.ContinueRequest(ev.RequestID).Do(ctx)
	} else {

	}
}

func CorsHandler(event *FetchRequestEvent) {
	// 追加响应头
	//for i, header := range ev.ResponseHeaders {
	//	// 删除原来的跨域数据
	//	if strings.ToLower(header.Name) == strings.ToLower("Access-Control-Allow-Origin") {
	//		ev.ResponseHeaders = append(ev.ResponseHeaders[:i], ev.ResponseHeaders[i+1:]...)
	//	}
	//}
	//ev.ResponseHeaders = append(ev.ResponseHeaders, &fetch.HeaderEntry{Name: "Access-Control-Allow-Origin", Value: "*"})

}
