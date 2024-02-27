package main

import (
	"embed"
	"fmt"
	"github.com/chromedp/cdproto/fetch"
	"github.com/imblowsnow/chromedp"
	"github.com/tawesoft/golib/v2/dialog"
	"main/chromium"
)

//go:embed front/*
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
	userDataDir := chromium.GetCurrentBrowserFlagDir("default")

	// https://www.browserscan.net/zh
	// https://bot.sannysoft.com/

	err := chromium.Run(chromium.ChromiumOptions{
		//Url:               "https://www.browserscan.net/zh",
		UserDataDir: userDataDir,
		FrontFiles:  frontFiles,
		CorsFilter: func(paused *fetch.EventRequestPaused) bool {
			return true
		},
		ChromeOpts: []chromedp.ExecAllocatorOption{
			// 禁用跨域安全策略
			// chromedp.Flag("disable-web-security", true),
			// 隐身模式
			// chromedp.Flag("incognito", true),
		},
		RandomFingerprint: true,
		Binds: []interface{}{
			TestBindJs{},
		},
	})
	if err != nil {
		dialog.Error(err.Error())
	}
}
