package handler

import (
	"context"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
)

type FetchHandler struct {
	handlers []func(event *FetchRequestEvent)
}

func (f *FetchHandler) Add(h func(event *FetchRequestEvent)) {
	f.handlers = append(f.handlers, h)
}
func (f *FetchHandler) Handle(ev *fetch.EventRequestPaused, ctx context.Context, extraHeader network.Headers) *FetchRequestEvent {
	extraResponseHeaders := make(map[string]string)
	if extraHeader != nil {
		for k, v := range extraHeader {
			extraResponseHeaders[k] = v.(string)
		}
	}

	event := &FetchRequestEvent{
		Event: ev,
		Ctx:   ctx,
		index: 0,
		next: func(event *FetchRequestEvent) {
			event.index++
			if event.index >= len(f.handlers) {
				return
			}
			f.handleNext(event, event.index)
		},
		ExtraResponseHeaders: extraResponseHeaders,
	}

	f.handleNext(event, 0)

	return event
}

func (f *FetchHandler) handleNext(event *FetchRequestEvent, index int) {
	var handler = f.handlers[index]

	handler(event)
}
