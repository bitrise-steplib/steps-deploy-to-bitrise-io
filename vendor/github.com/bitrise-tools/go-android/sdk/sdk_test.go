package sdk

import (
	"os"
	"testing"

	"path/filepath"

	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestLatestBuildToolsDir(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("")
	require.NoError(t, err)

	buildToolsVersions := []string{"25.0.2", "25.0.3", "22.0.4"}
	for _, buildToolsVersion := range buildToolsVersions {
		buildToolsVersionPth := filepath.Join(tmpDir, "build-tools", buildToolsVersion)
		require.NoError(t, os.MkdirAll(buildToolsVersionPth, 0700))
	}

	sdk, err := New(tmpDir)
	require.NoError(t, err)

	latestBuildToolsDir, err := sdk.LatestBuildToolsDir()
	require.NoError(t, err)
	require.Equal(t, true, strings.Contains(latestBuildToolsDir, filepath.Join("build-tools", "25.0.3")), latestBuildToolsDir)
}

func TestLatestBuildToolPath(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("")
	require.NoError(t, err)

	buildToolsVersions := []string{"25.0.2", "25.0.3", "22.0.4"}
	for _, buildToolsVersion := range buildToolsVersions {
		buildToolsVersionPth := filepath.Join(tmpDir, "build-tools", buildToolsVersion)
		require.NoError(t, os.MkdirAll(buildToolsVersionPth, 0700))
	}

	latestBuildToolsVersions := filepath.Join(tmpDir, "build-tools", "25.0.3")
	zipalignPth := filepath.Join(latestBuildToolsVersions, "zipalign")
	require.NoError(t, fileutil.WriteStringToFile(zipalignPth, ""))

	sdk, err := New(tmpDir)
	require.NoError(t, err)

	t.Log("zipalign - exist")
	{
		pth, err := sdk.LatestBuildToolPath("zipalign")
		require.NoError(t, err)
		require.Equal(t, true, strings.Contains(pth, filepath.Join("build-tools", "25.0.3", "zipalign")), pth)
	}

	t.Log("aapt - NOT exist")
	{
		pth, err := sdk.LatestBuildToolPath("aapt")
		require.Equal(t, true, strings.Contains(err.Error(), "tool (aapt) not found at:"))
		require.Equal(t, "", pth)
	}

}
