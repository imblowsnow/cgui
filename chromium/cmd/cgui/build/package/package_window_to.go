package _package

import (
	"bytes"
	"fmt"
	"github.com/imblowsnow/cgui/chromium/cmd/cgui/build/buildassets"
	"github.com/imblowsnow/cgui/chromium/cmd/cgui/build/fs"
	"github.com/imblowsnow/cgui/chromium/internal/build"
	"github.com/leaanthony/winicon"
	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
	"os"
	"path/filepath"
)

func PackageApplicationForWindows(options *build.Options) error {
	// Generate app icon
	var err error
	err = generateIcoFile(options, "appicon", "icon")
	if err != nil {
		return err
	}

	// Generate FileAssociation Icons
	for _, fileAssociation := range options.ProjectData.Info.FileAssociations {
		err = generateIcoFile(options, fileAssociation.IconName, "")
		if err != nil {
			return err
		}
	}

	// Create syso file
	err = compileResources(options)
	if err != nil {
		return err
	}

	return nil
}

// 生成图标文件 build/appicon.png -> build/windows/icon.ico
func generateIcoFile(options *build.Options, iconName string, destIconName string) error {
	content, err := buildassets.ReadFile(options.ProjectData, iconName+".png")
	if err != nil {
		return err
	}

	if destIconName == "" {
		destIconName = iconName
	}

	// Check ico file exists already
	icoFile := buildassets.GetLocalPath(options.ProjectData, "windows/"+destIconName+".ico")
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

// 生成exe打包syso文件
func compileResources(options *build.Options) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Chdir(currentDir)
	}()
	windowsDir := filepath.Join(options.ProjectData.GetBuildDir(), "windows")
	err = os.Chdir(windowsDir)
	if err != nil {
		return err
	}
	rs := winres.ResourceSet{}
	icon := filepath.Join(windowsDir, "icon.ico")
	iconFile, err := os.Open(icon)
	if err != nil {
		return err
	}
	defer iconFile.Close()
	ico, err := winres.LoadICO(iconFile)
	if err != nil {
		return fmt.Errorf("couldn't load icon from icon.ico: %w", err)
	}
	err = rs.SetIcon(winres.RT_ICON, ico)
	if err != nil {
		return err
	}

	manifestData, err := buildassets.ReadFileWithProjectData(options.ProjectData, "windows/wails.exe.manifest")
	if err != nil {
		return err
	}

	xmlData, err := winres.AppManifestFromXML(manifestData)
	if err != nil {
		return err
	}
	rs.SetManifest(xmlData)

	versionInfo, err := buildassets.ReadFileWithProjectData(options.ProjectData, "windows/info.json")
	if err != nil {
		return err
	}

	if len(versionInfo) != 0 {
		var v version.Info
		if err := v.UnmarshalJSON(versionInfo); err != nil {
			return err
		}
		rs.SetVersionInfo(v)
	}

	targetFile := filepath.Join(options.ProjectData.Path, options.ProjectData.Name+"-res.syso")
	fout, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer fout.Close()

	archs := map[string]winres.Arch{
		"amd64": winres.ArchAMD64,
		"arm64": winres.ArchARM64,
		"386":   winres.ArchI386,
	}
	targetArch, supported := archs[options.Arch]
	if !supported {
		return fmt.Errorf("arch '%s' not supported", options.Arch)
	}

	err = rs.WriteObject(fout, targetArch)
	if err != nil {
		return err
	}
	return nil
}
