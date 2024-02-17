package chromium

import (
	"context"
	"embed"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/imblowsnow/chromedp"
	"io/fs"
	"main/chromium/event"
	"os"
	"reflect"
	"strconv"
	"strings"
)

//go:embed script/*.js
var goFiles embed.FS

func Run(option ChromiumOptions) error {
	if !CheckChromium() {
		// 未安装google浏览器，
		return fmt.Errorf("未安装google浏览器")
	}

	// 检测是否是多开
	//if option.UserDataDir != "" {
	//	devtoolsFile, _ := os.ReadFile(filepath.Join(option.UserDataDir, "DevToolsActivePort"))
	//	if devtoolsFile != nil {
	//		components := strings.Split(string(devtoolsFile), "\n")
	//		// 通过端口号连接，是否启动
	//		port, _ := strconv.Atoi(components[0])
	//		if !CheckPortAvailability("127.0.0.1", port) {
	//			wsurl := fmt.Sprintf("ws://127.0.0.1:%s%s", components[0], components[1])
	//			err := RunRemoteBrowser(wsurl, option)
	//			return err
	//		}
	//	}
	//}

	err := RunBrowser(option)
	if err != nil {
		return err
	}

	return nil
}

func RunBrowser(option ChromiumOptions) error {
	opts, url := buildOptions(option)

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
	goJsStr = strings.ReplaceAll(goJsStr, "{mode}", os.Getenv("APP_MODE"))
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

	// 等待信号量
	<-sem
}

func buildOptions(option ChromiumOptions) ([]chromedp.ExecAllocatorOption, string) {
	url := "https://www.baidu.com"
	if option.Url != "" {
		url = option.Url
	} else if flag, _ := isEmbedFSEmpty(option.FrontFiles); flag {
		panic("前端文件为空，且未指定url")
	} else {
		if option.FrontPrefix == "" {
			option.FrontPrefix = "front"
		}
		addr := RunFileServer(option.FrontFiles, option.FrontPrefix)
		url = "http://" + addr
	}
	fmt.Println("url:", url)
	var width, height int
	var centerX, centerY int
	if option.Width > 0 && option.Height > 0 {
		width = option.Width
		height = option.Height
	} else {
		width, height = getAutoWidthHeight()
	}
	if option.X > 0 && option.Y > 0 {
		centerX = option.X
		centerY = option.Y
	} else {
		centerX, centerY = getCenterPosition(width, height)
	}
	if option.ChromePath == "" {
		option.ChromePath = findExecPath()
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("hide-scrollbars", false),
		chromedp.Flag("mute-audio", false),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("new-window", true),
		// 以应用模式显示浏览器
		chromedp.Flag("app", url),
		chromedp.Flag("window-size", strconv.Itoa(width)+","+strconv.Itoa(height)),
		// 窗口居中  x,y
		chromedp.Flag("window-position", strconv.Itoa(centerX)+","+strconv.Itoa(centerY)),

		chromedp.ExecPath(option.ChromePath),
	)
	if option.UserDataDir != "" {
		opts = append(opts, chromedp.Flag("user-data-dir", option.UserDataDir))
	}
	if len(option.ChromeOpts) > 0 {
		opts = append(opts, option.ChromeOpts...)
	}

	return opts, url
}

// IsEmbedFSEmpty checks if an embed.FS is empty.
func isEmbedFSEmpty(eFS embed.FS) (bool, error) {
	dirEntries, err := fs.ReadDir(eFS, "front/index.html")
	if err != nil {
		return false, err
	}
	return len(dirEntries) == 0, nil
}
