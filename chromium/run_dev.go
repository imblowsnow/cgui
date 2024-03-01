//go:build !production && !generate

package chromium

import (
	"fmt"
	"github.com/imblowsnow/cgui/chromium/bind"
	"os"
)

func Run(option *ChromiumOptions) error {
	fmt.Println("Run dev mode", os.Getenv("devUrl"))
	if os.Getenv("devUrl") != "" {
		option.Url = os.Getenv("devUrl")
	}
	if os.Getenv("assetdir") != "" {
		option.FrontPrefix = os.Getenv("assetdir")
	}
	// 生成绑定文件
	if os.Getenv("bindjsdir") != "" {
		go func() {
			binds := option.Binds
			// 生成runtime绑定
			binds = append(binds, bind.GetRuntimeBinds()...)
			// 生成自定义绑定
			bind.Generate(os.Getenv("bindjsdir"), binds)
		}()
	}

	err := runBrowser(option)
	if err != nil {
		return err
	}

	return nil
}
