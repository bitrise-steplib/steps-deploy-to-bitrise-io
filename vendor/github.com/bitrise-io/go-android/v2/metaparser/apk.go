package parser

import (
	"fmt"

	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"

	"github.com/bitrise-io/go-utils/v2/fileutil"
)

// ParseAPKData ...
func ParseAPKData(pth string) (*ArtifactMetadata, error) {
	apkInfo, err := androidartifact.GetAPKInfo(pth)
	if err != nil {
		return nil, err
	}

	var warnings []string
	if apkInfo.PackageName == "" {
		warnings = append(warnings, fmt.Sprintf("Package name is undefined, AndroidManifest.xml package content:\n%s", apkInfo.RawPackageContent))
	}

	if apkInfo.VersionCode == "" {
		warnings = append(warnings, fmt.Sprintf("Version code is undefined, AndroidManifest.xml package content:\n%s", apkInfo.RawPackageContent))
	}

	if apkInfo.VersionName == "" {
		warnings = append(warnings, fmt.Sprintf("Version name is undefined, AndroidManifest.xml package content:\n%s", apkInfo.RawPackageContent))
	}

	fileSize, err := fileutil.NewFileManager().FileSizeInBytes(pth)
	if err != nil {
		return nil, fmt.Errorf("failed to get apk size, error: %s", err)
	}

	info := androidartifact.ParseArtifactPath(pth)

	return &ArtifactMetadata{
		AppInfo:        apkInfo,
		FileSizeBytes:  fileSize,
		Module:         info.Module,
		ProductFlavour: info.ProductFlavour,
		BuildType:      info.BuildType,
		Warnings:       warnings,
	}, nil
}
