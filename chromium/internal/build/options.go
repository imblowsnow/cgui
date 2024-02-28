package build

import (
	"encoding/json"
	"github.com/samber/lo"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Mode is the type used to indicate the build modes
type Mode int

const (
	// Dev mode
	Dev Mode = iota
	// Production mode
	Production
	// Debug build
	Debug
)

type FileAssociation struct {
	Ext         string `json:"ext"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconName    string `json:"iconName"`
	Role        string `json:"role"`
}

type Protocol struct {
	Scheme      string `json:"scheme"`
	Description string `json:"description"`
	Role        string `json:"role"`
}

type Info struct {
	CompanyName      string            `json:"companyName"`
	ProductName      string            `json:"productName"`
	ProductVersion   string            `json:"productVersion"`
	Copyright        *string           `json:"copyright"`
	Comments         *string           `json:"comments"`
	FileAssociations []FileAssociation `json:"fileAssociations"`
	Protocols        []Protocol        `json:"protocols"`
}
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Project holds the data related to a Wails project
type Project struct {
	/*** Application Data ***/
	Name           string `json:"name"`
	AssetDirectory string `json:"assetdir,omitempty"`

	ReloadDirectories string `json:"reloaddirs,omitempty"`

	BuildCommand   string `json:"frontend:build"`
	InstallCommand string `json:"frontend:install"`

	// Commands used in `wails dev`
	DevCommand        string `json:"frontend:dev"`
	DevBuildCommand   string `json:"frontend:dev:build"`
	DevInstallCommand string `json:"frontend:dev:install"`
	DevWatcherCommand string `json:"frontend:dev:watcher"`
	// The url of the external wails dev server. If this is set, this server is used for the frontend. Default ""
	FrontendDevServerURL string `json:"frontend:dev:serverUrl"`

	// Directory to generate the API Module
	WailsJSDir string `json:"wailsjsdir"`

	Version string `json:"version"`

	/*** Internal Data ***/

	// The path to the project directory
	Path string `json:"projectdir"`

	// Build directory
	BuildDir string `json:"build:dir"`

	// The output filename
	OutputFilename string `json:"outputfilename"`

	// The platform to target
	Platform string

	// RunNonNativeBuildHooks will run build hooks though they are defined for a GOOS which is not equal to the host os
	RunNonNativeBuildHooks bool `json:"runNonNativeBuildHooks"`

	// Build hooks for different targets, the hooks are executed in the following order
	// Key: GOOS/GOARCH - Executed at build level before/after a build of the specific platform and arch
	// Key: GOOS/*      - Executed at build level before/after a build of the specific platform
	// Key: */*         - Executed at build level before/after a build
	// The following keys are not yet supported.
	// Key: GOOS        - Executed at platform level before/after all builds of the specific platform
	// Key: *           - Executed at platform level before/after all builds of a platform
	// Key: [empty]     - Executed at global level before/after all builds of all platforms
	PostBuildHooks map[string]string `json:"postBuildHooks"`
	PreBuildHooks  map[string]string `json:"preBuildHooks"`

	// The application author
	Author Author

	// The application information
	Info Info

	// Fully qualified filename
	filename string

	// NSISType to be build
	NSISType string `json:"nsisType"`

	// Frontend directory
	FrontendDir string `json:"frontend:dir"`
}

func (p *Project) setDefaults() {
	if p.Name == "" {
		p.Name = "build"
	}
	if p.Path == "" {
		p.Path = lo.Must(os.Getwd())
	}
	if p.Version == "" {
		p.Version = "2"
	}
	if p.OutputFilename == "" {
		p.OutputFilename = p.Name
	}
	if p.FrontendDir == "" {
		p.FrontendDir = "frontend"
	}
	if p.WailsJSDir == "" {
		p.WailsJSDir = p.FrontendDir
	}
	if p.BuildDir == "" {
		p.BuildDir = "build"
	}
	if p.Info.CompanyName == "" {
		p.Info.CompanyName = p.Name
	}
	if p.Info.ProductName == "" {
		p.Info.ProductName = p.Name
	}
	if p.Info.ProductVersion == "" {
		p.Info.ProductVersion = "1.0.0"
	}
	if p.Info.Copyright == nil {
		v := "Copyright........."
		p.Info.Copyright = &v
	}
	if p.Info.Comments == nil {
		v := "Built using ChromeGui"
		p.Info.Comments = &v
	}
	// Fix up OutputFilename
	switch runtime.GOOS {
	case "windows":
		if !strings.HasSuffix(p.OutputFilename, ".exe") {
			p.OutputFilename += ".exe"
		}
	case "darwin", "linux":
		p.OutputFilename = strings.TrimSuffix(p.OutputFilename, ".exe")
	}
}

func (p *Project) GetDevBuildCommand() string {
	if p.DevBuildCommand != "" {
		return p.DevBuildCommand
	}
	if p.DevCommand != "" {
		return p.DevCommand
	}
	return p.BuildCommand
}
func (p *Project) GetDevInstallerCommand() string {
	if p.DevInstallCommand != "" {
		return p.DevInstallCommand
	}
	return p.InstallCommand
}
func (p *Project) IsFrontendDevServerURLAutoDiscovery() bool {
	return p.FrontendDevServerURL == "auto" || p.FrontendDevServerURL == ""
}

// Parse the given JSON data into a Project struct
func Parse(projectData []byte) (*Project, error) {
	project := &Project{}
	err := json.Unmarshal(projectData, project)
	if err != nil {
		return nil, err
	}
	project.setDefaults()
	return project, nil
}
func Load(projectPath string) (*Project, error) {
	projectFile := filepath.Join(projectPath, "project.json")
	rawBytes, err := os.ReadFile(projectFile)
	if err != nil {
		return nil, err
	}
	result, err := Parse(rawBytes)
	if err != nil {
		return nil, err
	}
	result.filename = projectFile
	return result, nil
}

func (p *Project) GetBuildDir() string {
	if filepath.IsAbs(p.BuildDir) {
		return p.BuildDir
	}
	return filepath.Join(p.Path, p.BuildDir)
}

// Options contains all the build options as well as the project data
type Options struct {
	Mode        Mode     // release or dev
	Devtools    bool     // Enable devtools in production
	ProjectData *Project // The project data

	Platform string // The platform to build for
	Arch     string // The architecture to build for

	Compiler          string // The compiler command to use
	SkipModTidy       bool   //  Skip mod tidy before compile
	IgnoreFrontend    bool   // Indicates if the frontend does not need building
	IgnoreApplication bool   // Indicates if the application does not need building
	OutPutCompileCmd  bool   // 只输出编译命令

	BinDirectory      string // Directory to use to write the built applications
	CleanBinDirectory bool   // Indicates if the bin output directory should be cleaned before building

	KeepAssets bool // Keep the generated assets/files

	Compress       bool   // Compress the final binary
	CompressFlags  string // Flags to pass to UPX
	CompiledBinary string // Fully qualified path to the compiled binary

	BindJSDir string // Directory to generate the wailsjs module

	ForceBuild bool // Force

	BundleName string // Bundlename for Mac

	TrimPath     bool // Use Go's trimpath compiler flag
	RaceDetector bool // Build with Go's race detector

	// 显示window console
	WindowsConsole bool // Indicates that the windows console should be kept

	// 是否跳过绑定生成
	SkipBindings bool // Skip binding generation

}
