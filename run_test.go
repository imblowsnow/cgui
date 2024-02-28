package main

import (
	"embed"
	"github.com/chromedp/chromedp"
	"github.com/imblowsnow/cgui/chromium"
	"github.com/tawesoft/golib/v2/dialog"
)

//go:embed all:frontend/dist
//go:embed project.json
var frontFiles embed.FS

func main() {
	err := chromium.Run(chromium.ChromiumOptions{
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
