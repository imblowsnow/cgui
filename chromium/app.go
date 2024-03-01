package chromium

import (
	"context"
	"github.com/chromedp/cdproto/page"
)

type App struct {
	OnPageLifecycleEvent func(ctx context.Context, ev *page.EventLifecycleEvent)

	OnFrameLifecycleEvent func(ctx context.Context, ev *page.EventLifecycleEvent)
}
