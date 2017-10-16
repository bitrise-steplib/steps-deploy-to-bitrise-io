package sdkmanager

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-android/sdk"
	"github.com/bitrise-tools/go-android/sdkcomponent"
)

// Model ...
type Model struct {
	androidHome string
	legacy      bool
	binPth      string
}

// IsLegacySDKManager ...
func IsLegacySDKManager(androidHome string) (bool, error) {
	exist, err := pathutil.IsPathExists(filepath.Join(androidHome, "tools", "bin", "sdkmanager"))
	return !exist, err
}

// New ...
func New(sdk sdk.AndroidSdkInterface) (*Model, error) {
	binPth := filepath.Join(sdk.GetAndroidHome(), "tools", "bin", "sdkmanager")

	legacy, err := IsLegacySDKManager(sdk.GetAndroidHome())
	if err != nil {
		return nil, err
	} else if legacy {
		binPth = filepath.Join(sdk.GetAndroidHome(), "tools", "android")
	}

	if exist, err := pathutil.IsPathExists(binPth); err != nil {
		return nil, err
	} else if !exist {
		return nil, fmt.Errorf("no sdk manager tool found at: %s", binPth)
	}

	return &Model{
		androidHome: sdk.GetAndroidHome(),
		legacy:      legacy,
		binPth:      binPth,
	}, nil
}

// IsLegacySDK ...
func (model Model) IsLegacySDK() bool {
	return model.legacy
}

// IsInstalled ...
func (model Model) IsInstalled(component sdkcomponent.Model) (bool, error) {
	relPth := component.InstallPathInAndroidHome()
	indicatorFile := component.InstallationIndicatorFile()
	installPth := filepath.Join(model.androidHome, relPth)

	if indicatorFile != "" {
		installPth = filepath.Join(installPth, indicatorFile)
	}
	return pathutil.IsPathExists(installPth)
}

// InstallCommand ...
func (model Model) InstallCommand(component sdkcomponent.Model) *command.Model {
	if model.legacy {
		return command.New(model.binPth, "update", "sdk", "--no-ui", "--all", "--filter", component.GetLegacySDKStylePath())
	}
	return command.New(model.binPth, component.GetSDKStylePath())
}
