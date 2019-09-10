package uploaders

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-steplib/steps-xcode-test/pretty"
)

// BuildArtifactsMap module/buildType/flavour/artifacts
type BuildArtifactsMap map[string]map[string]map[string][]string

// based on: https://developer.android.com/ndk/guides/abis.html#sa
var abis = []string{"armeabi-v7a", "arm64-v8a", "x86_64", "x86", "universal"}
var unsupportedAbis = []string{"mips64", "mips", "armeabi"}

// based on: https://developer.android.com/studio/build/configure-apk-splits#configure-density-split
var screenDensities = []string{"xxxhdpi", "xxhdpi", "xhdpi", "hdpi", "mdpi", "ldpi", "280", "360", "420", "480", "560"}

// fileName return the given path's file name without extension and `-bitrise-signed`, `-unsigned` suffixes.
func fileName(pth string) string {
	// sign-apk step adds `-bitrise-signed` suffix to the artifact base name
	// https://github.com/bitrise-steplib/steps-sign-apk/blob/master/main.go#L411
	ext := filepath.Ext(pth)
	base := filepath.Base(pth)
	base = strings.TrimSuffix(base, ext)
	base = strings.TrimSuffix(base, "-bitrise-signed")
	return strings.TrimSuffix(base, "-unsigned")
}

func parseAppPath(pth string) (module string, productFlavour string, buildType string) {
	base := fileName(pth)

	// based on: https://developer.android.com/studio/build/build-variants
	// - <build variant> = <product flavor> + <build type>
	// - debug and release build types always exists
	// - APK/AAB base name layout: <module>-<product flavor?>-<build type>.<apk|aab>
	// - Sample APK path: $BITRISE_DEPLOY_DIR/app-minApi21-demo-hdpi-debug.apk
	s := strings.Split(base, "-")
	if len(s) < 2 {
		// unknown app base name
		// app artifact name can be customized: https://stackoverflow.com/a/28250257
		return "", "", ""
	}

	module = s[0]
	buildType = s[len(s)-1]
	if len(s) > 2 {
		productFlavour = strings.Join(s[1:len(s)-1], "-")
	}
	return
}

func firstLetterUpper(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}

// removeSplitParams removes split parts of the flavour.
func removeSplitParams(flavour string) string {
	// 2 flavours + density split: minApi21-full-hdpi
	// density and abi split: hdpiArmeabi
	// flavour + density and abi split: demo-hdpiArm64-v8a
	var splitParams []string
	splitParams = append(splitParams, abis...)
	splitParams = append(splitParams, unsupportedAbis...)
	splitParams = append(splitParams, screenDensities...)

	for _, splitParam := range splitParams {
		if strings.Contains(flavour, splitParam) {
			flavour = strings.Replace(flavour, splitParam, "", 1)
		}

		// in case of density + ABI split the 2. split param starts with upper case letter: demo-hdpiArm64-v8a
		if strings.Contains(flavour, firstLetterUpper(splitParam)) {
			flavour = strings.Replace(flavour, firstLetterUpper(splitParam), "", 1)
		}
	}

	// after removing split params, may leading/trailing - char remains: demo-hdpiArm64-v8a
	flavour = strings.TrimPrefix(flavour, "-")
	return strings.TrimSuffix(flavour, "-")
}

// mapBuildArtifacts returns map[module]map[buildType]map[productFlavour]path.
func mapBuildArtifacts(pths []string) BuildArtifactsMap {
	buildArtifacts := map[string]map[string]map[string][]string{}
	for _, pth := range pths {
		module, productFlavour, buildType := parseAppPath(pth)
		productFlavour = removeSplitParams(productFlavour)

		moduleArtifacts, ok := buildArtifacts[module]
		if !ok {
			moduleArtifacts = map[string]map[string][]string{}
		}

		buildTypeArtifacts, ok := moduleArtifacts[buildType]
		if !ok {
			buildTypeArtifacts = map[string][]string{}
		}

		artifacts := buildTypeArtifacts[productFlavour]
		added := false
		for _, suffix := range []string{"", "-unsigned", "-bitrise-signed"} {
			artifact := filepath.Join(filepath.Dir(pth), fileName(pth)+suffix+filepath.Ext(pth))

			if sliceutil.IsStringInSlice(artifact, artifacts) {
				added = true
				break
			}
		}

		if !added {
			buildTypeArtifacts[productFlavour] = append(artifacts, pth)
		}

		moduleArtifacts[buildType] = buildTypeArtifacts
		buildArtifacts[module] = moduleArtifacts
	}
	return buildArtifacts
}

func splitMeta(pth string, pths []string) (map[string]interface{}, error) {
	artifactsMap := mapBuildArtifacts(pths)
	module, flavourWithSplitParams, buildType := parseAppPath(pth)
	flavour := removeSplitParams(flavourWithSplitParams)

	if flavourWithSplitParams == flavour {
		return nil, nil
	}

	moduleArtifacts, ok := artifactsMap[module]
	if !ok {
		return nil, fmt.Errorf("artifact: %s is not part of the artifact mapping: %s", pth, pretty.Object(artifactsMap))
	}

	buildTypeArtifacts, ok := moduleArtifacts[buildType]
	if !ok {
		return nil, fmt.Errorf("artifact: %s is not part of the artifact mapping: %s", pth, pretty.Object(artifactsMap))
	}

	artifacts, ok := buildTypeArtifacts[flavour]
	if !ok {
		return nil, fmt.Errorf("artifact: %s is not part of the artifact mapping: %s", pth, pretty.Object(artifactsMap))
	}

	return map[string]interface{}{
		"split":     artifacts,
		"include":   sliceutil.IsStringInSlice(pth, artifacts),
		"universal": strings.Contains(flavourWithSplitParams, "universal"),
	}, nil
}
