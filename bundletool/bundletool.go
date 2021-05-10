package bundletool

import (
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/filedownloader"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
)

// Path ...
type Path string

// New ...
func New(version string) (Path, error) {
	return fetchAny(
		"https://github.com/google/bundletool/releases/download/"+version+"/bundletool-all-"+version+".jar",
		"https://github.com/google/bundletool/releases/download/"+version+"/bundletool-all.jar",
	)
}

func fetchAny(source string, fallbackSources ...string) (Path, error) {
	tmpPth, err := pathutil.NormalizedOSTempDirPath("tool")
	if err != nil {
		return "", err
	}

	downloader := filedownloader.New(retry.NewHTTPClient())

	toolPath := filepath.Join(tmpPth, "bundletool-all.jar")
	if err := downloader.GetWithFallback(toolPath, source, fallbackSources...); err != nil {
		return "", err
	}

	return Path(toolPath), err
}

// Command ...
func (p Path) Command(cmd string, args ...string) *command.Model {
	return command.New("java", append([]string{"-jar", string(p), cmd}, args...)...)
}
