package main

import (
	"embed"
	"fmt"
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
	err := chromium.Run(chromium.ChromiumOptions{
		Url:               "https://www.yalala.com/",
		UserDataDir:       userDataDir,
		FrontFiles:        frontFiles,
		ChromeOpts:        []chromedp.ExecAllocatorOption{},
		RandomFingerprint: true,
		Binds: []interface{}{
			TestBindJs{},
		},
	})
	if err != nil {
		dialog.Error(err.Error())
	}
}
