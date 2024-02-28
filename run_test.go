package main

import (
	"embed"
	"github.com/chromedp/chromedp"
	"github.com/tawesoft/golib/v2/dialog"
	"main/chromium"
)

//go:embed all:frontend/dist
var frontFiles embed.FS

func main() {
	err := chromium.Run(chromium.ChromiumOptions{
		//Url:               "https://www.browserscan.net/zh",
		FrontFiles:  frontFiles,
		FrontPrefix: "frontend/dist",
		//UserDataDir: utils.GetCurrentBrowserFlagDir("default"),
		ChromeOpts: []chromedp.ExecAllocatorOption{
			// 禁用跨域安全策略
			// chromedp.Flag("disable-web-security", true),
			// 隐身模式
			// chromedp.Flag("incognito", true),
		},
	})

	if err != nil {
		dialog.Error(err.Error())
	}
}