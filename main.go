package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/imblowsnow/cgui/chromium"
	"github.com/imblowsnow/cgui/chromium/handler"
	"github.com/tawesoft/golib/v2/dialog"
)

//go:embed all:frontend
//go:embed project.json
var frontFiles embed.FS

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
				fmt.Println("on request", event.Event.Request.URL)
				event.Next()
			},
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
	})

	if err != nil {
		dialog.Error(err.Error())
	}
}
