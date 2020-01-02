package bundletool

import (
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Path ...
type Path string

// New ...
func New() (Path, error) {
	const downloadURL = "https://github.com/google/bundletool/releases/download/0.12.0/bundletool-all-0.12.0.jar"

	tmpPth, err := pathutil.NormalizedOSTempDirPath("tool")
	if err != nil {
		return "", err
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", err
	}

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := resp.Body.Close(); err != nil {
		return "", err
	}

	toolPath := filepath.Join(tmpPth, filepath.Base(downloadURL))

	return Path(toolPath), ioutil.WriteFile(toolPath, d, 0777)
}

// Command ...
func (p Path) Command(cmd string, args ...string) *command.Model {
	return command.New("java", append([]string{"-jar", string(p), cmd}, args...)...)
}
