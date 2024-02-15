package chromium

import (
	"context"
	"embed"
	"github.com/chromedp/cdproto/fetch"
	"github.com/imblowsnow/chromedp"
)

type ChromiumOptions struct {
	Url string

	// user-data-dir
	UserDataDir string

	FrontPrefix string

	// 自定义chrome参数
	ChromeOpts []chromedp.ExecAllocatorOption

	// 窗口大小
	Width  int
	Height int
	X      int
	Y      int

	// 前端文件
	FrontFiles embed.FS

	OnCreatedBrowser func(context.Context)
	OnCreatedPage    func(context.Context)
	//  false 不拦截，true 拦截，自己处理
	//  body, err := fetch.GetResponseBody(ev.RequestID).Do(executorCtx)
	//	if err != nil {
	//		fmt.Println("GetResponseBody error:", err)
	//		return
	//	}
	//	// 修改请求内容
	//	fmt.Println("RequestPaused body:", base64.StdEncoding.EncodeToString(body))
	//	// 传输回去需要base64编码
	//	fetch.FulfillRequest(ev.RequestID, ev.ResponseStatusCode).WithResponseHeaders(ev.ResponseHeaders).WithBody(base64.StdEncoding.EncodeToString(body)).Do(executorCtx)
	OnRequestIntercept  func(*fetch.EventRequestPaused, context.Context) bool
	OnResponseIntercept func(*fetch.EventRequestPaused, context.Context) bool

	// 绑定方法
	Binds []interface{}
}
