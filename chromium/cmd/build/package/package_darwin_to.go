package _package

import (
	"bytes"
	"github.com/jackmordaunt/icns"
	"github.com/pkg/errors"
	"image"
	"main/chromium/cmd/build/buildassets"
	"main/chromium/cmd/build/fs"
	"main/chromium/internal/build"
	"os"
	"path/filepath"
)

func PackageApplicationForDarwin(options *build.Options) error {
	var err error

	// Create directory structure
	bundlename := options.BundleName
	if bundlename == "" {
		bundlename = options.ProjectData.Name + ".app"
	}

	contentsDirectory := filepath.Join(options.BinDirectory, bundlename, "/Contents")
	exeDir := filepath.Join(contentsDirectory, "/MacOS")
	err = fs.MkDirs(exeDir, 0o755)
	if err != nil {
		return err
	}
	resourceDir := filepath.Join(contentsDirectory, "/Resources")
	err = fs.MkDirs(resourceDir, 0o755)
	if err != nil {
		return err
	}
	// Copy binary
	packedBinaryPath := filepath.Join(exeDir, options.ProjectData.Name)
	err = fs.MoveFile(options.CompiledBinary, packedBinaryPath)
	if err != nil {
		return errors.Wrap(err, "Cannot move file: "+options.ProjectData.OutputFilename)
	}

	// Generate Info.plist
	err = processPList(options, contentsDirectory)
	if err != nil {
		return err
	}

	// Generate App Icon
	err = processDarwinIcon(options.ProjectData, "appicon", resourceDir, "iconfile")
	if err != nil {
		return err
	}

	// Generate FileAssociation Icons
	//for _, fileAssociation := range options.ProjectData.Info.FileAssociations {
	//	err = processDarwinIcon(options.ProjectData, fileAssociation.IconName, resourceDir, "")
	//	if err != nil {
	//		return err
	//	}
	//}

	options.CompiledBinary = packedBinaryPath

	return nil
}

func processPList(options *build.Options, contentsDirectory string) error {
	sourcePList := "Info.plist"
	if options.Mode == build.Dev {
		// Use Info.dev.plist if using build mode
		sourcePList = "Info.dev.plist"
	}

	// Read the resolved BuildAssets file and copy it to the destination
	content, err := buildassets.ReadFileWithProjectData(options.ProjectData, "darwin/"+sourcePList)
	if err != nil {
		return err
	}

	targetFile := filepath.Join(contentsDirectory, "Info.plist")
	return os.WriteFile(targetFile, content, 0o644)
}

func processDarwinIcon(projectData *build.Project, iconName string, resourceDir string, destIconName string) (err error) {
	appIcon, err := buildassets.ReadFile(projectData, iconName+".png")
	if err != nil {
		return err
	}

	srcImg, _, err := image.Decode(bytes.NewBuffer(appIcon))
	if err != nil {
		return err
	}

	if destIconName == "" {
		destIconName = iconName
	}

	tgtBundle := filepath.Join(resourceDir, destIconName+".icns")
	dest, err := os.Create(tgtBundle)
	if err != nil {
		return err
	}
	defer func() {
		err = dest.Close()
		if err == nil {
			return
		}
	}()
	return icns.Encode(dest, srcImg)
}
