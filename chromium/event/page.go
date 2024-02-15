package event

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/page"
)

func OnPageLifecycleEvent(ctx context.Context, ev *page.EventLifecycleEvent) {
	fmt.Println("OnPageChange:", ev.Name)
}
