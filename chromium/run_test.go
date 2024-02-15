package chromium

import (
	"context"
	"github.com/imblowsnow/chromedp"
	"log"
	"testing"
)

func TestRun(t *testing.T) {
	opts := buildOptions(ChromiumOptions{
		Url:         "http://www.google.com",
		UserDataDir: GetCurrentBrowserFlagDir("default"),
	})

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// log the protocol messages to understand how it works.
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	// navigate to a page, wait for an element, click
	err := chromedp.Run(ctx)

	if err != nil {
		log.Fatal(err)
	}

	err2 := chromedp.Run(ctx)

	if err2 != nil {
		log.Fatal(err2)
	}

	select {}
}
