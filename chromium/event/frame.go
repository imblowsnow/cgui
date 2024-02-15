package event

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/page"
)

func OnFrameLifecycleEvent(ctx context.Context, ev *page.EventLifecycleEvent) {
	fmt.Println("OnFrameChange:", ev.Name, ev.FrameID)
}
