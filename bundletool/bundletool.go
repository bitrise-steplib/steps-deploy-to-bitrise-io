package bundletool

import (
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Runner ...
type Runner struct {
	localPath string
}

// NewRunner ...
func NewRunner() (Runner, error) {
	const downloadURL = "https://github.com/google/bundletool/releases/download/0.9.0/bundletool-all-0.9.0.jar"

	tmpPth, err := pathutil.NormalizedOSTempDirPath("tool")
	if err != nil {
		return Runner{}, err
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return Runner{}, err
	}

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Runner{}, err
	}

	if err := resp.Body.Close(); err != nil {
		return Runner{}, err
	}

	toolPath := filepath.Join(tmpPth, filepath.Base(downloadURL))

	return Runner{toolPath}, ioutil.WriteFile(toolPath, d, 0777)
}

// Command ...
func (r Runner) Command(cmd string, args ...string) *command.Model {
	return command.New("java", append([]string{"-jar", r.localPath, cmd}, args...)...)
}
