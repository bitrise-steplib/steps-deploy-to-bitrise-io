package sdk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/hashicorp/go-version"
)

// Model ...
type Model struct {
	androidHome string
}

// Environment is used to pass in environment variables used to locate Android SDK
type Environment struct {
	AndroidHome    string // ANDROID_HOME
	AndroidSDKRoot string // ANDROID_SDK_ROOT
}

// NewEnvironment gets needed environment variables
func NewEnvironment() *Environment {
	return &Environment{
		AndroidHome:    os.Getenv("ANDROID_HOME"),
		AndroidSDKRoot: os.Getenv("ANDROID_SDK_ROOT"),
	}
}

// AndroidSdkInterface ...
type AndroidSdkInterface interface {
	GetAndroidHome() string
	CmdlineToolsPath() (string, error)
}

// New creates a Model with a supplied Android SDK path
func New(androidHome string) (*Model, error) {
	evaluatedSDKRoot, err := validateAndroidSDKRoot(androidHome)
	if err != nil {
		return nil, err
	}

	return &Model{androidHome: evaluatedSDKRoot}, nil
}

// NewDefaultModel locates Android SDK based on environement variables
func NewDefaultModel(envs Environment) (*Model, error) {
	// https://developer.android.com/studio/command-line/variables#envar
	// Sets the path to the SDK installation directory.
	// ANDROID_HOME, which also points to the SDK installation directory, is deprecated.
	// If you continue to use it, the following rules apply:
	//  If ANDROID_HOME is defined and contains a valid SDK installation, its value is used instead of the value in ANDROID_SDK_ROOT.
	//  If ANDROID_HOME is not defined, the value in ANDROID_SDK_ROOT is used.
	var warnings []string
	for _, SDKdir := range []string{envs.AndroidHome, envs.AndroidSDKRoot} {
		if SDKdir == "" {
			warnings = append(warnings, "environment variable is unset or empty")
			continue
		}

		evaluatedSDKRoot, err := validateAndroidSDKRoot(SDKdir)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}

		return &Model{androidHome: evaluatedSDKRoot}, nil
	}

	return nil, fmt.Errorf("could not locate Android SDK root directory: %s", warnings)
}

func validateAndroidSDKRoot(androidSDKRoot string) (string, error) {
	evaluatedSDKRoot, err := filepath.EvalSymlinks(androidSDKRoot)
	if err != nil {
		return "", err
	}

	if exist, err := pathutil.IsDirExists(evaluatedSDKRoot); err != nil {
		return "", err
	} else if !exist {
		return "", fmt.Errorf("(%s) is not a valid Android SDK root", evaluatedSDKRoot)
	}

	return evaluatedSDKRoot, nil
}

// GetAndroidHome ...
func (model *Model) GetAndroidHome() string {
	return model.androidHome
}

// LatestBuildToolsDir ...
func (model *Model) LatestBuildToolsDir() (string, error) {
	buildTools := filepath.Join(model.androidHome, "build-tools")
	pattern := filepath.Join(buildTools, "*")

	buildToolsDirs, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	var latestVersion *version.Version
	for _, buildToolsDir := range buildToolsDirs {
		versionStr := strings.TrimPrefix(buildToolsDir, buildTools+"/")
		version, err := version.NewVersion(versionStr)
		if err != nil {
			continue
		}

		if latestVersion == nil || version.GreaterThan(latestVersion) {
			latestVersion = version
		}
	}

	if latestVersion == nil || latestVersion.String() == "" {
		return "", errors.New("failed to find latest build-tools dir")
	}

	return filepath.Join(buildTools, latestVersion.String()), nil
}

// LatestBuildToolPath ...
func (model *Model) LatestBuildToolPath(name string) (string, error) {
	buildToolsDir, err := model.LatestBuildToolsDir()
	if err != nil {
		return "", err
	}

	pth := filepath.Join(buildToolsDir, name)
	if exist, err := pathutil.IsPathExists(pth); err != nil {
		return "", err
	} else if !exist {
		return "", fmt.Errorf("tool (%s) not found at: %s", name, buildToolsDir)
	}

	return pth, nil
}

// CmdlineToolsPath locates the command-line tools directory
func (model *Model) CmdlineToolsPath() (string, error) {
	toolPaths := []string{
		filepath.Join(model.GetAndroidHome(), "cmdline-tools", "latest", "bin"),
		filepath.Join(model.GetAndroidHome(), "cmdline-tools", "*", "bin"),
		filepath.Join(model.GetAndroidHome(), "tools", "bin"),
		filepath.Join(model.GetAndroidHome(), "tools"), // legacy
	}

	var warnings []string
	for _, dirPattern := range toolPaths {
		matches, err := filepath.Glob(dirPattern)
		if err != nil {
			return "", fmt.Errorf("failed to locate Android command-line tools directory, invalid patterns specified (%s): %s", toolPaths, err)
		}

		if len(matches) == 0 {
			continue
		}

		sdkmanagerPath := matches[0]
		if exists, err := pathutil.IsDirExists(sdkmanagerPath); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to validate path (%s): %v", sdkmanagerPath, err))
			continue
		} else if !exists {
			warnings = append(warnings, "path (%s) does not exist or it is not a directory")
			continue
		}

		return sdkmanagerPath, nil
	}

	return "", fmt.Errorf("failed to locate Android command-line tools directory on paths (%s), warnings: %s", toolPaths, warnings)
}
