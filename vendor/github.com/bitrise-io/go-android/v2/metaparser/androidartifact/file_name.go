package androidartifact

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pretty"
	"github.com/bitrise-io/go-utils/sliceutil"
)

const universalSplitParam = "universal"

// The order of split params matter, while the artifact path parsing is done, we remove the split params in this order.
// If we would remove `xhdpi` from app-xxxhdpi-debug.apk, the remaining part would be: app-xx-debug.apk.
var (
	// based on: https://developer.android.com/ndk/guides/abis.html#sa
	abis            = []string{"armeabi-v7a", "arm64-v8a", "x86_64", "x86", universalSplitParam}
	unsupportedAbis = []string{"mips64", "mips", "armeabi"}

	// based on: https://developer.android.com/studio/build/configure-apk-splits#configure-density-split
	screenDensities = []string{"xxxhdpi", "xxhdpi", "xhdpi", "hdpi", "mdpi", "ldpi", "280", "360", "420", "480", "560"}
)

// ArtifactSigningInfo ...
type ArtifactSigningInfo struct {
	Unsigned      bool
	BitriseSigned bool
}

const bitriseSignedSuffix = "-bitrise-signed"
const unsignedSuffix = "-unsigned"

// parseSigningInfo parses android artifact path
// and returns codesigning info and the artifact's base name without signing params.
func parseSigningInfo(pth string) (ArtifactSigningInfo, string) {
	info := ArtifactSigningInfo{}

	ext := filepath.Ext(pth)
	base := filepath.Base(pth)
	base = strings.TrimSuffix(base, ext)

	// a given artifact is either:
	// signed: no suffix
	// unsigned: `-unsigned` suffix
	// bitrise signed: `-bitrise-signed` suffix: https://github.com/bitrise-steplib/steps-sign-apk/blob/master/main.go#L411
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

// firstLetterUpper makes the given string's first letter uppercase.
func firstLetterUpper(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}

// parseSplitInfo parses the flavour candidate part of the artifact's base name
// and returns APK split info and the flavour without split params.
func parseSplitInfo(flavour string) (ArtifactSplitInfo, string) {
	// 2 flavours + density split: minApi21-full-hdpi
	// density and abi split: hdpiArmeabi
	// flavour + density and abi split: demo-hdpiArm64-v8a
	var info ArtifactSplitInfo

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

// ParseArtifactPath parses an android artifact path.
func ParseArtifactPath(pth string) ArtifactInfo {
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

// ArtifactMap module/buildType/flavour/artifacts
type ArtifactMap map[string]map[string]map[string]Artifact

// Artifact ...
type Artifact struct {
	APK          string   `json:"apk"` // set if a single APK represents the app
	AAB          string   `json:"aab"`
	Split        []string `json:"split"` // split apk paths including the universal apk path, excluding the bundle path
	UniversalApk string   `json:"universal"`
}

// FindSameArtifact returns the first artifact which is the same variant as the reference artifact,
// code signing differences does not matter.
func FindSameArtifact(pth string, pths []string) string {
	for _, suffix := range []string{"", unsignedSuffix, bitriseSignedSuffix} {
		_, base := parseSigningInfo(pth)
		artifactPth := filepath.Join(filepath.Dir(pth), base+suffix+filepath.Ext(pth))

		if idx := sliceutil.IndexOfStringInSlice(artifactPth, pths); idx > -1 {
			return pths[idx]
		}
	}
	return ""
}

// mapBuildArtifacts creates a module/buildType/productFlavour[artifactPaths] mapping.
func mapBuildArtifacts(pths []string) ArtifactMap {
	buildArtifacts := map[string]map[string]map[string]Artifact{}
	for _, pth := range pths {
		info := ParseArtifactPath(pth)

		moduleArtifacts, ok := buildArtifacts[info.Module]
		if !ok {
			moduleArtifacts = map[string]map[string]Artifact{}
		}

		buildTypeArtifacts, ok := moduleArtifacts[info.BuildType]
		if !ok {
			buildTypeArtifacts = map[string]Artifact{}
		}

		artifact := buildTypeArtifacts[info.ProductFlavour]

		if filepath.Ext(pth) == ".aab" {
			if len(artifact.AAB) != 0 {
				log.Warnf("Multiple AAB generated for module: %s, productFlavour: %s, buildType: %s: %s", info.Module, info.ProductFlavour, info.BuildType, pth)
			}
			artifact.AAB = pth
			buildTypeArtifacts[info.ProductFlavour] = artifact
			moduleArtifacts[info.BuildType] = buildTypeArtifacts
			buildArtifacts[info.Module] = moduleArtifacts
			continue
		}

		if len(info.SplitInfo.SplitParams) == 0 {
			artifact.APK = pth
			buildTypeArtifacts[info.ProductFlavour] = artifact
			moduleArtifacts[info.BuildType] = buildTypeArtifacts
			buildArtifacts[info.Module] = moduleArtifacts
			continue
		}

		if info.SplitInfo.Universal {
			if len(artifact.UniversalApk) != 0 {
				log.Warnf("Multiple universal APK generated for module: %s, productFlavour: %s, buildType: %s: %s", info.Module, info.ProductFlavour, info.BuildType, pth)
			}
			artifact.UniversalApk = pth
		}

		// might -unsigned and -bitrise-signed versions of the same apk is listed
		pairPth := FindSameArtifact(pth, artifact.Split)
		if len(pairPth) == 0 {
			artifact.Split = append(artifact.Split, pth)
			buildTypeArtifacts[info.ProductFlavour] = artifact
		}

		moduleArtifacts[info.BuildType] = buildTypeArtifacts
		buildArtifacts[info.Module] = moduleArtifacts
	}

	return buildArtifacts
}

// remove deletes an element of an array.
func remove(slice []string, i uint) []string {
	if int(i) > len(slice)-1 {
		return slice
	}
	return append(slice[:i], slice[i+1:]...)
}

// SplitArtifactMeta ...
type SplitArtifactMeta Artifact

// CreateSplitArtifactMeta ...
func CreateSplitArtifactMeta(pth string, pths []string) (SplitArtifactMeta, error) {
	artifactsMap := mapBuildArtifacts(pths)
	info := ParseArtifactPath(pth)

	moduleArtifacts, ok := artifactsMap[info.Module]
	if !ok {
		return SplitArtifactMeta{}, fmt.Errorf("artifact: %s is not part of the artifact mapping: %s", pth, pretty.Object(artifactsMap))
	}

	buildTypeArtifacts, ok := moduleArtifacts[info.BuildType]
	if !ok {
		return SplitArtifactMeta{}, fmt.Errorf("artifact: %s is not part of the artifact mapping: %s", pth, pretty.Object(artifactsMap))
	}

	artifact, ok := buildTypeArtifacts[info.ProductFlavour]
	if !ok {
		return SplitArtifactMeta{}, fmt.Errorf("artifact: %s is not part of the artifact mapping: %s", pth, pretty.Object(artifactsMap))
	}

	return SplitArtifactMeta(artifact), nil
}
