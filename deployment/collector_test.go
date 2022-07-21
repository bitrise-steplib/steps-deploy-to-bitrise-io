package deployment

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GivenIntermediateFiles_WhenProcessing_ThenConvertsCorrectly(t *testing.T) {
	tempDir := t.TempDir()
	currentDir, err := os.Getwd()
	assert.NoError(t, err)

	directories := []string{
		"/output_folder",
		filepath.Join(currentDir, "/local/build"),
		filepath.Join(currentDir, "/folder"),
	}

	tests := []struct {
		name    string
		list    string
		want    []DeployableItem
		wantErr bool
	}{
		{
			name: "Path value from an env var",
			list: "$BITRISE_IPA_PATH:BITRISE_IPA_PATH",
			want: []DeployableItem{
				{
					Path: filepath.Join(currentDir, "$BITRISE_IPA_PATH"),
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "BITRISE_IPA_PATH",
						IsDir:  false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Relative file paths",
			list: "test.txt:TEST_FILE" + "\n" + "./a/folder/another_test.txt:ANOTHER_TEST_FILE",
			want: []DeployableItem{
				{
					Path: filepath.Join(currentDir, "test.txt"),
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "TEST_FILE",
						IsDir:  false,
					},
				},
				{
					Path: filepath.Join(currentDir, "/a/folder/another_test.txt"),
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "ANOTHER_TEST_FILE",
						IsDir:  false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Path is an absolute path",
			list: "/test.txt:TEST_FILE",
			want: []DeployableItem{
				{
					Path: "/test.txt",
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "TEST_FILE",
						IsDir:  false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Path is a directory",
			list: "/output_folder:OUTPUT_FOLDER" + "\n" + "./local/build:BUILD_DIRECTORY" + "\n" + "folder:JUST_A_FOLDER",
			want: []DeployableItem{
				{
					Path: filepath.Join(tempDir, "output_folder.zip"),
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "OUTPUT_FOLDER",
						IsDir:  true,
					},
				},
				{
					Path: filepath.Join(tempDir, "build.zip"),
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "BUILD_DIRECTORY",
						IsDir:  true,
					},
				},
				{
					Path: filepath.Join(tempDir, "folder.zip"),
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "JUST_A_FOLDER",
						IsDir:  true,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Item is empty",
			list:    "",
			want:    []DeployableItem{},
			wantErr: false,
		},
		{
			name:    "Item has multiple separators",
			list:    "test.txt:TEST_FILE:",
			want:    []DeployableItem{},
			wantErr: true,
		},
		{
			name:    "Item does not have a path specified",
			list:    ":TEST_FILE",
			want:    []DeployableItem{},
			wantErr: true,
		},
		{
			name:    "Item does not have a value specified",
			list:    "test.txt:",
			want:    []DeployableItem{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewCollector(isDirFunction(directories), emptyZipFunction(), tempDir)
			deployableItems, err := collector.FinalListOfDeployableItems([]string{}, tt.list)

			if err != nil && tt.wantErr {
				return
			}

			assert.NoError(t, err)

			if !reflect.DeepEqual(deployableItems, tt.want) {
				t.Errorf("%s got = %v, want %v", t.Name(), deployableItems, tt.want)
			}
		})
	}
}

func Test_GivenDeployFiles_WhenIntermediateFilesSpecified_ThenMergesThem(t *testing.T) {
	tempDir := t.TempDir()
	currentDir, err := os.Getwd()
	assert.NoError(t, err)

	directories := []string{
		"/output_folder",
		filepath.Join(currentDir, "/local/build"),
		filepath.Join(currentDir, "/folder"),
	}

	tests := []struct {
		name              string
		deployFiles       []string
		intermediateFiles string
		want              []DeployableItem
	}{
		{
			name:              "File can be deployable and intermediate at the same time",
			deployFiles:       []string{"/ios-app.ipa"},
			intermediateFiles: "/ios-app.ipa:IPA_PATH",
			want: []DeployableItem{
				{
					Path: "/ios-app.ipa",
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "IPA_PATH",
						IsDir:  false,
					},
				},
			},
		},
		{
			name: "Absolute and relative paths are compatible",
			deployFiles: []string{
				filepath.Join(currentDir, "/test.xcresult"),
			},
			intermediateFiles: "test.xcresult:RESULT_PATH",
			want: []DeployableItem{
				{
					Path: filepath.Join(currentDir, "/test.xcresult"),
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "RESULT_PATH",
						IsDir:  false,
					},
				},
			},
		},
		{
			name: "Deploy and intermediate file lists are merged",
			deployFiles: []string{
				filepath.Join(currentDir, "/test.ipa"),
				filepath.Join(currentDir, "/test.xcresult"),
			},
			intermediateFiles: "test.xcresult:RESULT_PATH" + "\n" + "./folder/secret.txt:SECRET_FILE",
			want: []DeployableItem{
				{
					Path:         filepath.Join(currentDir, "/test.ipa"),
					PipelineMeta: nil,
				},
				{
					Path: filepath.Join(currentDir, "/test.xcresult"),
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "RESULT_PATH",
						IsDir:  false,
					},
				},
				{
					Path: filepath.Join(currentDir, "/folder/secret.txt"),
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "SECRET_FILE",
						IsDir:  false,
					},
				},
			},
		},
		{
			name:              "Empty deploy file list is handled",
			deployFiles:       []string{},
			intermediateFiles: "/test.xcresult:RESULT_PATH",
			want: []DeployableItem{
				{
					Path: "/test.xcresult",
					PipelineMeta: &IntermediateFileMetaData{
						EnvKey: "RESULT_PATH",
						IsDir:  false,
					},
				},
			},
		},
		{
			name:              "Empty intermediate file list is handled",
			deployFiles:       []string{"/test.xcresult"},
			intermediateFiles: "",
			want: []DeployableItem{
				{
					Path:         "/test.xcresult",
					PipelineMeta: nil,
				},
			},
		},
		{
			name:              "Empty lists can be merged",
			deployFiles:       []string{},
			intermediateFiles: "",
			want:              []DeployableItem{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewCollector(isDirFunction(directories), emptyZipFunction(), tempDir)
			deployableItems, err := collector.FinalListOfDeployableItems(tt.deployFiles, tt.intermediateFiles)

			assert.NoError(t, err)

			if !reflect.DeepEqual(deployableItems, tt.want) {
				t.Errorf("%s got = %v, want %v", t.Name(), deployableItems, tt.want)
			}
		})
	}
}

// Helpers

func isDirFunction(directoryEntries []string) IsDirFunction {
	return func(path string) (bool, error) {
		for _, entry := range directoryEntries {
			if entry == path {
				return true, nil
			}
		}

		return false, nil
	}
}

func emptyZipFunction() ZipDirFunction {
	return func(sourceDirPth, destinationZipPth string, isContentOnly bool) error {
		return nil
	}
}
