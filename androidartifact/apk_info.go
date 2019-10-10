package androidartifact

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/avast/apkparser"
	"github.com/bitrise-io/go-android/sdk"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
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

// parseAppName parses the application name from `aapt dump badging` command output.
func parseAppName(aaptOut string) string {
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

// parseMinSDKVersion parses the min sdk version from `aapt dump badging` command output.
func parseMinSDKVersion(aaptOut string) string {
	pattern := `sdkVersion:\'(?P<min_sdk_version>.*)\'`
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(aaptOut); len(matches) == 2 {
		return matches[1]
	}
	return ""
}

// parsePackageField parses fields from `aapt dump badging` command output.
func parsePackageField(aaptOut, key string) string {
	pattern := fmt.Sprintf(`%s=['"](.*?)['"]`, key)

	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(aaptOut); len(matches) == 2 {
		return matches[1]
	}

	return ""
}

// ParsePackageInfos parses package name, version code and name from `aapt dump badging` command output.
func ParsePackageInfos(aaptOut string) (string, string, string) {
	return parsePackageField(aaptOut, "name"),
		parsePackageField(aaptOut, "versionCode"),
		parsePackageField(aaptOut, "versionName")
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

func parseAPKInfo(apkPath string) (ApkInfo, error) {
	var manifestContent bytes.Buffer
	enc := xml.NewEncoder(&manifestContent)
	enc.Indent("", "\t")

	zipErr, resErr, manErr := apkparser.ParseApk(apkPath, enc)
	if zipErr != nil {
		return ApkInfo{}, fmt.Errorf("failed to unzip the APK, error: %s", zipErr)
	}
	if resErr != nil {
		return ApkInfo{}, fmt.Errorf("failed to parse resources, error: %s", zipErr)
	}
	if manErr != nil {
		return ApkInfo{}, fmt.Errorf("failed to parse AndroidManifest.xml, error: %s", zipErr)
	}

	var manifest manifest
	if err := xml.Unmarshal(manifestContent.Bytes(), &manifest); err != nil {
		return ApkInfo{}, fmt.Errorf("failed to unmarshal AndroidManifest.xml, error: %s", err)
	}

	return ApkInfo{
		AppName:           manifest.Application.AppName,
		PackageName:       manifest.PackageName,
		VersionCode:       manifest.VersionCode,
		VersionName:       manifest.VersionName,
		MinSDKVersion:     manifest.UsesSdk.MinSDKVersion,
		RawPackageContent: string(manifestContent.Bytes()),
	}, nil
}

func getAPKInfoWithAapt(apkPth string) (ApkInfo, error) {
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

	appName := parseAppName(aaptOut)
	packageName, versionCode, versionName := ParsePackageInfos(aaptOut)
	minSDKVersion := parseMinSDKVersion(aaptOut)

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

// GetAPKInfo returns infos about the APK.
func GetAPKInfo(apkPth string) (ApkInfo, error) {
	parsedInfo, err := parseAPKInfo(apkPth)
	if err == nil {
		return parsedInfo, nil
	}
	// err != nil
	log.Warnf("Failed to parse APK info: %s", err)
	log.RWarnf("deploy-to-bitrise-io", "apk-parse", nil, "apkparser package failed to parse APK, error: %s", err)

	return getAPKInfoWithAapt(apkPth)
}
