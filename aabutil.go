package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/androidartifact"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/bundletool"
)

// run executes a given command.
func run(tool string, args ...string) error {
	cmd := command.New(tool, args...)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		if errorutil.IsExitStatusError(err) {
			return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), out)
		}
		return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), err)
	}
	return nil
}

// generateKeystore creates a debug keystore.
func generateKeystore(tmpPth string) (string, error) {
	pth := filepath.Join(tmpPth, "debug.keystore")
	return pth, run("keytool", "-genkey", "-v",
		"-keystore", pth,
		"-storepass", "android",
		"-alias", "androiddebugkey",
		"-keypass", "android",
		"-keyalg", "RSA",
		"-keysize", "2048",
		"-validity", "10000",
		"-dname", "C=US, O=Android, CN=Android Debug",
	)
}

// buildApksArchive generates universal apks from an aab file.
func buildApksArchive(bundleTool bundletool.BundleTool, tmpPth, aabPth, keystorePath string) (string, error) {
	pth := filepath.Join(tmpPth, "universal.apks")
	tool, args := bundleTool.Command("build-apks", "--mode=universal",
		"--bundle", aabPth,
		"--output", pth,
		"--ks", keystorePath,
		"--ks-pass", "pass:android",
		"--ks-key-alias", "androiddebugkey",
		"--key-pass", "pass:android",
	)
	return pth, run(tool, args...)
}

// unzipUniversalAPKsArchive unzips an universal apks archive.
func unzipUniversalAPKsArchive(archive, destDir string) (string, error) {
	return filepath.Join(destDir, "universal.apk"), run("unzip", archive, "-d", destDir)
}

// GenerateUniversalAPK generates universal apks from an aab file.
func GenerateUniversalAPK(aabPth string) (string, error) {
	r, err := bundletool.New()
	if err != nil {
		return "", err
	}

	tmpPth, err := pathutil.NormalizedOSTempDirPath("aab-bundle")
	if err != nil {
		return "", err
	}

	keystorePath, err := generateKeystore(tmpPth)
	if err != nil {
		return "", err
	}

	apksPth, err := buildApksArchive(r, tmpPth, aabPth, keystorePath)
	if err != nil {
		return "", err
	}

	universalAPKPath, err := unzipUniversalAPKsArchive(apksPth, tmpPth)
	if err != nil {
		return "", err
	}

	renamedUniversalAPKPath := filepath.Join(tmpPth, androidartifact.UniversalAPKBase(aabPth))
	if err := os.Rename(universalAPKPath, renamedUniversalAPKPath); err != nil {
		return "", err
	}

	return renamedUniversalAPKPath, nil
}
