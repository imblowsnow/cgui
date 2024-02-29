package utils

import (
	"context"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"sync"
)

var pagesMu sync.RWMutex

func AddScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return err
	})
}

//func EvaluateOnAllFrames(script string) chromedp.Action {
//	return chromedp.ActionFunc(func(ctx context.Context) error {
//		c := chromedp.FromContext(ctx)
//
//		c.Target.frameMu.RLock()
//		actions := make([]chromedp.Action, 0, len(c.Target.execContexts))
//		for _, executionContextID := range c.Target.execContexts {
//			id := executionContextID
//			actions = append(actions, chromedp.Evaluate(script, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
//				return p.WithContextID(id)
//			}))
//		}
//		c.Target.frameMu.RUnlock()
//
//		return chromedp.Tasks(actions).Do(ctx)
//	})
//}
//
//func GetFrameExecutionContextID(frameId string) {
//
//}
