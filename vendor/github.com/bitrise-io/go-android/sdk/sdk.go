package sdk

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/hashicorp/go-version"
)

// Model ...
type Model struct {
	androidHome string
}

// AndroidSdkInterface ...
type AndroidSdkInterface interface {
	GetAndroidHome() string
}

// New ...
func New(androidHome string) (*Model, error) {
	evaluatedAndroidHome, err := filepath.EvalSymlinks(androidHome)
	if err != nil {
		return nil, err
	}

	if exist, err := pathutil.IsDirExists(evaluatedAndroidHome); err != nil {
		return nil, err
	} else if !exist {
		return nil, fmt.Errorf("android home not exists at: %s", evaluatedAndroidHome)
	}

	return &Model{androidHome: evaluatedAndroidHome}, nil
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

	if latestVersion.String() == "" {
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
