package fileredactor

import (
	"reflect"
	"testing"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_ProcessFilePaths(t *testing.T) {
	deployDirPath := "/some/absolute/path/deploy_dir"
	tests := []struct {
		name      string
		input     string
		output    []string
		outputErr string
		envs      map[string]string
	}{
		{
			name:      "Empty input",
			input:     "    ",
			output:    nil,
			outputErr: "",
			envs:      nil,
		},
		{
			name: "Expand relative path",
			input: `    
/some/absolute/path/to/file.txt
file_in_deploy_dir.txt
`,
			output: []string{
				"/some/absolute/path/to/file.txt",
				"/some/absolute/path/deploy_dir/file_in_deploy_dir.txt",
			},
			outputErr: "",
			envs:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockModifier := new(mocks.PathModifier)
			mockModifier.On("AbsPath", "a_file.txt").Return(deployDirPath+"/a_file.txt", nil)
			mockModifier.On("AbsPath", "/some/absolute/path/to/file.txt").Return("/some/absolute/path/to/file.txt", nil)
			mockModifier.On("AbsPath", "file_in_deploy_dir.txt").Return(deployDirPath+"/file_in_deploy_dir.txt", nil)

			mockChecker := new(mocks.PathChecker)
			mockChecker.On("IsDirExists", mock.Anything).Return(false, nil)

			pathProcessor := NewFilePathProcessor(mockModifier, mockChecker)
			result, err := pathProcessor.ProcessFilePaths(tt.input)

			if err != nil && tt.outputErr != "" {
				assert.EqualError(t, err, tt.outputErr)
			} else if err != nil {
				t.Errorf("%s got = %v, want %v", t.Name(), err, tt.outputErr)
			}

			if !reflect.DeepEqual(result, tt.output) {
				t.Errorf("%s got = %v, want %v", t.Name(), result, tt.output)
			}
		})
	}
}
