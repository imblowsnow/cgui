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
	event.AddResponseHeader("Access-Control-Allow-Origin", "*")
}
