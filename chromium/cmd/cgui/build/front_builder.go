package build

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/imblowsnow/cgui/chromium/cmd/cgui/build/buildassets"
	"github.com/imblowsnow/cgui/chromium/cmd/cgui/build/dev"
	"github.com/imblowsnow/cgui/chromium/cmd/cgui/build/fs"
	"github.com/imblowsnow/cgui/chromium/cmd/cgui/build/shell"
	build2 "github.com/imblowsnow/cgui/chromium/internal/build"
	"github.com/leaanthony/winicon"
	"github.com/pterm/pterm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type FrontBuilder struct {
	options *build2.Options
}

func NewFrontBuilder(options *build2.Options) *FrontBuilder {
	return &FrontBuilder{
		options: options,
	}
}

// NpmInstall runs "npm install" in the given directory
func (b *FrontBuilder) NpmInstall(sourceDir string, verbose bool) error {
	return b.NpmInstallUsingCommand(sourceDir, "npm install", verbose)
}

// NpmInstallUsingCommand runs the given install command in the specified npm project directory
func (b *FrontBuilder) NpmInstallUsingCommand(sourceDir string, installCommand string, verbose bool) error {
	packageJSON := filepath.Join(sourceDir, "package.json")

	// Check package.json exists
	if !fs.FileExists(packageJSON) {
		// No package.json, no install
		return nil
	}

	install := false

	// Get the MD5 sum of package.json
	packageJSONMD5 := fs.MustMD5File(packageJSON)

	// Check whether we need to npm install
	packageChecksumFile := filepath.Join(sourceDir, "package.json.md5")
	if fs.FileExists(packageChecksumFile) {
		// Compare checksums
		storedChecksum := fs.MustLoadString(packageChecksumFile)
		if storedChecksum != packageJSONMD5 {
			fs.MustWriteString(packageChecksumFile, packageJSONMD5)
			install = true
		}
	} else {
		install = true
		fs.MustWriteString(packageChecksumFile, packageJSONMD5)
	}

	// Install if node_modules doesn't exist
	nodeModulesDir := filepath.Join(sourceDir, "node_modules")
	if !fs.DirExists(nodeModulesDir) {
		install = true
	}

	// check if forced install
	if b.options.ForceBuild {
		install = true
	}

	// Shortcut installation
	if !install {
		if verbose {
			pterm.Println("Skipping npm install")
		}
		return nil
	}

	// Split up the InstallCommand and execute it
	cmd := strings.Split(installCommand, " ")
	stdout, stderr, err := shell.RunCommand(sourceDir, cmd[0], cmd[1:]...)
	if verbose || err != nil {
		for _, l := range strings.Split(stdout, "\n") {
			pterm.Printf("    %s\n", l)
		}
		for _, l := range strings.Split(stderr, "\n") {
			pterm.Printf("    %s\n", l)
		}
	}

	return err
}

// NpmRun executes the npm target in the provided directory
func (b *FrontBuilder) NpmRun(projectDir, buildTarget string, verbose bool) error {
	stdout, stderr, err := shell.RunCommand(projectDir, "npm", "run", buildTarget)
	if verbose || err != nil {
		for _, l := range strings.Split(stdout, "\n") {
			pterm.Printf("    %s\n", l)
		}
		for _, l := range strings.Split(stderr, "\n") {
			pterm.Printf("    %s\n", l)
		}
	}
	return err
}

// NpmRunWithEnvironment executes the npm target in the provided directory, with the given environment variables
func (b *FrontBuilder) NpmRunWithEnvironment(projectDir, buildTarget string, verbose bool, envvars []string) error {
	cmd := shell.CreateCommand(projectDir, "npm", "run", buildTarget)
	cmd.Env = append(os.Environ(), envvars...)
	var stdo, stde bytes.Buffer
	cmd.Stdout = &stdo
	cmd.Stderr = &stde
	err := cmd.Run()
	if verbose || err != nil {
		for _, l := range strings.Split(stdo.String(), "\n") {
			pterm.Printf("    %s\n", l)
		}
		for _, l := range strings.Split(stde.String(), "\n") {
			pterm.Printf("    %s\n", l)
		}
	}
	return err
}

// BuildFrontend executes the `npm build` command for the frontend directory
func (b *FrontBuilder) BuildFrontend(verbos bool) error {
	frontendDir := b.options.ProjectData.FrontendDir
	if !fs.DirExists(frontendDir) {
		return fmt.Errorf("frontend directory '%s' does not exist", frontendDir)
	}

	// Check there is an 'InstallCommand' provided in wails.json
	installCommand := b.options.ProjectData.InstallCommand
	if b.options.Mode == build2.Dev {
		installCommand = b.options.ProjectData.GetDevInstallerCommand()
	}
	if installCommand == "" {
		// No - don't install
		pterm.Println("No Install command. Skipping.")
		pterm.Println("")
	} else {
		// Do install if needed
		pterm.Info.Print("Installing frontend dependencies: ")
		if err := b.NpmInstallUsingCommand(frontendDir, installCommand, verbos); err != nil {
			return err
		}
		pterm.Println("Done.")
	}

	// Check if there is a build command
	buildCommand := b.options.ProjectData.BuildCommand
	if b.options.Mode == build2.Dev {
		buildCommand = b.options.ProjectData.GetDevBuildCommand()
	}
	if buildCommand == "" {
		pterm.Println("No Build command. Skipping.")
		pterm.Println("")
		// No - ignore
		return nil
	}

	pterm.Info.Print("Compiling frontend: ")
	cmd := strings.Split(buildCommand, " ")

	if verbos {
		pterm.Println("")
		pterm.Println("Build command: '" + buildCommand + "'")
	}

	stdout, stderr, err := shell.RunCommand(frontendDir, cmd[0], cmd[1:]...)
	if err != nil {
		for _, l := range strings.Split(stdout, "\n") {
			pterm.Printf("    %s\n", l)
		}
		for _, l := range strings.Split(stderr, "\n") {
			pterm.Printf("    %s\n", l)
		}
	}
	if err != nil {
		return err
	}

	pterm.Println("Done.")
	return nil
}

