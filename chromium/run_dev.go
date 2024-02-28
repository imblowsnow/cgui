//go:build !production

package chromium

import (
	"fmt"
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

	err := runBrowser(option)
	if err != nil {
		return err
	}

	return nil
}
