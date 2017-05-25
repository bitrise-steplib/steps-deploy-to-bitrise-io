package uploaders

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-android/sdk"
)

// ApkInfo ...
type ApkInfo struct {
	AppName       string
	PackageName   string
	VersionCode   string
	VersionName   string
	MinSDKVersion string
}

func filterPackageInfos(aaptOut string) (string, string, string) {
	pattern := `package: name=\'(?P<package_name>.*)\' versionCode=\'(?P<version_code>.*)\' versionName=\'(?P<version_name>.*)\' platformBuildVersionName=`
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(aaptOut); len(matches) == 4 {
		return matches[1], matches[2], matches[3]
	}
	return "", "", ""
}

func filterAppLable(aaptOut string) string {
	pattern := `application: label=\'(?P<label>.+)\' icon=`
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(aaptOut); len(matches) == 2 {
		return matches[1]
	}

	pattern = `application-label:\'(?P<label>.*)\'`
	re = regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(aaptOut); len(matches) == 2 {
		return matches[1]
	}

	return ""
}

func filterMinSDKVersion(aaptOut string) string {
	pattern := `sdkVersion:\'(?P<min_sdk_version>.*)\'`
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(aaptOut); len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func getAPKInfo(apkPth string) (ApkInfo, error) {
	sdkModel, err := sdk.New(os.Getenv("ANDROID_HOME"))
	if err != nil {
		return ApkInfo{}, err
	}

	aaptPth, err := sdkModel.LatestBuildToolPath("aapt")
	if err != nil {
		return ApkInfo{}, err
	}

	aaptOut, err := command.New(aaptPth, "dump", "badging", apkPth).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return ApkInfo{}, err
	}

	appName := filterAppLable(aaptOut)
	packageName, versionCode, versionName := filterPackageInfos(aaptOut)
	minSDKVersion := filterMinSDKVersion(aaptOut)

	return ApkInfo{
		AppName:       appName,
		PackageName:   packageName,
		VersionCode:   versionCode,
		VersionName:   versionName,
		MinSDKVersion: minSDKVersion,
	}, nil
}

// DeployAPK ...
func DeployAPK(pth, buildURL, token, notifyUserGroups, notifyEmails, isEnablePublicPage string) (string, error) {
	log.Printf("analyzing apk")

	apkInfo, err := getAPKInfo(pth)
	if err != nil {
		return "", fmt.Errorf("failed to get apk infos, error: %s", err)
	}

	appInfo := map[string]interface{}{
		"app_name":        apkInfo.AppName,
		"package_name":    apkInfo.PackageName,
		"version_code":    apkInfo.VersionCode,
		"version_name":    apkInfo.VersionName,
		"min_sdk_version": apkInfo.MinSDKVersion,
	}

	log.Printf("  apk infos: %v", appInfo)

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
