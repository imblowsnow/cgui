package chromium

import (
	"context"
	"github.com/chromedp/cdproto/page"
)

type App struct {
	OnReady func(ctx context.Context)

	OnPageLifecycleEvent func(ctx context.Context, ev *page.EventLifecycleEvent)

	OnFrameLifecycleEvent func(ctx context.Context, ev *page.EventLifecycleEvent)
}
