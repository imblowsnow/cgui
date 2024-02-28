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
	"github.com/google/uuid"
	"io/fs"
	"main/chromium/event"
	"main/chromium/front"
	"main/chromium/utils"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
)

//go:embed script/*.js
var goFiles embed.FS

func Run(option ChromiumOptions) error {
	// 如果是开发模式，需要启动开发模式
	if utils.IsDev() {
		fmt.Println("Run dev mode", os.Getenv("devUrl"))
		if os.Getenv("devUrl") != "" {
			option.Url = os.Getenv("devUrl")
		}
		if os.Getenv("assetdir") != "" {
			option.FrontPrefix = os.Getenv("assetdir")
		}
	}

	err := runBrowser(option)
	if err != nil {
		return err
	}

	return nil
}

func runBrowser(option ChromiumOptions) error {
	opts, url, error := buildOptions(option)
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
func injectTarget(ctx context.Context, option ChromiumOptions, frameID string) {
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
		figerprintJs, _ := goFiles.ReadFile("script/fingerprint.js")
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
		runtime.Evaluate(GenerateBindJs(option.Binds)).Do(executorCtx)
	}

	goJs := []byte{}
	if frameID == "" {
		goJs, _ = goFiles.ReadFile("script/initPage.js")
	} else {
		goJs, _ = goFiles.ReadFile("script/initFrame.js")
	}
	goJsStr := string(goJs)
	// 替换内容
	goJsStr = strings.ReplaceAll(goJsStr, "{mode}", utils.Mode())
	runtime.Evaluate(goJsStr).Do(executorCtx)
}
func listenTarget(ctx context.Context, option ChromiumOptions) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		chromeCtx := chromedp.FromContext(ctx)
		executorCtx := cdp.WithExecutor(ctx, chromeCtx.Target)

		// 获取事件的类型
		//eventType := reflect.TypeOf(ev).String()
		//fmt.Println("listenTarget Event type:", eventType)
		// 防止阻塞
		go func() {
			// 请求监听器
			switch ev := ev.(type) {
			// 注意，有2中事件，一种是请求，一种是响应通过 ev.ResponseStatusCode 区分
			case *fetch.EventRequestPaused:
				if strings.HasSuffix(ev.Request.URL, "/sub-jstogo") {
					OnRequestGoUrl(ev, executorCtx, option.Binds)
					return
				}

				if ev.ResponseStatusCode > 0 {
					// 如果处理跨域
					if option.CorsFilter != nil && option.CorsFilter(ev) {
						err := corsHandler(ev, executorCtx)
						if err == nil {
							return
						} else {
							fmt.Println("corsHandler error:", ev.Request.URL, err)
						}
					}
					// 响应拦截
					if option.OnResponseIntercept != nil {
						if option.OnResponseIntercept(ev, executorCtx) {
							return
						}
					}
				} else {
					// 请求拦截
					if option.OnRequestIntercept != nil {
						if option.OnRequestIntercept(ev, executorCtx) {
							return
						}
					}
				}
				fetch.ContinueRequest(ev.RequestID).Do(executorCtx)
			case *page.EventLifecycleEvent:
				if ev.FrameID.String() == chromeCtx.Target.TargetID.String() {
					event.OnPageLifecycleEvent(ctx, ev)
					if ev.Name == "init" {
						injectTarget(ctx, option, "")
					}
				} else {
					if ev.Name == "init" {
						injectTarget(ctx, option, ev.FrameID.String())
					}
					event.OnFrameLifecycleEvent(ctx, ev)
				}
			case *page.EventLoadEventFired:
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

func buildOptions(option ChromiumOptions) ([]chromedp.ExecAllocatorOption, string, error) {
	url := "https://www.baidu.com"
	if option.Url != "" {
		url = option.Url
		//} else if flag, _ := isEmbedFSEmpty(option.FrontFiles); flag {
		//	return nil, "", fmt.Errorf("前端文件为空，且未指定访问的url")
	} else {
		// 判断GO当前环境

		if option.FrontPrefix == "" {
			option.FrontPrefix = "frontend"
		}
		addr := front.RunEmbedFileServer(option.FrontFiles, option.FrontPrefix)
		url = "http://" + addr
	}
	fmt.Println("url:", url)
	var width, height int
	var centerX, centerY int
	if option.Width > 0 && option.Height > 0 {
		width = option.Width
		height = option.Height
	} else {
		width, height = utils.GetAutoWidthHeight()
	}
	if option.X > 0 && option.Y > 0 {
		centerX = option.X
		centerY = option.Y
	} else {
		centerX, centerY = utils.GetCenterPosition(width, height)
	}
	if option.ChromePath == "" {
		option.ChromePath = utils.FindExecPath()
	}
	if option.ChromePath == "" {
		return nil, "", fmt.Errorf("未安装google浏览器，请自行安装")
	}

	uuidStr := uuid.New().String()
	// 为了解决重复复用窗口的问题
	randomSite := fmt.Sprintf("file://%s", uuidStr)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("hide-scrollbars", false),
		chromedp.Flag("mute-audio", false),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("new-window", true),
		// 以应用模式显示浏览器
		chromedp.Flag("app", randomSite),
		chromedp.Flag("window-size", strconv.Itoa(width)+","+strconv.Itoa(height)),
		// 窗口居中  x,y
		chromedp.Flag("window-position", strconv.Itoa(centerX)+","+strconv.Itoa(centerY)),

		chromedp.ExecPath(option.ChromePath),
	)
	if option.UserAgent != "" {
		opts = append(opts, chromedp.UserAgent(option.UserAgent))
	}
	if option.UserDataDir != "" {
		opts = append(opts, chromedp.Flag("user-data-dir", option.UserDataDir))
	}
	if len(option.ChromeOpts) > 0 {
		opts = append(opts, option.ChromeOpts...)
	}

	return opts, url, nil
}

// IsEmbedFSEmpty checks if an embed.FS is empty.
func isEmbedFSEmpty(eFS embed.FS) (bool, error) {
	dirEntries, err := fs.ReadDir(eFS, "front/index.html")
	if err != nil {
		return false, err
	}
	return len(dirEntries) == 0, nil
}

func corsHandler(ev *fetch.EventRequestPaused, ctx context.Context) error {
	body, err := fetch.GetResponseBody(ev.RequestID).Do(ctx)
	if err != nil {
		return err
	}
	// 追加响应头
	for i, header := range ev.ResponseHeaders {
		// 删除原来的跨域数据
		if strings.ToLower(header.Name) == strings.ToLower("Access-Control-Allow-Origin") {
			ev.ResponseHeaders = append(ev.ResponseHeaders[:i], ev.ResponseHeaders[i+1:]...)
		}
	}
	ev.ResponseHeaders = append(ev.ResponseHeaders, &fetch.HeaderEntry{Name: "Access-Control-Allow-Origin", Value: "*"})
	err = fetch.FulfillRequest(ev.RequestID, ev.ResponseStatusCode).WithResponseHeaders(ev.ResponseHeaders).WithBody(base64.StdEncoding.EncodeToString(body)).Do(ctx)
	return err
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
