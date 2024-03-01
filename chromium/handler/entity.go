package handler

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/imblowsnow/cgui/chromium/utils"
	"strings"
)

type FetchRequestEvent struct {
	Event       *fetch.EventRequestPaused
	Ctx         context.Context
	ExecutorCtx context.Context
	body        []byte

	requestHeaders  map[string]string
	responseHeaders map[string]string

	index int
	next  func(event *FetchRequestEvent)
	flag  bool
}

func (f *FetchRequestEvent) Init() {
	f.requestHeaders = make(map[string]string)
	for k, v := range f.Event.Request.Headers {
		f.requestHeaders[strings.ToLower(k)] = (v).(string)
	}

	f.responseHeaders = make(map[string]string)
	for _, header := range f.Event.ResponseHeaders {
		f.responseHeaders[strings.ToLower(header.Name)] = header.Value
	}
}
func (f *FetchRequestEvent) WithRequestExtraHeaders(headers network.Headers) {
	for k, v := range headers {
		if strings.HasPrefix(k, ":") {
			continue
		}
		f.requestHeaders[strings.ToLower(k)] = (v).(string)
	}
}
func (f *FetchRequestEvent) WithResponseExtraHeaders(headers network.Headers) {
	for k, v := range headers {
		f.responseHeaders[strings.ToLower(k)] = (v).(string)
	}
}
func (f *FetchRequestEvent) Next() {
	f.next(f)
}

func (f *FetchRequestEvent) AddRequestHeader(name string, value string) {
	f.RemoveRequestHeader(name)

	f.requestHeaders[strings.ToLower(name)] = value

	f.flag = true
}

func (f *FetchRequestEvent) RemoveRequestHeader(name string) {
	for headerName, _ := range f.requestHeaders {
		if strings.ToLower(headerName) == strings.ToLower(name) {
			delete(f.requestHeaders, headerName)
		}
	}
	f.flag = true
}
func (f *FetchRequestEvent) ModifyRequestHeaders(headers map[string]string) {
	f.requestHeaders = headers
	f.flag = true
}

func (f *FetchRequestEvent) AddResponseHeader(name string, value string) {
	// 追加响应头
	f.RemoveResponseHeader(name)
	f.responseHeaders[strings.ToLower(name)] = value

	f.flag = true
}

func (f *FetchRequestEvent) ModifyResponseHeader(headers map[string]string) {
	f.responseHeaders = headers

	f.flag = true
}

func (f *FetchRequestEvent) RemoveResponseHeader(name string) {
	for headerName, _ := range f.responseHeaders {
		// 删除原来的数据
		if strings.ToLower(headerName) == strings.ToLower(name) {
			delete(f.responseHeaders, headerName)
		}
	}
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

func (f *FetchRequestEvent) SetHandle() {
	f.flag = true
}

func (f *FetchRequestEvent) IsHandle() bool {
	return f.flag
}

func (f *FetchRequestEvent) GetSetCookies() string {
	cookies := f.responseHeaders["set-cookie"]
	return cookies
}

// 是否是主框架
func (f *FetchRequestEvent) IsPageFrame() bool {
	chromeCtx := chromedp.FromContext(f.ExecutorCtx)
	return f.Event.FrameID.String() == chromeCtx.Target.TargetID.String()
}

// 自动识别主页面/Frame 上下文
func (f *FetchRequestEvent) GetContext() context.Context {
	if !f.IsPageFrame() {
		return utils.GetFrameContext(f.Ctx, f.Event.FrameID.String())
	}
	return f.Ctx
}

func (f *FetchRequestEvent) GetRequestHeaders() map[string]string {
	return f.requestHeaders
}
func (f *FetchRequestEvent) GetResponseHeaders() map[string]string {
	return f.responseHeaders
}

func (f *FetchRequestEvent) BuildRequestHeaderEntry() []*fetch.HeaderEntry {
	var headers []*fetch.HeaderEntry
	for k, v := range f.requestHeaders {
		headers = append(headers, &fetch.HeaderEntry{Name: k, Value: v})
	}
	return headers
}

func (f *FetchRequestEvent) BuildResponseHeaderEntry() []*fetch.HeaderEntry {
	var headers []*fetch.HeaderEntry
	for k, v := range f.responseHeaders {
		headers = append(headers, &fetch.HeaderEntry{Name: k, Value: v})
	}
	return headers
}

func (f *FetchRequestEvent) SetCookie(cookieParam *network.CookieParam) {
	go func() {
		ctx := f.GetContext()

		chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			err := network.SetCookies([]*network.CookieParam{
				cookieParam,
			}).Do(ctx)
			if err != nil {
				fmt.Println("SetCookie error", cookieParam.Name, err.Error())
				return err
			}

			return nil
		}))
	}()
}
func (f *FetchRequestEvent) SetCookieS(cookies string, domain string) {
	properties := strings.Split(cookies, ";")
	cookieParam := &network.CookieParam{}

	// event.Event.Request.URL 使用 / 结尾
	domainUrl := strings.TrimRight(f.Event.Request.URL, "/")
	cookieParam.URL = domainUrl

	cookieParam.SameSite = network.CookieSameSiteNone
	// Iterate over the properties and set the corresponding fields in the CookieParam
	for i, property := range properties {
		parts := strings.SplitN(property, "=", 2)
		key, value := parts[0], parts[1]
		key = strings.TrimSpace(key)
		if i == 0 {
			cookieParam.Name = key
			cookieParam.Value = value
			continue
		}

		switch key {
		case "path":
			cookieParam.Path = value
		case "expires":
			fmt.Println("Error parsing date:", value)
		case "domain":
			cookieParam.Domain = value
		case "secure":
			cookieParam.Secure = true
		case "httponly":
			cookieParam.HTTPOnly = true
		case "samesite":
			cookieParam.SameSite = network.CookieSameSite(value)
		}
	}
	cookieParam.Domain = domain

	f.SetCookie(cookieParam)
}
