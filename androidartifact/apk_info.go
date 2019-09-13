package androidartifact

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-android/sdk"
	"github.com/bitrise-io/go-utils/command"
)

// ApkInfo ...
type ApkInfo struct {
	AppName           string
	PackageName       string
	VersionCode       string
	VersionName       string
	MinSDKVersion     string
	RawPackageContent string
}

func packageField(data, key string) string {
	pattern := fmt.Sprintf(`%s=['"](.*?)['"]`, key)

	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(data); len(matches) == 2 {
		return matches[1]
	}

	return ""
}

// ParsePackageInfos ...
func ParsePackageInfos(aaptOut string) (string, string, string) {
	return packageField(aaptOut, "name"),
		packageField(aaptOut, "versionCode"),
		packageField(aaptOut, "versionName")
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

// GetAPKInfo ...
func GetAPKInfo(apkPth string) (ApkInfo, error) {
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome == "" {
		return ApkInfo{}, errors.New("ANDROID_HOME environment not set")
	}

	sdkModel, err := sdk.New(androidHome)
	if err != nil {
		return ApkInfo{}, fmt.Errorf("failed to create sdk model, error: %s", err)
	}

	aaptPth, err := sdkModel.LatestBuildToolPath("aapt")
	if err != nil {
		return ApkInfo{}, fmt.Errorf("failed to find latest aapt binary, error: %s", err)
	}

	aaptOut, err := command.New(aaptPth, "dump", "badging", apkPth).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return ApkInfo{}, fmt.Errorf("failed to get apk infos, output: %s, error: %s", aaptOut, err)
	}

	appName := filterAppLable(aaptOut)
	packageName, versionCode, versionName := ParsePackageInfos(aaptOut)
	minSDKVersion := filterMinSDKVersion(aaptOut)

	packageContent := ""
	for _, line := range strings.Split(aaptOut, "\n") {
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
