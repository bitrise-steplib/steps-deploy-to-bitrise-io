package uploaders

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/bundletool"
)

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

	tmpPth, err := pathutil.NormalizedOSTempDirPath("apks")
	if err != nil {
		return "", err
	}

	apksPth := filepath.Join(tmpPth, "apks.apks")

	_, err = r.Execute("build-apks", "--mode=universal", "--bundle", pth, "--output", apksPth)
	if err != nil {
		return "", err
	}

	const spec = `{
		"sdkVersion": 100,
		"screenDensity": 560,
		"supportedAbis": ["arm64-v8a", "armeabi-v7a", "armeabi"],
		"supportedLocales": ["en-US"]
	}`

	specPath := filepath.Join(tmpPth, "spec.json")
	if err := ioutil.WriteFile(specPath, []byte(spec), 0777); err != nil {
		return "", err
	}
	//extract-apks --apks /tmp/lol.apks --output-dir /tmp/here --device-spec /tmp/spec.json
	o, err := r.Execute("extract-apks", "--apks", apksPth, "--output-dir", tmpPth, "--device-spec", specPath)
	if err != nil {
		return "", err
	}

	fmt.Println(o)

	universalAPKPath := filepath.Join(tmpPth, "universal.apk")
	renamedUniversalAPKPath := filepath.Join(tmpPth, filepath.Base(pth)+".universal.apk")
	if err := os.Rename(universalAPKPath, renamedUniversalAPKPath); err != nil {
		return "", err
	}

	out, err := r.Execute("dump", "manifest", "--bundle", pth)
	if err != nil {
		return "", err
	}

	aabInfo, err := getAABInfo(out)
	if err != nil {
		return "", err
	}

	appInfo := map[string]interface{}{
		"app_name":        aabInfo.AppName,
		"package_name":    aabInfo.PackageName,
		"version_code":    aabInfo.VersionCode,
		"version_name":    aabInfo.VersionName,
		"min_sdk_version": aabInfo.MinSDKVersion,
	}

	log.Printf("  aab infos: %v", appInfo)

	if aabInfo.PackageName == "" {
		log.Warnf("Package name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent)
	}

	if aabInfo.VersionCode == "" {
		log.Warnf("Version code is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent)
	}

	if aabInfo.VersionName == "" {
		log.Warnf("Version name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent)
	}

	// ---

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return "", fmt.Errorf("failed to get apk size, error: %s", err)
	}

	aabInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%f", fileSize),
		"app_info":        appInfo,
	}

	artifactInfoBytes, err := json.Marshal(aabInfoMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal apk infos, error: %s", err)
	}

	// ---

	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "android-apk")
	if err != nil {
		return "", fmt.Errorf("failed to create apk artifact, error: %s", err)
	}

	if err := uploadArtifact(uploadURL, pth, ""); err != nil {
		return "", fmt.Errorf("failed to upload apk artifact, error: %s", err)
	}

	if _, err = finishArtifact(buildURL, token, artifactID, string(artifactInfoBytes), "", "", "false"); err != nil {
		return "", fmt.Errorf("failed to finish apk artifact, error: %s", err)
	}

	fmt.Println()

	return DeployAPK(renamedUniversalAPKPath, buildURL, token, notifyUserGroups, notifyEmails, isEnablePublicPage)
}
