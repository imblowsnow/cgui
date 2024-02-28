package build

import (
	"fmt"
	"github.com/imblowsnow/cgui/chromium/cmd/build/fs"
	"github.com/imblowsnow/cgui/chromium/cmd/build/package"
	"github.com/imblowsnow/cgui/chromium/cmd/build/shell"
	build2 "github.com/imblowsnow/cgui/chromium/internal/build"
	"github.com/leaanthony/slicer"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Builder struct {
	options *build2.Options
}

func NewBuilder(options *build2.Options) *Builder {
	return &Builder{
		options: options,
	}
}

func (b *Builder) PackageProject(platform string) error {
	var err error
	switch platform {
	case "darwin":
		err = _package.PackageApplicationForDarwin(b.options)
	case "windows":
		err = _package.PackageApplicationForWindows(b.options)
	case "linux":
		// linux 直接打包就可以了，不需要编译应用信息
		//err = packageApplicationForLinux(options)
	default:
		err = fmt.Errorf("packing not supported for %s yet", platform)
	}

	if err != nil {
		return err
	}

	return nil
}

func (b *Builder) CompileProject() error {
	options := b.options
	// Run go mod tidy first
	if !options.SkipModTidy {
		cmd := exec.Command(options.Compiler, "mod", "tidy")
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	compiler := options.Compiler

	commands := slicer.String()

	// Default go build command
	commands.Add("build")

	// Add better debugging flags
	if options.Mode == build2.Dev || options.Mode == build2.Debug {
		commands.Add("-gcflags")
		commands.Add("all=-N -l")
	}

	if options.ForceBuild {
		commands.Add("-a")
	}

	if options.TrimPath {
		commands.Add("-trimpath")
	}

	if options.RaceDetector {
		commands.Add("-race")
	}

	var tags slicer.StringSlicer
	if options.Mode == build2.Production || options.Mode == build2.Debug {
		tags.Add("production")
	}
	if options.Mode == build2.Debug {
		tags.Add("debug")
	}
	if options.Devtools {
		tags.Add("devtools")
	}
	tags.Deduplicate()
	commands.Add("-tags")
	commands.Add(tags.Join(","))

	ldflags := slicer.String()

	// 删除控制台
	if options.Mode == build2.Production {
		ldflags.Add("-w", "-s")
		if options.Platform == "windows" && !options.WindowsConsole {
			ldflags.Add("-H windowsgui")
		}
	}
	ldflags.Deduplicate()

	if ldflags.Length() > 0 {
		commands.Add("-ldflags")
		commands.Add(ldflags.Join(" "))
	}

	// Get application build directory
	appDir := options.BinDirectory
	if options.CleanBinDirectory {
		err := b.cleanBinDirectory()
		if err != nil {
			return err
		}
	}

	// Set up output filename
	outputFile := options.ProjectData.OutputFilename
	compiledBinary := filepath.Join(appDir, outputFile)
	commands.Add("-o")
	commands.Add(compiledBinary)

	options.CompiledBinary = compiledBinary

	if options.OutPutCompileCmd {
		// 输出编译命令
		pterm.Println("Compiling with: ", compiler+" "+commands.Join(" "))
		return nil
	}

	// Build the application
	cmd := exec.Command(compiler, commands.AsSlice()...)
	cmd.Stderr = os.Stderr

	// Set the directory
	cmd.Dir = options.ProjectData.Path

	cmd.Env = os.Environ() // inherit env

	// Run command
	err := cmd.Run()
	cmd.Stderr = os.Stderr

	// Format error if we have one
	if err != nil {
		if options.Platform == "darwin" {
			output, _ := cmd.CombinedOutput()
			stdErr := string(output)
			if strings.Contains(err.Error(), "ld: framework not found UniformTypeIdentifiers") ||
				strings.Contains(stdErr, "ld: framework not found UniformTypeIdentifiers") {
				pterm.Warning.Println(`
NOTE: It would appear that you do not have the latest Xcode cli tools installed.
Please reinstall by doing the following:
  1. Remove the current installation located at "xcode-select -p", EG: sudo rm -rf /Library/Developer/CommandLineTools
  2. Install latest Xcode tools: xcode-select --install`)
			}
		}
		return err
	}

	// 下面开始压缩
	if !options.Compress {
		return nil
	}

	// Do we have upx installed?
	if !shell.CommandExists("upx") {
		pterm.Warning.Println("Warning: Cannot compress binary: upx not found")
		return nil
	}

	args := []string{"--best", "--no-color", "--no-progress", options.CompiledBinary}

	if options.CompressFlags != "" {
		args = strings.Split(options.CompressFlags, " ")
		args = append(args, options.CompiledBinary)
	}

	output, err := exec.Command("upx", args...).Output()
	if err != nil {
		return errors.Wrap(err, "Error during compression:")
	}
	pterm.Println("Done.", output)
	return nil
}

func (b *Builder) RunProject() error {
	// Run go mod tidy to ensure we're up-to-date
	_, _, err := shell.RunCommand(b.options.ProjectData.Path, b.options.Compiler, "mod", "tidy")
	if err != nil {
		return err
	}
	// TODO Watch for changes and trigger restartApp()

	pterm.Info.Println("Run application for development...")

	commands := slicer.String()
	commands.Add("run")
	commands.Add(".")

	var tags slicer.StringSlicer
	commands.Add("-tags")
	tags.Add("dev")

	commands.Add(tags.Join(","))

	commands.Add("-gcflags")
	commands.Add("all=-N -l")

	cmd := exec.Command(b.options.Compiler, commands.AsSlice()...)

	cmd.Dir = b.options.ProjectData.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 获取当前的环境变量
	env := os.Environ()
	// 添加新的环境变量

	newEnvs := slicer.String()
	newEnvs.Add("mode=dev")
	newEnvs.Add("devUrl=" + b.options.ProjectData.FrontendDevServerURL)
	newEnvs.Add("assetdir=" + b.options.ProjectData.AssetDirectory)

	pterm.Info.Println("env:", strings.Join(newEnvs.AsSlice(), ";"))

	env = append(env, newEnvs.AsSlice()...)

	cmd.Env = env

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (b *Builder) cleanBinDirectory() error {
	buildDirectory := b.options.BinDirectory

	// Clear out old builds
	if fs.DirExists(buildDirectory) {
		err := os.RemoveAll(buildDirectory)
		if err != nil {
			return err
		}
	}

	// Create clean directory
	err := os.MkdirAll(buildDirectory, 0o700)
	if err != nil {
		return err
	}

	return nil
}
