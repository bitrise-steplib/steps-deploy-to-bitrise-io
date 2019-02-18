package gradle

import (
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
)

// If we parse tasks that starts with lint, we will have tasks that starts
// with lintVital also. So list here each conflicting tasks. (only overlapping ones)
var conflicts = map[string][]string{
	"lint": []string{
		"lintVital",
		"lintFix",
	},
}

func getGradleOutput(projPath string, tasks ...string) (string, error) {
	c := command.New(filepath.Join(projPath, "gradlew"), tasks...)
	c.SetDir(projPath)
	return c.RunAndReturnTrimmedCombinedOutput()
}

func cleanStringSlice(in []string) (out []string) {
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return
}
