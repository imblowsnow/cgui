package handler

import (
	"context"
	"github.com/chromedp/cdproto/fetch"
	"strings"
)

type FetchRequestEvent struct {
	Event *fetch.EventRequestPaused
	Ctx   context.Context
	body  []byte

	index int
	next  func(event *FetchRequestEvent)
	flag  bool
}

func (f *FetchRequestEvent) Next() {
	f.next(f)
}

func (f *FetchRequestEvent) AddRequestHeader(name string, value string) {
	f.Event.Request.Headers[name] = value

	f.flag = true
}

func (f *FetchRequestEvent) AddResponseHeader(name string, value string) {
	// 追加响应头
	for i, header := range f.Event.ResponseHeaders {
		// 删除原来的数据
		if strings.ToLower(header.Name) == strings.ToLower(name) {
			f.Event.ResponseHeaders = append(f.Event.ResponseHeaders[:i], f.Event.ResponseHeaders[i+1:]...)
		}
	}
	f.Event.ResponseHeaders = append(f.Event.ResponseHeaders, &fetch.HeaderEntry{Name: name, Value: value})

	f.flag = true
}

func (f *FetchRequestEvent) GetBody() []byte {
	if f.body == nil {
		return []byte{}
	}
	return f.body
}

func (f *FetchRequestEvent) SetBody(body []byte) {
	f.body = body
	f.flag = true
}

func (f *FetchRequestEvent) IsHandle() bool {
	return f.flag
}
