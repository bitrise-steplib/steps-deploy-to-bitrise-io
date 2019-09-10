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

// create debug keystore for signing
func generateKeystore(tmpPth string) (string, error) {
	log.Printf("- generating debug keystore")

	keystorePath := filepath.Join(tmpPth, "debug.keystore")
	cmd := command.New("keytool", "-genkey", "-v", "-keystore", keystorePath, "-storepass", "android", "-alias", "androiddebugkey",
		"-keypass", "android", "-keyalg", "RSA", "-keysize", "2048", "-validity", "10000", "-dname", "C=US, O=Android, CN=Android Debug").
		SetStdout(os.Stdout).
		SetStderr(os.Stderr)

	log.Donef("$ %s", cmd.PrintableCommandArgs())

	return keystorePath, cmd.Run()
}

// generate `tmpDir/universal.apks` from aab file
func buildApksArchive(r bundletool.Runner, tmpPth, aabPth, keystorePath string) (string, error) {
	log.Printf("- generating universal apk")

	apksPth := filepath.Join(tmpPth, "universal.apks")
	cmd := r.Command("build-apks", "--mode=universal", "--bundle", aabPth, "--output", apksPth, "--ks", keystorePath, "--ks-pass", "pass:android", "--ks-key-alias", "androiddebugkey", "--key-pass", "pass:android").SetStdout(os.Stdout).SetStderr(os.Stderr)

	log.Donef("$ %s", cmd.PrintableCommandArgs())

	return apksPth, cmd.Run()
}

// DeployAAB ...
func DeployAAB(pth string, artifacts []string, buildURL, token, notifyUserGroups, notifyEmails, isEnablePublicPage string) (string, error) {
	log.Printf("- analyzing aab")

	tmpPth, err := pathutil.NormalizedOSTempDirPath("aab-bundle")
	if err != nil {
		return "", err
	}

	r, err := bundletool.NewRunner()
	if err != nil {
		return "", err
	}

	fmt.Println()

	keystorePath, err := generateKeystore(tmpPth)
	if err != nil {
		return "", err
	}

	fmt.Println()

	apksPth, err := buildApksArchive(r, tmpPth, pth, keystorePath)
	if err != nil {
		return "", err
	}

	fmt.Println()

	// unzip `tmpDir/universal.apks` to tmpPth to have `tmpDir/universal.apk`
	log.Printf("- unzip")
	cmd := command.New("unzip", apksPth, "-d", tmpPth).SetStdout(os.Stdout).SetStderr(os.Stderr)

	log.Donef("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		return "", err
	}

	fmt.Println()
	log.Printf("- rename")

	// rename `tmpDir/universal.apk` to `tmpDir/aab-name.aab.universal.apk`
	universalAPKPath, renamedUniversalAPKPath := filepath.Join(tmpPth, "universal.apk"), filepath.Join(tmpPth, filepath.Base(pth)+".universal.apk")
	if err := os.Rename(universalAPKPath, renamedUniversalAPKPath); err != nil {
		return "", err
	}

	fmt.Println()

	// get aab manifest dump
	log.Printf("- fetching info")

	cmd = r.Command("dump", "manifest", "--bundle", pth)

	log.Donef("$ %s", cmd.PrintableCommandArgs())

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return "", err
	}

	packageName, versionCode, versionName := filterPackageInfos(out)

	appInfo := map[string]interface{}{
		"package_name": packageName,
		"version_code": versionCode,
		"version_name": versionName,
	}

	splitM, err := splitMeta(pth, artifacts)
	if err != nil {
		log.Errorf("Failed to create split meta, error: %s", err)
	} else {
		for key, value := range splitM {
			appInfo[key] = value
		}
	}

	log.Printf("  aab infos: %v", appInfo)

	if packageName == "" {
		log.Warnf("Package name is undefined, AndroidManifest.xml package content:\n%s", out)
	}

	if versionCode == "" {
		log.Warnf("Version code is undefined, AndroidManifest.xml package content:\n%s", out)
	}

	if versionName == "" {
		log.Warnf("Version name is undefined, AndroidManifest.xml package content:\n%s", out)
	}

	// ---

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return "", fmt.Errorf("failed to get apk size, error: %s", err)
	}

	module, productFlavour, buildType := parseAppPath(pth)

	aabInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%f", fileSize),
		"app_info":        appInfo,
		"module":          module,
		"product_flavour": productFlavour,
		"build_type":      buildType,
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

	return DeployAPK(renamedUniversalAPKPath, []string{}, buildURL, token, notifyUserGroups, notifyEmails, isEnablePublicPage)
}
