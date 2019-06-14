package uploaders

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/bundletool"
)

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

	if err := command.New("unzip", apksPth, "-d", tmpPth).Run(); err != nil {
		return "", err
	}

	universalAPKPath := filepath.Join(tmpPth, "universal.apk")
	renamedUniversalAPKPath := filepath.Join(tmpPth, filepath.Base(pth)+".universal.apk")
	if err := os.Rename(universalAPKPath, renamedUniversalAPKPath); err != nil {
		return "", err
	}

	aabInfo, err := getAPKInfo(renamedUniversalAPKPath)
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

	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "file", "")
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

	uploadURL, artifactID, err = createArtifact(buildURL, token, renamedUniversalAPKPath, "android-apk", filepath.Base(pth))
	if err != nil {
		return "", fmt.Errorf("failed to create apk artifact, error: %s", err)
	}

	if err := uploadArtifact(uploadURL, renamedUniversalAPKPath, "application/vnd.android.package-archive"); err != nil {
		return "", fmt.Errorf("failed to upload apk artifact, error: %s", err)
	}

	publicInstallPage, err := finishArtifact(buildURL, token, artifactID, string(artifactInfoBytes), notifyUserGroups, notifyEmails, isEnablePublicPage)
	if err != nil {
		return "", fmt.Errorf("failed to finish apk artifact, error: %s", err)
	}

	return publicInstallPage, nil
}
