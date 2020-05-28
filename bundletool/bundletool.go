package bundletool

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Path ...
type Path string

// New ...
func New(version string) (Path, error) {
	if len(version) == 0 {
		var err error
		version, err = getLatestReleaseVersion("google/bundletool")
		if err != nil {
			return "", fmt.Errorf("failed to fetch latest version, error: %s", err)
		}
	}

	tmpPth, err := pathutil.NormalizedOSTempDirPath("tool")
	if err != nil {
		return "", err
	}

	resp, err := getFromMultipleSources([]string{
		"https://github.com/google/bundletool/releases/download/" + version + "/bundletool-all-" + version + ".jar",
		"https://github.com/google/bundletool/releases/download/" + version + "/bundletool-all.jar",
	})
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

	toolPath := filepath.Join(tmpPth, "bundletool-all.jar")

	return Path(toolPath), ioutil.WriteFile(toolPath, d, 0777)
}

// Command ...
func (p Path) Command(cmd string, args ...string) *command.Model {
	return command.New("java", append([]string{"-jar", string(p), cmd}, args...)...)
}

func getFromMultipleSources(sources []string) (*http.Response, error) {
	for _, source := range sources {
		resp, err := http.Get(source)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("none of the sources returned 200 OK status")
}

// when you open "https://github.com/githubOwnerAndRepo/releases/latest" it will redirect you to
// "https://github.com/githubOwnerAndRepo/releases/<version>" and this redirection helps us to find version
func getLatestReleaseVersion(githubOwnerAndRepo string) (string, error) {
	resp, err := (&http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// do not follow redirect
			return http.ErrUseLastResponse
		},
	}).Get("https://github.com/" + githubOwnerAndRepo + "/releases/latest")
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	if loc := resp.Header.Get("location"); len(loc) > 0 {
		return path.Base(loc), nil
	}

	return "", fmt.Errorf("no location header found")
}
