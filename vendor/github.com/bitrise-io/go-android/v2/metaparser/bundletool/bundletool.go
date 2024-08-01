package bundletool

import (
	"errors"
	"fmt"
	"os/exec"
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

	downloader := filedownloader.New(retry.NewHTTPClient().StandardClient())

	toolPath := filepath.Join(tmpPth, "bundletool-all.jar")
	if err := downloader.GetWithFallback(toolPath, source, fallbackSources...); err != nil {
		return "", err
	}

	return Path(toolPath), err
}

// Command ...
func (p Path) Command(cmd string, args ...string) *command.Model {
	return command.New("java", append([]string{"-Djdk.util.zip.disableZip64ExtraFieldValidation=true", "-jar", string(p), cmd}, args...)...)
}

// Exec ...
func (p Path) Exec(cmd string, args ...string) (string, error) {
	c := p.Command(cmd, args...)

	out, err := c.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("command failed with exit status %d (%s): %s", exitErr.ExitCode(), c.PrintableCommandArgs(), out)
		}
		return "", fmt.Errorf("executing command failed (%s): %w", c.PrintableCommandArgs(), err)
	}

	return out, nil
}
