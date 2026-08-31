package main

import (
	"archive/zip"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-utils/v2/ziputil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_generateUrlOutputWithTemplate(t *testing.T) {
	defaultTemplate := "{{range $index, $element := .}}{{if $index}}|{{end}}{{$element.File}}=>{{$element.URL}}{{end}}"
	temp := template.New("test")
	temp, err := temp.Parse(defaultTemplate)
	if err != nil {
		t.Errorf("error during parsing: %s", err)
	}
	tests := []struct {
		name         string
		pages        []PublicInstallPage
		maxEnvLength int
		want         string
		wantWarn     bool
	}{
		{
			name:         "Empty list gives empty value",
			pages:        []PublicInstallPage{},
			maxEnvLength: 100,
			want:         "",
		},
		{
			name: "All content fits the variable",
			pages: []PublicInstallPage{
				{
					File: "Foo",
					URL:  "Bar",
				},
			},
			maxEnvLength: 100,
			want:         "Foo=>Bar",
		},
		{
			name: "One item doesn't fit",
			pages: []PublicInstallPage{
				{
					File: "Foo",
					URL:  "Bar",
				},
				{
					File: "Baz",
					URL:  "Qux",
				},
			},
			maxEnvLength: 10,
			want:         "Foo=>Bar",
			wantWarn:     true,
		},
		{
			name: "Multiple items doesn't fit",
			pages: []PublicInstallPage{
				{
					File: "Foo",
					URL:  "Bar",
				},
				{
					File: "Baz",
					URL:  "Qux",
				},
				{
					File: "Apple",
					URL:  "Pear",
				},
				{
					File: "Peach",
					URL:  "Grapes",
				},
			},
			maxEnvLength: 20,
			want:         "Foo=>Bar|Baz=>Qux",
			wantWarn:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotWarn, err := applyTemplateWithMaxSize(temp, tt.pages, tt.maxEnvLength)
			if err != nil {
				t.Errorf("applyTemplateWithMaxSize() error: %s", err)
			}
			if gotWarn != tt.wantWarn {
				t.Errorf("applyTemplateWithMaxSize() warning = %v, want %v", gotWarn, tt.wantWarn)
			}
			if got != tt.want {
				t.Errorf("applyTemplateWithMaxSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadConcurrency(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   int
	}{
		{
			name: "Zero value",
			config: Config{
				UploadConcurrency: "0",
			},
			want: 1,
		},
		{
			name: "Negative value",
			config: Config{
				UploadConcurrency: "-1",
			},
			want: 1,
		},
		{
			name: "In range value",
			config: Config{
				UploadConcurrency: "3",
			},
			want: 3,
		},
		{
			name: "Too large value",
			config: Config{
				UploadConcurrency: "100",
			},
			want: 20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, determineConcurrency(tt.config))
		})
	}
}

func Test_validateUserGroups(t *testing.T) {
	tests := []struct {
		name          string
		userGroupsStr string
		logger        func() log.Logger
		wantErr       error
	}{
		{
			name:          "Empty user groups",
			userGroupsStr: "",
			logger:        func() log.Logger { return mocks.NewLogger(t) },
		},
		{
			name:          "Valid user groups",
			userGroupsStr: strings.Join(validUserGroups, ","),
			logger:        func() log.Logger { return mocks.NewLogger(t) },
		},
		{
			name:          "Valid user groups with capital letter",
			userGroupsStr: "Testers",
			logger: func() log.Logger {
				logger := mocks.NewLogger(t)
				logger.On("Warnf", "User group %s is accepted by the backend, but it is not the recommended value. Please use one of the following values: %s", "Testers", strings.Join(validUserGroups, ", "))
				return logger
			},
		},
		{
			name:          "Accepted user groups",
			userGroupsStr: strings.Join(acceptedUserGroups, ","),
			logger: func() log.Logger {
				logger := mocks.NewLogger(t)
				// Expect a warning for each accepted but not valid user group
				for _, userGroup := range acceptedUserGroups {
					if !slices.Contains(validUserGroups, userGroup) {
						logger.On("Warnf", "User group %s is accepted by the backend, but it is not the recommended value. Please use one of the following values: %s", userGroup, strings.Join(validUserGroups, ", "))
					}
				}
				return logger
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr == nil {
				require.NoError(t, validateUserGroups(tt.userGroupsStr, tt.logger()))
			} else {
				require.EqualError(t, validateUserGroups(tt.userGroupsStr, tt.logger()), tt.wantErr.Error())
			}
		})
	}
}

func TestCollectFilesToDeploy_compressDir_producesZipWithContentsAndSymlink(t *testing.T) {
	// Uses a real filesystem: symlink and permission handling cannot be verified against a mock.
	srcDir := t.TempDir()
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "top.txt"), []byte("top"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub", "nested.txt"), []byte("nested"), 0644))
	require.NoError(t, os.Symlink("top.txt", filepath.Join(srcDir, "link.txt")))

	config := Config{IsCompress: true}
	files, err := collectFilesToDeploy(srcDir, config, tmpDir, newZipManager(), pathutil.NewPathChecker(), log.NewLogger())
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
	files, err := collectFilesToDeploy(srcDir, config, tmpDir, newZipManager(), pathutil.NewPathChecker(), log.NewLogger())
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(tmpDir, "custom-name.zip")}, files)
}

func TestCollectFilesToDeploy_compressEmptyDir_returnsNothing(t *testing.T) {
	config := Config{IsCompress: true}
	files, err := collectFilesToDeploy(t.TempDir(), config, t.TempDir(), newZipManager(), pathutil.NewPathChecker(), log.NewLogger())
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestCollectFilesToDeploy_uncompressedDir_listsFilesSkipsSubdirs(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0755))

	config := Config{IsCompress: false}
	files, err := collectFilesToDeploy(srcDir, config, t.TempDir(), newZipManager(), pathutil.NewPathChecker(), log.NewLogger())
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(srcDir, "a.txt")}, files)
}

func newZipManager() *ziputil.ZipManager {
	return ziputil.NewZipManager(pathutil.NewPathChecker())
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
