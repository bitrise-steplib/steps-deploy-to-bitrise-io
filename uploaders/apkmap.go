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

const universalSplitParam = "universal"

// based on: https://developer.android.com/ndk/guides/abis.html#sa
var abis = []string{"armeabi-v7a", "arm64-v8a", "x86_64", "x86", universalSplitParam}
var unsupportedAbis = []string{"mips64", "mips", "armeabi"}

// based on: https://developer.android.com/studio/build/configure-apk-splits#configure-density-split
var screenDensities = []string{"xxxhdpi", "xxhdpi", "xhdpi", "hdpi", "mdpi", "ldpi", "280", "360", "420", "480", "560"}

// ArtifactSigningInfo ...
type ArtifactSigningInfo struct {
	Unsigned      bool
	BitriseSigned bool
}

const bitriseSignedSuffix = "-bitrise-signed"
const unsignedSuffix = "-unsigned"

func parseSigningInfo(pth string) (ArtifactSigningInfo, string) {
	info := ArtifactSigningInfo{}

	ext := filepath.Ext(pth)
	base := filepath.Base(pth)
	base = strings.TrimSuffix(base, ext)

	// a given artifact is either:
	// - signed: no suffix
	// - unsigned: `-unsigned` suffix
	// - bitrise signed: `-bitrise-signed` suffix: https://github.com/bitrise-steplib/steps-sign-apk/blob/master/main.go#L411
	if strings.HasSuffix(base, bitriseSignedSuffix) {
		base = strings.TrimSuffix(base, bitriseSignedSuffix)
		info.BitriseSigned = true
	}

	if strings.HasSuffix(base, unsignedSuffix) {
		base = strings.TrimSuffix(base, unsignedSuffix)
		info.Unsigned = true
	}

	return info, base
}

// ArtifactSplitInfo ...
type ArtifactSplitInfo struct {
	SplitParams []string
	Universal   bool
}

func firstLetterUpper(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}

func parseSplitInfo(flavour string) (ArtifactSplitInfo, string) {
	// 2 flavours + density split: minApi21-full-hdpi
	// density and abi split: hdpiArmeabi
	// flavour + density and abi split: demo-hdpiArm64-v8a
	info := ArtifactSplitInfo{}

	var splitParams []string
	splitParams = append(splitParams, abis...)
	splitParams = append(splitParams, unsupportedAbis...)
	splitParams = append(splitParams, screenDensities...)

	for _, splitParam := range splitParams {
		// in case of density + ABI split the 2. split param starts with upper case letter: demo-hdpiArm64-v8a
		for _, param := range []string{splitParam, firstLetterUpper(splitParam)} {
			if strings.Contains(flavour, param) {
				flavour = strings.Replace(flavour, param, "", 1)

				info.SplitParams = append(info.SplitParams, splitParam)
				if splitParam == universalSplitParam {
					info.Universal = true
				}

				break
			}
		}
	}

	// after removing split params, may leading/trailing - char remains: demo-hdpiArm64-v8a
	flavour = strings.TrimPrefix(flavour, "-")
	return info, strings.TrimSuffix(flavour, "-")
}

// ArtifactInfo ...
type ArtifactInfo struct {
	Module         string
	ProductFlavour string
	BuildType      string

	SigningInfo ArtifactSigningInfo
	SplitInfo   ArtifactSplitInfo
}

func parseAppPath(pth string) ArtifactInfo {
	info := ArtifactInfo{}

	var base string
	info.SigningInfo, base = parseSigningInfo(pth)

	// based on: https://developer.android.com/studio/build/build-variants
	// - <build variant> = <product flavor> + <build type>
	// - debug and release build types always exists
	// - APK/AAB base name layout: <module>-<product flavor?>-<build type>.<apk|aab>
	// - Sample APK path: $BITRISE_DEPLOY_DIR/app-minApi21-demo-hdpi-debug.apk
	s := strings.Split(base, "-")
	if len(s) < 2 {
		// unknown app base name
		// app artifact name can be customized: https://stackoverflow.com/a/28250257
		return info
	}

	info.Module = s[0]
	info.BuildType = s[len(s)-1]
	if len(s) > 2 {
		productFlavourWithSplitParams := strings.Join(s[1:len(s)-1], "-")
		info.SplitInfo, info.ProductFlavour = parseSplitInfo(productFlavourWithSplitParams)

	}
	return info
}

// mapBuildArtifacts returns map[module]map[buildType]map[productFlavour]path.
func mapBuildArtifacts(pths []string) BuildArtifactsMap {
	buildArtifacts := map[string]map[string]map[string][]string{}
	for _, pth := range pths {
		info := parseAppPath(pth)

		moduleArtifacts, ok := buildArtifacts[info.Module]
		if !ok {
			moduleArtifacts = map[string]map[string][]string{}
		}

		buildTypeArtifacts, ok := moduleArtifacts[info.BuildType]
		if !ok {
			buildTypeArtifacts = map[string][]string{}
		}

		artifacts := buildTypeArtifacts[info.ProductFlavour]
		added := false
		for _, suffix := range []string{"", "-unsigned", "-bitrise-signed"} {
			_, base := parseSigningInfo(pth)
			artifact := filepath.Join(filepath.Dir(pth), base+suffix+filepath.Ext(pth))

			if sliceutil.IsStringInSlice(artifact, artifacts) {
				added = true
				break
			}
		}

		if !added {
			buildTypeArtifacts[info.ProductFlavour] = append(artifacts, pth)
		}

		moduleArtifacts[info.BuildType] = buildTypeArtifacts
		buildArtifacts[info.Module] = moduleArtifacts
	}
	return buildArtifacts
}

func splitMeta(pth string, pths []string) (map[string]interface{}, error) {
	artifactsMap := mapBuildArtifacts(pths)
	info := parseAppPath(pth)

	if len(info.SplitInfo.SplitParams) == 0 {
		return nil, nil
	}

	moduleArtifacts, ok := artifactsMap[info.Module]
	if !ok {
		return nil, fmt.Errorf("artifact: %s is not part of the artifact mapping: %s", pth, pretty.Object(artifactsMap))
	}

	buildTypeArtifacts, ok := moduleArtifacts[info.BuildType]
	if !ok {
		return nil, fmt.Errorf("artifact: %s is not part of the artifact mapping: %s", pth, pretty.Object(artifactsMap))
	}

	artifacts, ok := buildTypeArtifacts[info.ProductFlavour]
	if !ok {
		return nil, fmt.Errorf("artifact: %s is not part of the artifact mapping: %s", pth, pretty.Object(artifactsMap))
	}

	return map[string]interface{}{
		"split":     artifacts,
		"include":   sliceutil.IsStringInSlice(pth, artifacts),
		"universal": info.SplitInfo.Universal,
	}, nil
}