func (b *FrontBuilder) RunFrontend(verbos bool) (func(), string, string, error) {
	frontendDir := b.options.ProjectData.FrontendDir
	if !fs.DirExists(frontendDir) {
		return nil, "", "", fmt.Errorf("frontend directory '%s' does not exist", frontendDir)
	}

	// Check there is an 'InstallCommand' provided in wails.json
	installCommand := b.options.ProjectData.GetDevInstallerCommand()

	if installCommand == "" {
		// No - don't install
		pterm.Println("No Install command. Skipping.")
		pterm.Println("")
	} else {
		// Do install if needed
		pterm.Info.Print("Installing frontend dependencies: ")
		if err := b.NpmInstallUsingCommand(frontendDir, installCommand, verbos); err != nil {
			return nil, "", "", err
		}
		pterm.Println("Done.")
	}

	// Check if there is a build command
	buildCommand := b.options.ProjectData.GetDevBuildCommand()

	pterm.Info.Print("Compiling frontend: ")
	cmd := strings.Split(buildCommand, " ")

	if buildCommand == "" {
		return nil, "", "", nil
	}

	if verbos {
		pterm.Println("")
		pterm.Println("Build command: '" + buildCommand + "'")
	}

	return runFrontendDevWatcherCommand(frontendDir, strings.Join(cmd, " "), b.options.ProjectData.IsFrontendDevServerURLAutoDiscovery())
}

func (b *FrontBuilder) GenerateFrontIco() error {
	options := b.options
	iconName := "appicon"
	content, err := buildassets.ReadFile(options.ProjectData, iconName+".png")
	if err != nil {
		return err
	}

	// Check ico file exists already
	icoFile := filepath.Clean(filepath.Join(options.ProjectData.Path, options.ProjectData.AssetDirectory, filepath.FromSlash("favicon.ico")))
	if !fs.FileExists(icoFile) {
		if dir := filepath.Dir(icoFile); !fs.DirExists(dir) {
			if err := fs.MkDirs(dir, 0o755); err != nil {
				return err
			}
		}

		output, err := os.OpenFile(icoFile, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer output.Close()

		err = winicon.GenerateIcon(bytes.NewBuffer(content), output, []int{256, 128, 64, 48, 32, 16})
		if err != nil {
			return err
		}
	}
	return nil
}

func runFrontendDevWatcherCommand(frontendDirectory string, devCommand string, discoverViteServerURL bool) (func(), string, string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	scanner := dev.NewStdoutScanner()
	cmdSlice := strings.Split(devCommand, " ")
	cmd := exec.CommandContext(ctx, cmdSlice[0], cmdSlice[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = scanner
	cmd.Dir = frontendDirectory
	//setParentGID(cmd)

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, "", "", fmt.Errorf("unable to start frontend DevWatcher: %w", err)
	}

	var viteServerURL string
	if discoverViteServerURL {
		select {
		case serverURL := <-scanner.ViteServerURLChan:
			viteServerURL = serverURL
		case <-time.After(time.Second * 10):
			cancel()
			return nil, "", "", errors.New("failed to find Vite server URL")
		}
	}

	viteVersion := ""
	select {
	case version := <-scanner.ViteServerVersionC:
		viteVersion = version

	case <-time.After(time.Second * 5):
		// That's fine, then most probably it was not vite that was running
	}

	fmt.Println("Running frontend DevWatcher command: '%s'", devCommand)
	var wg sync.WaitGroup
	wg.Add(1)

	const (
		stateRunning   int32 = 0
		stateCanceling int32 = 1
		stateStopped   int32 = 2
	)
	state := stateRunning
	go func() {
		if err := cmd.Wait(); err != nil {
			wasRunning := atomic.CompareAndSwapInt32(&state, stateRunning, stateStopped)
			if err.Error() != "exit status 1" && wasRunning {
				fmt.Println("Error from DevWatcher '%s': %s", devCommand, err.Error())
			}
		}
		atomic.StoreInt32(&state, stateStopped)
		wg.Done()
	}()

	return func() {
		if atomic.CompareAndSwapInt32(&state, stateRunning, stateCanceling) {
			cmd.Process.Kill()
		}
		cancel()
		wg.Done()
	}, viteServerURL, viteVersion, nil
}
