package main

import (
	"embed"
	"fmt"
	"github.com/chromedp/cdproto/fetch"
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
			//chromedp.Flag("disable-web-security", true),
			// 隐身模式
			// chromedp.Flag("incognito", true),
		},
		RequestHandlers: []func(e *handler.FetchRequestEvent){
			func(e *handler.FetchRequestEvent) {
				if strings.HasPrefix(e.Event.Request.URL, "xxxx") {
					// 替换 referer
					e.AddRequestHeader("Referer", "xxxx")
					e.AddRequestHeader("Sec-Fetch-Dest", "empty")
					e.AddRequestHeader("Sec-Fetch-Mode", "cors")
					e.AddRequestHeader("Sec-Fetch-Site", "same-site")
				}
				fmt.Println("on request", e.Event.NetworkID, e.Event.RequestID, e.Event.Request.URL)
				e.Next()
			},
		},
		ResponseHandlers: []func(e *handler.FetchRequestEvent){
			func(e *handler.FetchRequestEvent) {
				e.Next()
			},
		},

		CorsFilter: func(paused *fetch.EventRequestPaused) bool {
			return true
		},

		App: &chromium.App{},

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
