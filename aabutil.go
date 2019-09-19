package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/androidartifact"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/bundletool"
)

// handleError creates error with layout: `<cmd> failed (status: <status_code>): <cmd output>`.
func handleError(cmd, out string, err error) error {
	if err == nil {
		return nil
	}

	msg := fmt.Sprintf("%s failed", cmd)
	if status, exitCodeErr := errorutil.CmdExitCodeFromError(err); exitCodeErr == nil {
		msg += fmt.Sprintf(" (status: %d)", status)
	}
	if len(out) > 0 {
		msg += fmt.Sprintf(": %s", out)
	}
	return errors.New(msg)
}

// run executes a given command.
func run(cmd *command.Model) error {
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	return handleError(cmd.PrintableCommandArgs(), out, err)
}

// generateKeystore creates a debug keystore.
func generateKeystore(tmpPth string) (string, error) {
	pth := filepath.Join(tmpPth, "debug.keystore")
	return pth, run(command.New("keytool", "-genkey", "-v",
		"-keystore", pth,
		"-storepass", "android",
		"-alias", "androiddebugkey",
		"-keypass", "android",
		"-keyalg", "RSA",
		"-keysize", "2048",
		"-validity", "10000",
		"-dname", "C=US, O=Android, CN=Android Debug",
	))
}

// buildApksArchive generates universal apks from an aab file.
func buildApksArchive(bundleTool bundletool.Path, tmpPth, aabPth, keystorePath string) (string, error) {
	pth := filepath.Join(tmpPth, "universal.apks")
	return pth, run(bundleTool.Command("build-apks", "--mode=universal",
		"--bundle", aabPth,
		"--output", pth,
		"--ks", keystorePath,
		"--ks-pass", "pass:android",
		"--ks-key-alias", "androiddebugkey",
		"--key-pass", "pass:android",
	))
}

// unzipUniversalAPKsArchive unzips an universal apks archive.
func unzipUniversalAPKsArchive(archive, destDir string) (string, error) {
	return filepath.Join(destDir, "universal.apk"), run(command.New("unzip", archive, "-d", destDir))
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
	return renamedUniversalAPKPath, os.Rename(universalAPKPath, renamedUniversalAPKPath)
}
