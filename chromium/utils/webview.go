package utils

import (
	"context"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"time"
)

func AddScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return err
	})
}

func EvaluateOnFrames(script string) chromedp.Action {
	return chromedp.Evaluate(script, nil)
}

func GetFrameContext(ctx context.Context, frameID string) context.Context {
	var tgt *target.Info

	// 循环等待iframe加载完成
	for tgt == nil {
		targets, _ := chromedp.Targets(ctx)
		for _, t := range targets {
			if t.TargetID.String() == frameID {
				tgt = t
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	ictx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(tgt.TargetID))

	tempChromeCtx := chromedp.FromContext(ictx)
	if tempChromeCtx.Target == nil {
		_ = chromedp.Run(
			ictx,
		)
	}
	return ictx
}
func GetFrameExecutorContext(ctx context.Context, iframeID string) context.Context {
	frameCtx := GetFrameContext(ctx, iframeID)

	chromeCtx := chromedp.FromContext(frameCtx)

	executorCtx := cdp.WithExecutor(frameCtx, chromeCtx.Target)

	return executorCtx
}
