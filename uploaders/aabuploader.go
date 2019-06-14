package uploaders

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/bundletool"
)

// AABInfo ...
type AABInfo struct {
	AppName           string
	PackageName       string
	VersionCode       string
	VersionName       string
	MinSDKVersion     string
	RawPackageContent string
}

func getAABInfo(toolOutput string) (ApkInfo, error) {
	appName := filterAppLable(toolOutput)
	packageName, versionCode, versionName := filterPackageInfos(toolOutput)
	minSDKVersion := filterMinSDKVersion(toolOutput)

	packageContent := ""
	for _, line := range strings.Split(toolOutput, "\n") {
		if strings.HasPrefix(line, "package:") {
			packageContent = line
		}
	}

	return ApkInfo{
		AppName:           appName,
		PackageName:       packageName,
		VersionCode:       versionCode,
		VersionName:       versionName,
		MinSDKVersion:     minSDKVersion,
		RawPackageContent: packageContent,
	}, nil
}

// DeployAAB ...
func DeployAAB(pth, buildURL, token, notifyUserGroups, notifyEmails, isEnablePublicPage string) (string, error) {
	log.Printf("analyzing aab")

	r, err := bundletool.NewRunner()
	if err != nil {
		return "", err
	}

	out, err := r.Execute("dump", "manifest", "--bundle", pth)
	if err != nil {
		return "", err
	}

	apkInfo, err := getAABInfo(out)
	if err != nil {
		return "", err
	}

	appInfo := map[string]interface{}{
		"app_name":        apkInfo.AppName,
		"package_name":    apkInfo.PackageName,
		"version_code":    apkInfo.VersionCode,
		"version_name":    apkInfo.VersionName,
		"min_sdk_version": apkInfo.MinSDKVersion,
	}

	log.Printf("  apk infos: %v", appInfo)

	if apkInfo.PackageName == "" {
		log.Warnf("Package name is undefined, AndroidManifest.xml package content:\n%s", apkInfo.RawPackageContent)
	}

	if apkInfo.VersionCode == "" {
		log.Warnf("Version code is undefined, AndroidManifest.xml package content:\n%s", apkInfo.RawPackageContent)
	}

	if apkInfo.VersionName == "" {
		log.Warnf("Version name is undefined, AndroidManifest.xml package content:\n%s", apkInfo.RawPackageContent)
	}

	// ---

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return "", fmt.Errorf("failed to get apk size, error: %s", err)
	}

	apkInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%f", fileSize),
		"app_info":        appInfo,
	}

	artifactInfoBytes, err := json.Marshal(apkInfoMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal apk infos, error: %s", err)
	}

	// ---

	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "android-apk")
	if err != nil {
		return "", fmt.Errorf("failed to create apk artifact, error: %s", err)
	}

	if err := uploadArtifact(uploadURL, pth, "application/vnd.android.package-archive"); err != nil {
		return "", fmt.Errorf("failed to upload apk artifact, error: %s", err)
	}

	publicInstallPage, err := finishArtifact(buildURL, token, artifactID, string(artifactInfoBytes), notifyUserGroups, notifyEmails, isEnablePublicPage)
	if err != nil {
		return "", fmt.Errorf("failed to finish apk artifact, error: %s", err)
	}

	return publicInstallPage, nil
}
