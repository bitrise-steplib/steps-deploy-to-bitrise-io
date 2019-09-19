package bundletool

import (
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/bitrise-io/go-utils/pathutil"
)

// BundleTool ...
type BundleTool string

// New ...
func New() (BundleTool, error) {
	const downloadURL = "https://github.com/google/bundletool/releases/download/0.9.0/bundletool-all-0.9.0.jar"

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

	return BundleTool(toolPath), ioutil.WriteFile(toolPath, d, 0777)
}

// Command ...
func (r BundleTool) Command(cmd string, args ...string) (string, []string) {
	return "java", append([]string{"-jar", string(r), cmd}, args...)
}
