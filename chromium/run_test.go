package chromium

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"testing"
)

func TestRun(t *testing.T) {
	opts := buildOptions(ChromiumOptions{
		Url: "http://www.baidu.com",
	})

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	var chromeContext *chromedp.Context
	// log the protocol messages to understand how it works.
	// , chromedp.WithDebugf(log.Printf)
	ctx, cancel = chromedp.NewContext(ctx, func(c *chromedp.Context) {
		// 获取到了浏览器的上下文
		// 通过这个上下文可以操作浏览器
		fmt.Println("获取到了浏览器的上下文", c)
		chromeContext = c
	})
	defer cancel()

	defer func() {
		fmt.Println("关闭浏览器")
		// 关闭浏览器
		chromeContext.Browser.Process().Kill()
	}()

	// create a timeout
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	// navigate to a page, wait for an element, click
	err := chromedp.Run(ctx)

	if err != nil {
		log.Fatal(err)
	}

}
