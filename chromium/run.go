package chromium

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/imblowsnow/cgui/chromium/bind"
	"github.com/imblowsnow/cgui/chromium/event"
	"github.com/imblowsnow/cgui/chromium/handler"
	"github.com/imblowsnow/cgui/chromium/utils"
	"github.com/imblowsnow/cgui/chromium/utils/env"
	"github.com/leaanthony/slicer"
	"io/fs"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"
)

//go:embed all:script
var goFiles embed.FS

func runBrowser(option *ChromiumOptions) error {
	opts, url, err := option.buildOptions()
	if err != nil {
		return err
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
	err = chromedp.Run(ctx, fetchEnable)

	if err != nil {
		return err
	}

	// 注入JS
	addInjectScript(ctx, option)

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
func GetIframeContext(ctx context.Context, iframeID string) context.Context {
	var tgt *target.Info

	// 循环等待iframe加载完成
	for tgt == nil {
		targets, _ := chromedp.Targets(ctx)
		for _, t := range targets {
			if t.TargetID.String() == iframeID {
				tgt = t
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	ictx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(tgt.TargetID))

	tempChromeCtx := chromedp.FromContext(ictx)
	if tempChromeCtx.Target == nil {
		_ = chromedp.Run(
			ictx,
		)
	}
	return ictx
}
func GetIframeExecutorContext(ctx context.Context, iframeID string) context.Context {
	frameCtx := GetIframeContext(ctx, iframeID)

	chromeCtx := chromedp.FromContext(frameCtx)

	executorCtx := cdp.WithExecutor(frameCtx, chromeCtx.Target)

	return executorCtx
}
func addInjectScript(ctx context.Context, option *ChromiumOptions) {
	chromeCtx := chromedp.FromContext(ctx)
	executorCtx := cdp.WithExecutor(ctx, chromeCtx.Target)

	if option.RandomFingerprint {
		figerprintJs, _ := goFiles.ReadFile("script/inject/fingerprint.js")
		figerprintJsStr := string(figerprintJs)

		_, err := page.AddScriptToEvaluateOnNewDocument(figerprintJsStr).Do(executorCtx)
		if err != nil {
			fmt.Println("addInjectScript RandomFingerprint error", err.Error())
		}
	}

	//fmt.Println("注入 js to go 能力")
	if option.Binds != nil {
		// 生成绑定js
		bind.Bind(ctx, option.Binds)
	}

	var scriptFiles = slicer.String()
	scriptFiles.Add("script/inject/common.js")

	for _, scriptFile := range scriptFiles.AsSlice() {
		scriptBytes, _ := goFiles.ReadFile(scriptFile)
		scriptStr := string(scriptBytes)
		// 替换变量
		scriptStr = strings.ReplaceAll(scriptStr, "{mode}", env.Mode())

		err := chromedp.Run(ctx,
			utils.EvaluateOnFrames(scriptStr),
			// Make it effective after navigation.
			utils.AddScriptToEvaluateOnNewDocument(scriptStr),
		)
		if err != nil {
			fmt.Println("addInjectScript error", scriptFile, err.Error())
		}
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

	for _, requestHandler := range option.RequestHandlers {
		requestFetchHandler.Add(requestHandler)
	}

	var responseFetchHandler = handler.FetchHandler{}
	// cors 过滤
	responseFetchHandler.Add(corsFilter)
	for _, responseHandler := range option.ResponseHandlers {
		responseFetchHandler.Add(responseHandler)
	}

	var extraHeaderRequestMap = make(map[string]network.Headers)

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		chromeCtx := chromedp.FromContext(ctx)
		executorCtx := cdp.WithExecutor(ctx, chromeCtx.Target)
		// 获取事件的类型
		//eventType := reflect.TypeOf(ev).String()
		//pterm.Info.Println("listenTarget Event type:", eventType)
		// 防止阻塞
		go func() {
			// 请求监听器
			switch ev := ev.(type) {
			case *fetch.EventRequestPaused:
				//if ev.FrameID.String() != chromeCtx.Target.TargetID.String() {
				//	executorCtx = getIframeExecutorContext(ctx, ev.FrameID.String())
				//}
				var event *handler.FetchRequestEvent
				extraHeader := extraHeaderRequestMap[ev.NetworkID.String()]
				if extraHeader != nil {
					// 删除原来的数据
					delete(extraHeaderRequestMap, ev.NetworkID.String())
				}
				if ev.ResponseStatusCode > 0 {
					event = responseFetchHandler.Handle(ev, executorCtx, extraHeader)
				} else {
					event = requestFetchHandler.Handle(ev, executorCtx, extraHeader)
				}
				var err error

				if ev.ResponseStatusCode > 0 {
					if event.IsHandle() {
						err = fetch.FulfillRequest(ev.RequestID, ev.ResponseStatusCode).WithResponseHeaders(ev.ResponseHeaders).WithBody(base64.StdEncoding.EncodeToString(event.GetBody())).Do(executorCtx)
					} else {
						err = fetch.ContinueRequest(ev.RequestID).Do(executorCtx)
					}
				} else {
					if event.IsHandle() {
						var headers []*fetch.HeaderEntry
						for k, v := range ev.Request.Headers {
							headers = append(headers, &fetch.HeaderEntry{Name: k, Value: v.(string)})
						}
						err = fetch.ContinueRequest(ev.RequestID).WithHeaders(headers).Do(executorCtx)
					} else {
						err = fetch.ContinueRequest(ev.RequestID).Do(executorCtx)
					}
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
				// 创建了新的iframe
				if ev.FrameID.String() != chromeCtx.Target.TargetID.String() {
					// 获取iframe的上下文
					ictx := GetIframeContext(ctx, ev.FrameID.String())
					addInjectScript(ictx, option)
				}
			case *network.EventResponseReceivedExtraInfo:
				extraHeaderRequestMap[ev.RequestID.String()] = ev.Headers
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
