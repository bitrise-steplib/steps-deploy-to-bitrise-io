package adbmanager

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-android/sdk"
)

// Model ...
type Model struct {
	binPth string
}

// New ...
func New(sdk sdk.AndroidSdkInterface) (*Model, error) {
	binPth := filepath.Join(sdk.GetAndroidHome(), "platform-tools", "adb")
	if exist, err := pathutil.IsPathExists(binPth); err != nil {
		return nil, fmt.Errorf("failed to check if adb exist, error: %s", err)
	} else if !exist {
		return nil, fmt.Errorf("adb not exist at: %s", binPth)
	}

	return &Model{
		binPth: binPth,
	}, nil
}

// DevicesCmd ...
func (model Model) DevicesCmd() *command.Model {
	return command.New(model.binPth, "devices")
}

// IsDeviceBooted ...
func (model Model) IsDeviceBooted(serial string) (bool, error) {
	devBootCmd := command.New(model.binPth, "-s", serial, "shell", "getprop dev.bootcomplete")
	devBootOut, err := devBootCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, err
	}

	sysBootCmd := command.New(model.binPth, "-s", serial, "shell", "getprop sys.boot_completed")
	sysBootOut, err := sysBootCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, err
	}

	bootAnimCmd := command.New(model.binPth, "-s", serial, "shell", "getprop init.svc.bootanim")
	bootAnimOut, err := bootAnimCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, err
	}

	return (devBootOut == "1" && sysBootOut == "1" && bootAnimOut == "stopped"), nil
}

// UnlockDevice ...
func (model Model) UnlockDevice(serial string) error {
	keyEvent82Cmd := command.New(model.binPth, "-s", serial, "shell", "input", "82", "&")
	if err := keyEvent82Cmd.Run(); err != nil {
		return err
	}

	keyEvent1Cmd := command.New(model.binPth, "-s", serial, "shell", "input", "1", "&")
	return keyEvent1Cmd.Run()
}
