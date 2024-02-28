package main

import (
	"main/chromium/cmd/build"
	"main/chromium/cmd/build/flags"
	build2 "main/chromium/internal/build"
	"os"
	"path/filepath"
	"runtime"
)

func devApplication(dev *flags.Dev) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	buildDir := filepath.Join(currentDir, "build")

	// 加载项目信息
	projectInfo, err := build2.Load(currentDir)
	if err != nil {
		return err
	}
	// 构建环境参数
	options := &build2.Options{
		Mode:         build2.Dev,
		Platform:     runtime.GOOS,
		Arch:         runtime.GOARCH,
		BinDirectory: filepath.Join(buildDir, "bin"),
		Compiler:     "go",
		ProjectData:  projectInfo,
	}

	// TODO 生成 bindings

	if !options.IgnoreFrontend {
		// 编译前端资源文件
		frontBuilder := build.NewFrontBuilder(options)

		closer, devServerURL, _, err := frontBuilder.RunFrontend(true)
		if err != nil {
			return err
		}
		if devServerURL != "" {
			options.ProjectData.FrontendDevServerURL = devServerURL
		}

		defer closer()

	}

	// 调用go 启动应用
	builder := build.NewBuilder(options)
	err = builder.RunProject()
	if err != nil {
		return err
	}

	return nil
}
