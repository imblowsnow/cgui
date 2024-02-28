package chromium

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/imblowsnow/cgui/chromium/bind"
	"github.com/imblowsnow/cgui/chromium/event"
	"github.com/imblowsnow/cgui/chromium/handler"
	"github.com/imblowsnow/cgui/chromium/utils/env"
	"github.com/leaanthony/slicer"
	"io/fs"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
)

//go:embed all:script
var goFiles embed.FS

func runBrowser(option *ChromiumOptions) error {
	opts, url, error := option.buildOptions()
	if error != nil {
		return error
	}

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// log the protocol messages to understand how it works.
	// , chromedp.WithDebugf(log.Printf)
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	// 监听请求和响应，支持拦截
	fetchEnable := fetch.Enable().WithPatterns([]*fetch.RequestPattern{{URLPattern: "*", RequestStage: "Response"}, {URLPattern: "*", RequestStage: "Request"}})
	// navigate to a page, wait for an element, click
	err := chromedp.Run(ctx, fetchEnable)

	if err != nil {
		return err
	}

	defer func() {
		chromedp.Cancel(ctx)
	}()

	listenTarget(ctx, option)

	chromedp.Run(ctx, chromedp.Navigate(url))

	if option.DevTools {
		// 打开开发者工具

	}

	listenClose(ctx)

	fmt.Println("chrome is closed")

	return nil
}
func getIframeContext(ctx context.Context, iframeID string) context.Context {
	targets, _ := chromedp.Targets(ctx)
	var tgt *target.Info
	for _, t := range targets {
		if t.TargetID.String() == iframeID {
			tgt = t
			break
		}
	}
	if tgt != nil {
		ictx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(tgt.TargetID))
		return ictx
	}
	return nil
}
func injectTarget(ctx context.Context, option *ChromiumOptions, frameID string) {
	if frameID != "" {
		var tempCtx context.Context

		for tempCtx == nil {
			tempCtx = getIframeContext(ctx, frameID)
			if tempCtx != nil {
				ctx = tempCtx
			}
		}
		fmt.Println("injectTarget frameID:", frameID)
		tempChromeCtx := chromedp.FromContext(ctx)
		if tempChromeCtx.Target == nil {
			_ = chromedp.Run(
				ctx, // <-- instead of ctx
				chromedp.Reload(),
			)
		}
	}
	chromeCtx := chromedp.FromContext(ctx)
	executorCtx := cdp.WithExecutor(ctx, chromeCtx.Target)
	if option.RandomFingerprint {
		figerprintJs, _ := goFiles.ReadFile("script/inject/fingerprint.js")
		figerprintJsStr := string(figerprintJs)
		_, e, err := runtime.Evaluate(figerprintJsStr).Do(executorCtx)
		if err != nil {
			fmt.Println("fingerprint error:", err)
			fmt.Println("fingerprint error:", e)
		}
	}

	fmt.Println("注入 js to go 能力")
	if option.Binds != nil {
		// 生成绑定js
		runtime.Evaluate(bind.GenerateBindJs(option.Binds)).Do(executorCtx)
	}

	var scriptFiles = slicer.String()
	scriptFiles.Add("script/inject/common.js")

	if frameID == "" {
		scriptFiles.Add("script/inject/page.js")
	} else {
		scriptFiles.Add("script/inject/frame.js")
	}

	for _, scriptFile := range scriptFiles.AsSlice() {
		scriptBytes, _ := goFiles.ReadFile(scriptFile)
		scriptStr := string(scriptBytes)
		// 替换变量
		scriptStr = strings.ReplaceAll(scriptStr, "{mode}", env.Mode())
		runtime.Evaluate(scriptStr).Do(executorCtx)
	}

	// 执行app的ready方法
	if option.App != nil && option.App.OnReady != nil {
		option.App.OnReady(executorCtx)
	}
}
func listenTarget(ctx context.Context, option *ChromiumOptions) {
	var corsFilter = func(event *handler.FetchRequestEvent) {
		if option.CorsFilter != nil && option.CorsFilter(event.Event) {
			handler.CorsHandler(event)
		}
		event.Next()
	}

	var requestFetchHandler = handler.FetchHandler{}

	requestFetchHandler.Add(handler.BindHandler)
	// cors 过滤
	requestFetchHandler.Add(corsFilter)
	for _, requestHandler := range option.RequestHandlers {
		requestFetchHandler.Add(requestHandler)
	}

	var responseFetchHandler = handler.FetchHandler{}
	// cors 过滤
	responseFetchHandler.Add(corsFilter)
	for _, responseHandler := range option.ResponseHandlers {
		responseFetchHandler.Add(responseHandler)
	}

	chromeCtx := chromedp.FromContext(ctx)
	executorCtx := cdp.WithExecutor(ctx, chromeCtx.Target)

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		// 获取事件的类型
		//eventType := reflect.TypeOf(ev).String()
		//pterm.Info.Println("listenTarget Event type:", eventType)
		// 防止阻塞
		go func() {
			// 请求监听器
			switch ev := ev.(type) {
			// 注意，有2中事件，一种是请求，一种是响应通过 ev.ResponseStatusCode 区分
			case *fetch.EventRequestPaused:
				var event *handler.FetchRequestEvent
				if ev.ResponseStatusCode > 0 {
					event = responseFetchHandler.Handle(ev, executorCtx)
				} else {
					event = requestFetchHandler.Handle(ev, executorCtx)
				}
				var err error
				if event.IsHandle() {
					err = fetch.FulfillRequest(ev.RequestID, ev.ResponseStatusCode).WithResponseHeaders(ev.ResponseHeaders).WithBody(base64.StdEncoding.EncodeToString(event.GetBody())).Do(executorCtx)
				} else {
					err = fetch.ContinueRequest(ev.RequestID).Do(executorCtx)
				}
				if err != nil {
					fmt.Println("fetch handle error:", ev.Request.URL, err)
				}
			case *page.EventLifecycleEvent:
				if ev.FrameID.String() == chromeCtx.Target.TargetID.String() {
					event.OnPageLifecycleEvent(ctx, ev)
					if option.App != nil && option.App.OnPageLifecycleEvent != nil {
						option.App.OnPageLifecycleEvent(executorCtx, ev)
					}
				} else {
					event.OnFrameLifecycleEvent(ctx, ev)
					if option.App != nil && option.App.OnFrameLifecycleEvent != nil {
						option.App.OnFrameLifecycleEvent(executorCtx, ev)
					}
				}
			case *page.EventFrameStartedLoading:
				if ev.FrameID.String() == chromeCtx.Target.TargetID.String() {
					injectTarget(ctx, option, "")
				} else {
					injectTarget(ctx, option, ev.FrameID.String())
				}
			}
		}()
	})
}
func listenClose(ctx context.Context) {
	// 制作信号量
	sem := make(chan struct{}, 1)

	chromedp.ListenBrowser(ctx, func(ev interface{}) {
		chromeCtx := chromedp.FromContext(ctx)
		// 获取事件的类型
		eventType := reflect.TypeOf(ev).String()
		fmt.Println("ListenBrowser Event type:", eventType)

		switch ev := ev.(type) {
		//case *target.EventTargetCreated:
		//	if option.OnCreatedBrowser != nil {
		//		option.OnCreatedBrowser(ctx)
		//	}
		case *target.EventTargetDestroyed:
			// 当前浏览器关闭事件
			if ev.TargetID == chromeCtx.Target.TargetID {
				sem <- struct{}{}
			}
			fmt.Println("TargetDestroyed:", ev.TargetID)
		case *target.EventDetachedFromTarget:
			// 当前浏览器关闭事件
			if ev.SessionID == chromeCtx.Target.SessionID {
				sem <- struct{}{}
			}
		}
	})

	go func() {
		exitHandle()

		sem <- struct{}{}
	}()

	// 等待信号量
	<-sem
}

// IsEmbedFSEmpty checks if an embed.FS is empty.
func isEmbedFSEmpty(eFS embed.FS) (bool, error) {
	dirEntries, err := fs.ReadDir(eFS, "front/index.html")
	if err != nil {
		return false, err
	}
	return len(dirEntries) == 0, nil
}

func exitHandle() {
	exitChan := make(chan os.Signal)
	signal.Notify(exitChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	for {
		select {
		case sig := <-exitChan:
			fmt.Println("接受到来自系统的信号：", sig)
			return
		}
	}

}
