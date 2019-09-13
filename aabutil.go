package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/androidartifact"
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

// GenerateUniversalAPK ...
func GenerateUniversalAPK(aabPth string) (string, error) {
	r, err := bundletool.NewRunner()
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

	// unzip `tmpDir/universal.apks` to tmpPth to have `tmpDir/universal.apk`
	log.Printf("- unzip")
	cmd := command.New("unzip", apksPth, "-d", tmpPth).SetStdout(os.Stdout).SetStderr(os.Stderr)

	log.Donef("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		return "", err
	}

	fmt.Println()
	log.Printf("- rename")

	universalAPKPath := filepath.Join(tmpPth, "universal.apk")
	renamedUniversalAPKPath := filepath.Join(tmpPth, androidartifact.UniversalAPKBase(aabPth))
	if err := os.Rename(universalAPKPath, renamedUniversalAPKPath); err != nil {
		return "", err
	}

	return renamedUniversalAPKPath, nil
}
