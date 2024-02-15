package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/imblowsnow/chromedp"
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
	err := chromium.Run(chromium.ChromiumOptions{
		Url:         "https://www.xiaohongshu.com/explore",
		UserDataDir: userDataDir,
		FrontFiles:  frontFiles,
		ChromeOpts:  []chromedp.ExecAllocatorOption{},
		OnCreatedPage: func(ctx context.Context) {
			fmt.Println("页面创建完毕")
		},
		Binds: []interface{}{
			TestBindJs{},
		},
	})
	if err != nil {
		panic(err)
	}
}
