package parser

import (
	"fmt"

	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"
	"github.com/bitrise-io/go-android/v2/metaparser/androidsignature"
	"github.com/bitrise-io/go-android/v2/metaparser/bundletool"
	"github.com/bitrise-io/go-utils/v2/fileutil"
)

// ParseAABData ...
func ParseAABData(pth string, bt bundletool.Path) (map[string]interface{}, error) {
	aabInfo, err := androidartifact.GetAABInfo(bt, pth)
	if err != nil {
		return nil, err
	}

	appInfo := map[string]interface{}{
		"package_name":    aabInfo.PackageName,
		"version_code":    aabInfo.VersionCode,
		"version_name":    aabInfo.VersionName,
		"app_name":        aabInfo.AppName,
		"min_sdk_version": aabInfo.MinSDKVersion,
	}

	var warnings []string
	if aabInfo.PackageName == "" {
		warnings = append(warnings, fmt.Sprintf("Package name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent))
	}

	if aabInfo.VersionCode == "" {
		warnings = append(warnings, fmt.Sprintf("Version code is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent))
	}

	if aabInfo.VersionName == "" {
		warnings = append(warnings, fmt.Sprintf("Version name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent))
	}

	if aabInfo.MinSDKVersion == "" {
		warnings = append(warnings, fmt.Sprintf("Min SDK version is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent))
	}

	if aabInfo.AppName == "" {
		warnings = append(warnings, fmt.Sprintf("App name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent))
	}

	fileSize, err := fileutil.NewFileManager().FileSizeInBytes(pth)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to get apk size, error: %s", err))
	}

	info := androidartifact.ParseArtifactPath(pth)

	signature, err := androidsignature.Read(pth)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to read signature: %s", err))
	}

	return map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%d", fileSize),
		"app_info":        appInfo,
		"module":          info.Module,
		"product_flavour": info.ProductFlavour,
		"build_type":      info.BuildType,
		"signed_by":       signature,
		"warnings":        warnings,
	}, nil
}
