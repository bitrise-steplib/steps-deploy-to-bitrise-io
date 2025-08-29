package androidartifact

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/avast/apkparser"
	"github.com/bitrise-io/go-android/v2/sdk"
	"github.com/bitrise-io/go-utils/command"
)

func GetAPKInfoWithFallback(logger Logger, apkPth string) (Info, error) {
	parsedInfo, err := GetAPKInfo(apkPth)
	if err == nil {
		return parsedInfo, nil
	}
	// err != nil
	logger.Warnf("Failed to parse APK info: %s", err)
	logger.APKParseWarnf("apk-parse", "apkparser package failed to parse APK, error: %s", err)

	return GetAPKInfoWithAapt(apkPth)
}

type manifest struct {
	XMLName     xml.Name `xml:"manifest"`
	VersionCode string   `xml:"versionCode,attr"`
	VersionName string   `xml:"versionName,attr"`
	PackageName string   `xml:"package,attr"`
	Application application
	UsesSdk     usesSdk
}

type application struct {
	XMLName     xml.Name `xml:"application"`
	PackageName string   `xml:"name,attr"`
	AppName     string   `xml:"label,attr"`
}

type usesSdk struct {
	XMLName       xml.Name `xml:"uses-sdk"`
	MinSDKVersion string   `xml:"minSdkVersion,attr"`
}

// GetAPKInfo returns infos about the APK.
func GetAPKInfo(apkPath string) (Info, error) {
	var manifestContent bytes.Buffer
	enc := xml.NewEncoder(&manifestContent)
	enc.Indent("", "\t")

	zipErr, resErr, manErr := apkparser.ParseApk(apkPath, enc)
	if zipErr != nil {
		return Info{}, fmt.Errorf("failed to unzip the APK, error: %s", zipErr)
	}
	if resErr != nil {
		return Info{}, fmt.Errorf("failed to parse resources, error: %s", zipErr)
	}
	if manErr != nil {
		return Info{}, fmt.Errorf("failed to parse AndroidManifest.xml, error: %s", zipErr)
	}

	var manifest manifest
	if err := xml.Unmarshal(manifestContent.Bytes(), &manifest); err != nil {
		return Info{}, fmt.Errorf("failed to unmarshal AndroidManifest.xml, error: %s", err)
	}

	return Info{
		AppName:           manifest.Application.AppName,
		PackageName:       manifest.PackageName,
		VersionCode:       manifest.VersionCode,
		VersionName:       manifest.VersionName,
		MinSDKVersion:     manifest.UsesSdk.MinSDKVersion,
		RawPackageContent: manifestContent.String(),
	}, nil
}

func GetAPKInfoWithAapt(apkPth string) (Info, error) {
	sdkModel, err := sdk.NewDefaultModel(sdk.Environment{
		AndroidHome:    os.Getenv("ANDROID_HOME"),
		AndroidSDKRoot: os.Getenv("ANDROID_SDK_ROOT"),
	})
	if err != nil {
		return Info{}, fmt.Errorf("failed to create sdk model, error: %s", err)
	}

	aaptPth, err := sdkModel.LatestBuildToolPath("aapt")
	if err != nil {
		return Info{}, fmt.Errorf("failed to find latest aapt binary, error: %s", err)
	}

	aaptOut, err := command.New(aaptPth, "dump", "badging", apkPth).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return Info{}, fmt.Errorf("failed to get apk infos, output: %s, error: %s", aaptOut, err)
	}

	appName := parseAppName(aaptOut)
	packageName, versionCode, versionName := ParsePackageInfo(aaptOut, "name")
	minSDKVersion := getByPattern(aaptOut, `sdkVersion:\'(?P<min_sdk_version>.*)\'`)

	packageContent := ""
	for _, line := range strings.Split(aaptOut, "\n") {
		if strings.HasPrefix(line, "package:") {
			packageContent = line
		}
	}

	return Info{
		AppName:           appName,
		PackageName:       packageName,
		VersionCode:       versionCode,
		VersionName:       versionName,
		MinSDKVersion:     minSDKVersion,
		RawPackageContent: packageContent,
	}, nil
}

// parseAppName parses the application name from `aapt dump badging` command output.
func parseAppName(aaptOut string) string {
	pattern := `application: label=\'(?P<label>.+)\' icon=`
	found := getByPattern(aaptOut, pattern)
	if found != "" {
		return found
	}

	pattern = `application-label:\'(?P<label>.*)\'`
	return getByPattern(aaptOut, pattern)
}
