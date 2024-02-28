//go:build production

package chromium

import (
	"fmt"
	build2 "github.com/imblowsnow/cgui/chromium/internal/build"
)

func Run(option *ChromiumOptions) error {
	// 如果是开发模式，需要启动开发模式
	if option.FrontPrefix == "" {
		rawBytes, err := option.FrontFiles.ReadFile("project.json")
		if err == nil {
			projectInfo, err := build2.Parse(rawBytes)
			if err != nil {
				return err
			}
			// 设置前端文件路径
			if projectInfo.AssetDirectory != "" {
				option.FrontPrefix = projectInfo.AssetDirectory
			} else {
				return fmt.Errorf("FrontPrefix 为空")
			}
		} else {
			return fmt.Errorf("FrontPrefix 为空 且 读取 项目配置 失败")
		}
	}

	err := runBrowser(option)
	if err != nil {
		return err
	}

	return nil
}
