package avdmanager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-android/sdk"
	"github.com/bitrise-tools/go-android/sdkcomponent"
	"github.com/bitrise-tools/go-android/sdkmanager"
)

// Model ...
type Model struct {
	legacy bool
	binPth string
}

// IsLegacyAVDManager ...
func IsLegacyAVDManager(androidHome string) (bool, error) {
	exist, err := pathutil.IsPathExists(filepath.Join(androidHome, "tools", "bin", "avdmanager"))
	return !exist, err
}

// New ...
func New(sdk sdk.AndroidSdkInterface) (*Model, error) {
	binPth := filepath.Join(sdk.GetAndroidHome(), "tools", "bin", "avdmanager")

	legacySdk, err := sdkmanager.IsLegacySDKManager(sdk.GetAndroidHome())
	if err != nil {
		return nil, err
	}

	legacyAvd, err := IsLegacyAVDManager(sdk.GetAndroidHome())
	if err != nil {
		return nil, err
	} else if legacyAvd && legacySdk {
		binPth = filepath.Join(sdk.GetAndroidHome(), "tools", "android")
	} else if legacyAvd && !legacySdk {
		fmt.Println()
		log.Warnf("Found sdkmanager but no avdmanager, updating SDK Tools...")
		binPth = filepath.Join(sdk.GetAndroidHome(), "tools", "android")
		sdkManager, err := sdkmanager.New(sdk)
		if err == nil {
			sdkToolComponent := sdkcomponent.SDKTool{}
			updateCmd := sdkManager.InstallCommand(sdkToolComponent)
			updateCmd.SetStderr(os.Stderr)
			updateCmd.SetStdout(os.Stdout)
			if err := updateCmd.Run(); err == nil {
				legacyAvd, err = IsLegacyAVDManager(sdk.GetAndroidHome())
				if err == nil && !legacyAvd {
					log.Printf("- avdmanager successfully installed")
					binPth = filepath.Join(sdk.GetAndroidHome(), "tools", "bin", "avdmanager")
				} else if legacyAvd {
					log.Printf("- updating SDK tools was unsuccessful, continuing with legacy avd manager...")
				}
			} else {
				log.Errorf("Failed to run command:")
				fmt.Println()
				log.Donef("$ %s", updateCmd.PrintableCommandArgs())
				fmt.Println()
				log.Warnf("- continuing with legacy avd manager")
			}
		}
	}

	if exist, err := pathutil.IsPathExists(binPth); err != nil {
		return nil, err
	} else if !exist {
		return nil, fmt.Errorf("no avd manager tool found at: %s", binPth)
	}

	return &Model{
		legacy: legacyAvd,
		binPth: binPth,
	}, nil
}

// CreateAVDCommand ...
func (model Model) CreateAVDCommand(name string, systemImage sdkcomponent.SystemImage, options ...string) *command.Model {
	args := []string{"--verbose", "create", "avd", "--force", "--name", name, "--abi", systemImage.ABI}

	if model.legacy {
		args = append(args, "--target", systemImage.Platform)
	} else {
		args = append(args, "--package", systemImage.GetSDKStylePath())
	}

	if systemImage.Tag != "" && systemImage.Tag != "default" {
		args = append(args, "--tag", systemImage.Tag)
	}

	args = append(args, options...)
	return command.New(model.binPth, args...)
}
