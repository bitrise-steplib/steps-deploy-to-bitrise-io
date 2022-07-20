package uploaders

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GivenBuildConfig_WhenUploading_ThenUsesTheConfigValues(t *testing.T) {
	// Given
	buildUrl := "build-url"
	buildAPIToken := "build-api-token"

	var receivedBuildUrl string
	var receivedBuildAPIToken string
	uploadFunction := func(path, buildURL, token string, metaData interface{}) (ArtifactURLs, error) {
		receivedBuildUrl = buildURL
		receivedBuildAPIToken = token

		return ArtifactURLs{}, nil
	}

	sut := NewPipelineUploader(uploadFunction, isDirFunction([]string{}), emptyZipFunction(), "")

	// When
	err := sut.UploadFiles("/test.txt:TEST", buildUrl, buildAPIToken)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, buildUrl, receivedBuildUrl)
	assert.Equal(t, buildAPIToken, receivedBuildAPIToken)
}

func Test_GivenIntermediateFiles_WhenUploading_ThenProcessesFilesCorrectly(t *testing.T) {
	// Given
	currentDir, err := os.Getwd()
	assert.NoError(t, err)

	tempDir := t.TempDir()

	receivedValues := map[string]metaData{}
	uploadFunction := func(path, buildURL, token string, data interface{}) (ArtifactURLs, error) {
		receivedValues[path] = data.(metaData)
		return ArtifactURLs{}, nil
	}

	directories := []string{
		"/output_folder",
		filepath.Join(currentDir, "/local/build"),
		filepath.Join(currentDir, "/folder"),
	}
	sut := NewPipelineUploader(uploadFunction, isDirFunction(directories), emptyZipFunction(), tempDir)

	tests := []struct {
		name    string
		list    string
		want    map[string]metaData
		wantErr bool
	}{
		{
			name: "Path value from an env var",
			list: "$BITRISE_IPA_PATH:BITRISE_IPA_PATH",
			want: map[string]metaData{
				filepath.Join(currentDir, "$BITRISE_IPA_PATH"): {EnvKey: "BITRISE_IPA_PATH", IsDir: false},
			},
			wantErr: false,
		},
		{
			name: "Relative file paths",
			list: "test.txt:TEST_FILE" + "\n" + "./a/folder/another_test.txt:ANOTHER_TEST_FILE",
			want: map[string]metaData{
				filepath.Join(currentDir, "test.txt"):                   {EnvKey: "TEST_FILE", IsDir: false},
				filepath.Join(currentDir, "/a/folder/another_test.txt"): {EnvKey: "ANOTHER_TEST_FILE", IsDir: false},
			},
			wantErr: false,
		},
		{
			name: "Path is an absolute path",
			list: "/test.txt:TEST_FILE",
			want: map[string]metaData{
				"/test.txt": {EnvKey: "TEST_FILE", IsDir: false},
			},
			wantErr: false,
		},
		{
			name: "Path is a directory",
			list: "/output_folder:OUTPUT_FOLDER" + "\n" + "./local/build:BUILD_DIRECTORY" + "\n" + "folder:JUST_A_FOLDER",
			want: map[string]metaData{
				filepath.Join(tempDir, "output_folder.zip"): {EnvKey: "OUTPUT_FOLDER", IsDir: true},
				filepath.Join(tempDir, "build.zip"):         {EnvKey: "BUILD_DIRECTORY", IsDir: true},
				filepath.Join(tempDir, "folder.zip"):        {EnvKey: "JUST_A_FOLDER", IsDir: true},
			},
			wantErr: false,
		},
		{
			name:    "Item is empty",
			list:    "",
			want:    map[string]metaData{},
			wantErr: false,
		},
		{
			name:    "Item has multiple separators",
			list:    "test.txt:TEST_FILE:",
			want:    map[string]metaData{},
			wantErr: true,
		},
		{
			name:    "Item does not have a path specified",
			list:    ":TEST_FILE",
			want:    map[string]metaData{},
			wantErr: true,
		},
		{
			name:    "Item does not have a value specified",
			list:    "test.txt:",
			want:    map[string]metaData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			err := sut.UploadFiles(tt.list, "", "")
			if err != nil && tt.wantErr {
				return
			}

			// Then
			assert.NoError(t, err)

			if !reflect.DeepEqual(receivedValues, tt.want) {
				t.Errorf("%s got = %v, want %v", t.Name(), receivedValues, tt.want)
			}

			receivedValues = map[string]metaData{}
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
