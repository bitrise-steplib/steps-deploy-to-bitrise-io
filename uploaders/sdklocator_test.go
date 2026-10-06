package uploaders

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/stretchr/testify/require"
)

func Test_sdkLocator_LatestBuildToolPath(t *testing.T) {
	androidHome := t.TempDir()
	aaptPath := filepath.Join(androidHome, "build-tools", "34.0.0", "aapt")
	require.NoError(t, os.MkdirAll(filepath.Dir(aaptPath), 0o755))
	require.NoError(t, os.WriteFile(aaptPath, nil, 0o755))
	t.Setenv("ANDROID_HOME", androidHome)
	t.Setenv("ANDROID_SDK_ROOT", "")

	got, err := NewSDKLocator(env.NewRepository()).LatestBuildToolPath("aapt")

	require.NoError(t, err)
	wantPath, err := filepath.EvalSymlinks(aaptPath)
	require.NoError(t, err)
	require.Equal(t, wantPath, got)
}

func Test_sdkLocator_LatestBuildToolPath_withoutSDK(t *testing.T) {
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")

	locator := NewSDKLocator(env.NewRepository())

	_, err := locator.LatestBuildToolPath("aapt")
	require.Error(t, err)
}
