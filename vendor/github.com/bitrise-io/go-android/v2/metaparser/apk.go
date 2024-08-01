package parser

import (
	"fmt"

	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"

	"github.com/bitrise-io/go-utils/v2/fileutil"
)

// ParseAPKData ...
func ParseAPKData(pth string) (map[string]interface{}, error) {
	apkInfo, err := androidartifact.GetAPKInfo(pth)
	if err != nil {
		return nil, err
	}

	appInfo := map[string]interface{}{
		"app_name":        apkInfo.AppName,
		"package_name":    apkInfo.PackageName,
		"version_code":    apkInfo.VersionCode,
		"version_name":    apkInfo.VersionName,
		"min_sdk_version": apkInfo.MinSDKVersion,
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

	return map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%d", fileSize),
		"app_info":        appInfo,
		"module":          info.Module,
		"product_flavour": info.ProductFlavour,
		"build_type":      info.BuildType,
		"warnings":        warnings,
	}, nil
}
