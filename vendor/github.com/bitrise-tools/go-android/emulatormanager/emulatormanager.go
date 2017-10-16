package emulatormanager

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"io/ioutil"
	"os/user"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-android/sdk"
)

// Model ...
type Model struct {
	binPth string
	envs   []string
}

// IsLegacyEmulator ...
func IsLegacyEmulator(androidHome string) (bool, error) {
	exist, err := pathutil.IsPathExists(filepath.Join(androidHome, "emulator", "emulator"))
	return !exist, err
}

func emulatorBinPth(androidHome string, legacyEmulator bool) (string, error) {
	emulatorDir := filepath.Join(androidHome, "emulator")
	if legacyEmulator {
		emulatorDir = filepath.Join(androidHome, "tools")
	}

	binPth := filepath.Join(emulatorDir, "emulator")
	if exist, err := pathutil.IsPathExists(binPth); err != nil {
		return "", err
	} else if !exist {
		message := "no emulator binary found in $ANDROID_HOME/emulator"
		if legacyEmulator {
			message = "no emulator binary found in $ANDROID_HOME/tools"
		}
		return "", fmt.Errorf(message)
	}
	return binPth, nil
}

func lib64Env(androidHome, hostOSName string, legacyEmulator bool) (string, error) {
	envKey := ""

	if hostOSName == "linux" {
		envKey = "LD_LIBRARY_PATH"
	} else if hostOSName == "darwin" {
		envKey = "DYLD_LIBRARY_PATH"
	} else {
		return "", fmt.Errorf("unsupported os %s", hostOSName)
	}

	emulatorDir := filepath.Join(androidHome, "emulator")
	
	if legacyEmulator {
		emulatorDir = filepath.Join(androidHome, "tools")
                libPth := filepath.Join(emulatorDir, "lib64")
	        if exist, err := pathutil.IsPathExists(libPth); err != nil {
		        return "", err
	        } else if !exist {
		        return "", fmt.Errorf("lib64 does not exist at: %s", libPth)
	        }		
		return envKey + "=" + libPth, nil
	}

	qtLibPth := filepath.Join(emulatorDir, "lib64", "qt", "lib")

	if exist, err := pathutil.IsPathExists(qtLibPth); err != nil {
		return "", err
	} else if !exist {
		return "", fmt.Errorf("qt lib does not exist at: %s", qtLibPth)
	}
	
	libPth := filepath.Join(emulatorDir, "lib64")

	return envKey + "=" + libPth + ":" + qtLibPth, nil
}

// New ...
func New(sdk sdk.AndroidSdkInterface) (*Model, error) {
	legacyEmulator, err := IsLegacyEmulator(sdk.GetAndroidHome())
	if err != nil {
		return nil, err
	}

	binPth, err := emulatorBinPth(sdk.GetAndroidHome(), legacyEmulator)
	if err != nil {
		return nil, err
	}

	envs := []string{}
	if strings.HasSuffix(binPth, "emulator") {
		env, err := lib64Env(sdk.GetAndroidHome(), runtime.GOOS, legacyEmulator)
		if err != nil {
			log.Warnf("failed to get lib64 qt lib path, error: %s", err)
		} else {
			envs = append(envs, env)
		}
	}
	if legacyEmulator {
		bashPath := "/bin/bash"
		if exist, err := pathutil.IsPathExists(bashPath); err != nil {
			log.Warnf("Failed to determine if bash binary exists, error: %s", err)
		} else if !exist {
			log.Warnf("Bash binary does not exist at: %s", bashPath)
	        }
		envs = append(envs, "SHELL=" + bashPath)
	}

	return &Model{
		binPth: binPth,
		envs:   envs,
	}, nil
}

func isAVDarmeabiv7a(name string) bool {
	user, err := user.Current()
	if err != nil {
		log.Warnf("Failed to determine AVD ABI, could not get current user, error: %s", err)
		return false
	}
	content, err := ioutil.ReadFile(user.HomeDir + "/.android/avd/" + name + ".avd/config.ini")
	if err != nil {
		log.Warnf("Failed to determine AVD ABI, could not read AVD config file, error: %s", err)
		return false		
	}
	return strings.Contains(string(content), "abi.type=armeabi-v7")
}

// StartEmulatorCommand ...
func (model Model) StartEmulatorCommand(name, skin string, options ...string) *command.Model {
	if isAVDarmeabiv7a(name) {
                model.binPth += "64-arm"
                if exist, err := pathutil.IsPathExists(model.binPth); err != nil {
		        log.Warnf("Failed to determine whether emulator binary exists, error: %s", err)
	        } else if !exist {
			log.Warnf("Emulator binary does not exist at: %s", model.binPth)
		}
        }
	
	args := []string{model.binPth, "-avd", name}
	if len(skin) == 0 {
		args = append(args, "-noskin")
	} else {
		args = append(args, "-skin", skin)
	}
	args = append(args, options...)

	commandModel := command.New(args[0], args[1:]...).AppendEnvs(model.envs...)

	return commandModel
}
