package emulatormanager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestEmulatorBinPth(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("")
	require.NoError(t, err)

	emulatorDir := filepath.Join(tmpDir, "emulator")
	require.NoError(t, os.MkdirAll(emulatorDir, 0700))

	emulatorPth := filepath.Join(emulatorDir, "emulator")
	require.NoError(t, fileutil.WriteStringToFile(emulatorPth, ""))

	t.Log("fail if no emulator bin found")
	{
		require.NoError(t, os.RemoveAll(emulatorPth))
		pth, err := emulatorBinPth(tmpDir, false)
		require.EqualError(t, err, "no emulator binary found in $ANDROID_HOME/emulator")
		require.Equal(t, "", pth)
	}
}

func TestLegacyEmulatorBinPth(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("")
	require.NoError(t, err)

	emulatorDir := filepath.Join(tmpDir, "tools")
	require.NoError(t, os.MkdirAll(emulatorDir, 0700))

	emulatorPth := filepath.Join(emulatorDir, "emulator")
	require.NoError(t, fileutil.WriteStringToFile(emulatorPth, ""))

	t.Log("fail if no emulator bin found")
	{
		require.NoError(t, os.RemoveAll(emulatorPth))
		pth, err := emulatorBinPth(tmpDir, true)
		require.EqualError(t, err, "no emulator binary found in $ANDROID_HOME/tools")
		require.Equal(t, "", pth)
	}
}

func TestLib64Env(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("")
	require.NoError(t, err)

	lib64Dir := filepath.Join(tmpDir, "emulator", "lib64")
	require.NoError(t, os.MkdirAll(lib64Dir, 0700))
	
	lib64QTLibDir := filepath.Join(tmpDir, "emulator", "lib64", "qt", "lib")
	require.NoError(t, os.MkdirAll(lib64QTLibDir, 0700))

	t.Log("lib env on linux")
	{
		env, err := lib64Env(tmpDir, "linux", false)
		require.NoError(t, err)
		require.Equal(t, true, strings.HasPrefix(env, "LD_LIBRARY_PATH="), env)
		require.Equal(t, true, strings.Contains(env, "emulator/lib64:"), env)
		require.Equal(t, true, strings.HasSuffix(env, "emulator/lib64/qt/lib"), env)
	}

	t.Log("lib env on osx")
	{
		env, err := lib64Env(tmpDir, "darwin", false)
		require.NoError(t, err)
		require.Equal(t, true, strings.HasPrefix(env, "DYLD_LIBRARY_PATH="), env)
		require.Equal(t, true, strings.Contains(env, "emulator/lib64:"), env)
		require.Equal(t, true, strings.HasSuffix(env, "emulator/lib64/qt/lib"), env)
	}

	t.Log("lib qt missing")
	{
		require.NoError(t, os.RemoveAll(lib64QTLibDir))

		env, err := lib64Env(tmpDir, "linux", false)
		require.Error(t, err)
		require.Equal(t, true, strings.HasPrefix(err.Error(), "qt lib does not exist at:"))
		require.Equal(t, "", env)

		env, err = lib64Env(tmpDir, "darwin", false)
		require.Error(t, err)
		require.Equal(t, true, strings.HasPrefix(err.Error(), "qt lib does not exist at:"))
		require.Equal(t, "", env)
	}

	t.Log("unspported os")
	{
		env, err := lib64Env(tmpDir, "windows", false)
		require.Error(t, err)
		require.EqualError(t, err, "unsupported os windows")
		require.Equal(t, "", env)
	}
}

func TestLegacyLibEnv(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("")
	require.NoError(t, err)

	lib64Dir := filepath.Join(tmpDir, "tools", "lib64")
	require.NoError(t, os.MkdirAll(lib64Dir, 0700))
	
	lib64QTLibDir := filepath.Join(tmpDir, "tools", "lib64", "qt", "lib")
	require.NoError(t, os.MkdirAll(lib64QTLibDir, 0700))

	t.Log("lib env on linux")
	{
		env, err := lib64Env(tmpDir, "linux", true)
		require.NoError(t, err)
		require.Equal(t, true, strings.HasPrefix(env, "LD_LIBRARY_PATH="), env)
		require.Equal(t, true, strings.HasSuffix(env, "tools/lib64"), env)
	}

	t.Log("lib env on osx")
	{
		env, err := lib64Env(tmpDir, "darwin", true)
		require.NoError(t, err)
		require.Equal(t, true, strings.HasPrefix(env, "DYLD_LIBRARY_PATH="), env)
		require.Equal(t, true, strings.HasSuffix(env, "tools/lib64"), env)
	}

	t.Log("unspported os")
	{
		env, err := lib64Env(tmpDir, "windows", true)
		require.Error(t, err)
		require.EqualError(t, err, "unsupported os windows")
		require.Equal(t, "", env)
	}
}
