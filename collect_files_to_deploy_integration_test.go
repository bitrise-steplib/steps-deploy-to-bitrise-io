// Integration tests that zip real files on disk: symlink and permission handling cannot be
// verified against a mocked filesystem.
package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	loggerV2 "github.com/bitrise-io/go-utils/v2/log"
	pathutil2 "github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-utils/v2/ziputil"
	"github.com/stretchr/testify/require"
)

func TestCollectFilesToDeploy_compressDir_producesZipWithContentsAndSymlink(t *testing.T) {
	srcDir := t.TempDir()
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "top.txt"), []byte("top"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub", "nested.txt"), []byte("nested"), 0644))
	require.NoError(t, os.Symlink("top.txt", filepath.Join(srcDir, "link.txt")))

	config := Config{IsCompress: true}
	files, err := collectFilesToDeploy(srcDir, config, tmpDir, newZipManager(), loggerV2.NewLogger())
	require.NoError(t, err)

	wantZip := filepath.Join(tmpDir, filepath.Base(srcDir)+".zip")
	require.Equal(t, []string{wantZip}, files)

	entries := readZip(t, wantZip)
	require.Equal(t, "top", entries["top.txt"].content)
	require.Equal(t, "nested", entries["sub/nested.txt"].content)

	// isContentOnly=true: entries are relative to srcDir, not prefixed with its basename.
	require.NotContains(t, entries, filepath.Base(srcDir)+"/top.txt")

	// The symlink must be stored as a symlink pointing at its target, not followed and inlined.
	link, ok := entries["link.txt"]
	require.True(t, ok, "symlink entry missing from archive")
	require.True(t, link.mode&os.ModeSymlink != 0, "link.txt should be stored as a symlink")
	require.Equal(t, "top.txt", link.content)
}

func TestCollectFilesToDeploy_compressDir_usesCustomZipName(t *testing.T) {
	srcDir := t.TempDir()
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0644))

	config := Config{IsCompress: true, ZipName: "custom-name"}
	files, err := collectFilesToDeploy(srcDir, config, tmpDir, newZipManager(), loggerV2.NewLogger())
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(tmpDir, "custom-name.zip")}, files)
}

func TestCollectFilesToDeploy_compressEmptyDir_returnsNothing(t *testing.T) {
	config := Config{IsCompress: true}
	files, err := collectFilesToDeploy(t.TempDir(), config, t.TempDir(), newZipManager(), loggerV2.NewLogger())
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestCollectFilesToDeploy_uncompressedDir_listsFilesSkipsSubdirs(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0755))

	config := Config{IsCompress: false}
	files, err := collectFilesToDeploy(srcDir, config, t.TempDir(), newZipManager(), loggerV2.NewLogger())
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(srcDir, "a.txt")}, files)
}

func newZipManager() *ziputil.ZipManager {
	return ziputil.NewZipManager(pathutil2.NewPathChecker())
}

type zipEntry struct {
	content string
	mode    os.FileMode
}

func readZip(t *testing.T, zipPath string) map[string]zipEntry {
	t.Helper()
	r, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer r.Close()

	entries := map[string]zipEntry{}
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(rc)
		require.NoError(t, rc.Close())
		require.NoError(t, err)
		entries[f.Name] = zipEntry{content: string(content), mode: f.Mode()}
	}
	return entries
}
