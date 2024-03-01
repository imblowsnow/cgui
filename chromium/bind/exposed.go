package bind

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/imblowsnow/cgui/chromium/utils"
	"github.com/pterm/pterm"
	"strings"
)

// ExposedFunc is the function type that can be exposed to the browser env.
type ExposedFunc func(args string) (string, error)

// ExposeAction are actions which expose Go functions to the browser env.
type ExposeAction chromedp.Action

//go:embed all:script
var bindFiles embed.FS
var exposeJS string

func init() {
	file, err := bindFiles.ReadFile("script/bind.js")
	if err != nil {
		panic(err)
	}
	exposeJS = string(file)
}

// Expose is an action to add a function called fnName on the browser page's
// window object. When called, the function executes fn in the Go env and
// returns a Promise which resolves to the return value of fn.
//
// Note:
// 1. This is the lite version of puppeteer's [page.exposeFunction].
// 2. It adds "chromedpExposeFunc" to the page's window object too.
// 3. The exposed function survives page navigation until the tab is closed.
// 4. It exports the function to all frames on the current page.
// 5. Avoid exposing multiple funcs with the same name.
// 6. Maybe you just need runtime.AddBinding.
//
// [page.exposeFunction]: https://github.com/puppeteer/puppeteer/blob/v19.2.2/docs/api/puppeteer.page.exposefunction.md
const exposePrefix = "_cexposed"

func Expose(ctx context.Context, fnName string, fn ExposedFunc) error {
	extraName := exposePrefix + "_" + strings.ReplaceAll(fnName, ".", "_")
	expression := fmt.Sprintf(exposePrefix+`.wrapBinding("exposedFun","%s","%s");`, extraName, fnName)

	err := chromedp.Run(ctx,
		runtime.AddBinding(extraName),
		// 执行绑定的函数
		utils.EvaluateOnFrames(exposeJS),
		utils.EvaluateOnFrames(expression),
		// Make it effective after navigation.
		utils.AddScriptToEvaluateOnNewDocument(exposeJS),
		utils.AddScriptToEvaluateOnNewDocument(expression),
	)
	if err != nil {
		return err
	}
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventBindingCalled:
			if ev.Payload == "" {
				return
			}

			var payload struct {
				Type string `json:"type"`
				Seq  string `json:"seq"`
				Args string `json:"args"`
			}

			err := json.Unmarshal([]byte(ev.Payload), &payload)
			if err != nil {
				pterm.Error.Printf("failed to deliver result to exposed func %s: %s", fnName, err)
				return
			}

			if payload.Type != "exposedFun" || ev.Name != extraName {
				return
			}

			result, err := fn(payload.Args)

			pterm.Debug.Println("Expose function Exec", fnName, result)

			// 响应结果
			callback := exposePrefix + ".deliverResult"
			if err != nil {
				result = err.Error()
				callback = exposePrefix + ".deliverError"
			}

			// Prevent the message from being processed by other functions
			ev.Payload = ""

			go func() {
				err := chromedp.Run(ctx,
					chromedp.CallFunctionOn(callback,
						nil,
						func(p *runtime.CallFunctionOnParams) *runtime.CallFunctionOnParams {
							return p.WithExecutionContextID(ev.ExecutionContextID)
						},
						ev.Name,
						payload.Seq,
						result,
					),
				)

				if err != nil {
					pterm.Error.Println("Expose function Exec Error", fnName, err)
				}
			}()
		}
	})

	return nil
}
