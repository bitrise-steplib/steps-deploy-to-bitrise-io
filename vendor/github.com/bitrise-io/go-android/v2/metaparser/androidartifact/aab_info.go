package androidartifact

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-android/v2/metaparser/bundletool"
	"github.com/bitrise-io/go-utils/log"
)

// AabInfo ...
type AabInfo struct {
	AppName           string
	PackageName       string
	VersionCode       string
	VersionName       string
	MinSDKVersion     string
	RawPackageContent string
}

// GetAABInfo returns infos about the AAB.
func GetAABInfo(bt bundletool.Path, aabPth string) (AabInfo, error) {
	parsedInfo, err := getAABInfoWithBundletool(bt, aabPth)
	if err != nil {
		log.Warnf("Failed to parse AAB info: %s", err)
		log.RWarnf("deploy-to-bitrise-io", "aab-parse", nil, "aabparser package failed to parse AAB, error: %s", err)
	}

	return parsedInfo, err
}

func getAABInfoWithBundletool(bt bundletool.Path, aabPath string) (AabInfo, error) {
	manifestContent, err := bt.Exec("dump", "manifest", "--bundle", aabPath)
	if err != nil {
		return AabInfo{}, err
	}

	packageName, versionCode, versionName := ParsePackageInfo(manifestContent, "package")
	minSDKVersion := getByPattern(manifestContent, `minSdkVersion=['"](.*?)['"]`)
	appName := getAppNameFromManifest(manifestContent)

	if strings.HasPrefix(appName, "@") {
		resourcesContent, err := bt.Exec("dump", "resources", "--bundle", aabPath, "--resource", appName[1:], "--values")
		if err != nil {
			return AabInfo{}, err
		}

		appName = getAppNameFromResources(resourcesContent)
	}

	return AabInfo{
		AppName:           appName,
		PackageName:       packageName,
		VersionCode:       versionCode,
		VersionName:       versionName,
		MinSDKVersion:     minSDKVersion,
		RawPackageContent: manifestContent,
	}, nil
}

func getAppNameFromManifest(aaptOut string) string {
	return getByPattern(aaptOut, `label=['"](.*?)['"]`)
}

func getAppNameFromResources(aaptOut string) string {
	pattern := `['"](.*?)['"]`

	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(aaptOut, 2)
	if len(matches) == 2 {
		appNameMatch := matches[1]
		if reflect.TypeOf(appNameMatch).Kind() == reflect.Slice && len(appNameMatch) == 2 {
			return appNameMatch[1]
		}
	}

	return ""
}
