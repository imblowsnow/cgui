package main

import (
	"github.com/pterm/pterm"
	"main/chromium/cmd/build"
	"main/chromium/cmd/build/flags"
	build2 "main/chromium/internal/build"
	"os"
	"path/filepath"
	"runtime"
)

func buildApplication(f *flags.Build) error {
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
		Mode:         build2.Production,
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
		err = frontBuilder.BuildFrontend(true)
		if err != nil {
			return err
		}
	}

	builder := build.NewBuilder(options)
	// 忽略应用程序
	if !options.IgnoreApplication {
		pterm.Info.Print("Building application: ")

		err = builder.PackageProject(runtime.GOOS)
		if err != nil {
			return err
		}

		pterm.Println("Done.")

		// 删除 syso 文件
		if options.Platform == "windows" {
			defer func() {
				err := os.Remove(filepath.Join(options.ProjectData.Path, options.ProjectData.Name+"-res.syso"))
				if err != nil {
					fatal(err.Error())
				}
			}()
		}

		pterm.Info.Print("Compile application: ")

		// 调用 go 编译
		err = builder.CompileProject()

		if err != nil {
			return err
		}

		pterm.Println("Done.")
	}

	return nil
}

func fatal(message string) {
	printer := pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.FatalMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.ThemeDefault.FatalPrefixStyle,
			Text:  " FATAL ",
		},
	}
	printer.Println(message)
	os.Exit(1)
}
