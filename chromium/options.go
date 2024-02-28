package chromium

import (
	"embed"
	"fmt"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/imblowsnow/cgui/chromium/front"
	"github.com/imblowsnow/cgui/chromium/handler"
	"github.com/imblowsnow/cgui/chromium/utils"
	"strconv"
)

type ChromiumOptions struct {
	Url string

	// user-data-dir
	UserDataDir string
	UserAgent   string

	FrontPrefix string

	// 自定义chrome参数
	ChromeOpts []chromedp.ExecAllocatorOption

	ChromePath string

	DevTools bool

	// 窗口大小 为空宽高自适应
	Width  int
	Height int
	// 窗口位置 为空居中
	X int
	Y int

	// 前端文件
	FrontFiles embed.FS

	// 是否随机指纹
	RandomFingerprint bool

	// 是否处理跨域
	CorsFilter func(*fetch.EventRequestPaused) bool

	//  event.Next() // 继续执行
	RequestHandlers []func(event *handler.FetchRequestEvent)

	//  event.Next() // 继续执行
	ResponseHandlers []func(event *handler.FetchRequestEvent)

	// 绑定方法
	Binds []interface{}
}

func (option *ChromiumOptions) buildOptions() ([]chromedp.ExecAllocatorOption, string, error) {
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
