package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/imblowsnow/cgui/chromium"
	"github.com/imblowsnow/cgui/chromium/handler"
	"github.com/tawesoft/golib/v2/dialog"
	"strings"
)

//go:embed all:frontend
//go:embed project.json
var frontFiles embed.FS

type TestBindJs struct {
}
type Result struct {
	Msg string
}

func (TestBindJs) Test1(params string) Result {
	fmt.Println("Test1", params)
	return Result{Msg: "Test1"}
}
func (TestBindJs) Test2(params string) Result {
	fmt.Println("Test2", params)
	return Result{Msg: "Test2"}
}

func main() {
	err := chromium.Run(&chromium.ChromiumOptions{
		FrontFiles:  frontFiles,
		FrontPrefix: "frontend",
		//UserDataDir: utils.GetCurrentBrowserFlagDir("default"),
		ChromeOpts: []chromedp.ExecAllocatorOption{
			// 禁用跨域安全策略
			// chromedp.Flag("disable-web-security", true),
			// 隐身模式
			// chromedp.Flag("incognito", true),
		},
		RequestHandlers: []func(event *handler.FetchRequestEvent){
			func(event *handler.FetchRequestEvent) {
				if strings.HasPrefix(event.Event.Request.URL, "https://www.xiaohongshu.com/") {
					// 替换 referer
					event.AddRequestHeader("Referer", "https://www.xiaohongshu.com/")
				}
				fmt.Println("on request", event.Event.NetworkID, event.Event.RequestID, event.Event.Request.URL)
				event.Next()
			},
		},
		ResponseHandlers: []func(event *handler.FetchRequestEvent){
			func(event *handler.FetchRequestEvent) {
				if strings.HasPrefix(event.Event.Request.URL, "https://www.xiaohongshu.com/") {
					//cookies := event.GetSetCookies()
					//event.AddResponseHeader("Set-Cookie", cookies+"; sameSite=None")
					//// Split the cookie string into individual properties
					//properties := strings.Split(cookies, ";")
					//
					//cookieParam := &network.CookieParam{}
					//
					//// event.Event.Request.URL 使用 / 结尾
					//domainUrl := strings.TrimRight(event.Event.Request.URL, "/")
					//cookieParam.URL = domainUrl
					//cookieParam.SameSite = network.CookieSameSiteNone
					//// Iterate over the properties and set the corresponding fields in the CookieParam
					//for i, property := range properties {
					//	parts := strings.SplitN(property, "=", 2)
					//	key, value := parts[0], parts[1]
					//	key = strings.TrimSpace(key)
					//	if i == 0 {
					//		cookieParam.Name = key
					//		cookieParam.Value = value
					//		continue
					//	}
					//
					//	switch key {
					//	case "path":
					//		cookieParam.Path = value
					//	case "expires":
					//		fmt.Println("Error parsing date:", value)
					//	case "domain":
					//		cookieParam.Domain = value
					//	}
					//}
					//
					//// 设置cookie到网站里面
					//err := network.SetCookies([]*network.CookieParam{
					//	cookieParam,
					//}).Do(event.Ctx)
					//if err != nil {
					//	fmt.Println("SetCookies error", err.Error())
					//}
					//fmt.Println("on response", event.Event.NetworkID, event.Event.RequestID, event.Event.Request.URL, cookies)
				}
				event.Next()
			},
		},

		CorsFilter: func(paused *fetch.EventRequestPaused) bool {
			return true
		},

		App: &chromium.App{
			OnReady: func(ctx context.Context) {
				fmt.Println("on ready")
				_, _, err := runtime.Evaluate("console.log(2333)").Do(ctx)
				if err != nil {
					fmt.Println("Evaluate error", err.Error())
				}
			},
		},

		Binds: []interface{}{
			TestBindJs{},
			Test,
		},
	})

	if err != nil {
		dialog.Error(err.Error())
	}
}

func Test(args string) (string, error) {
	fmt.Println("Test", args)
	return "Test", nil
}
