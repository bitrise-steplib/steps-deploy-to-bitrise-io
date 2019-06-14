package xcresult3

import (
	"encoding/json"
	"fmt"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
)

// xcresulttoolGet performs xcrun xcresulttool get with --id flag defined if id provided and marshals the output into v.
func xcresulttoolGet(xcresultPth, id string, v interface{}) error {
	args := []string{"xcresulttool", "get", "--format", "json", "--path", xcresultPth}
	if id != "" {
		args = append(args, "--id", id)
	}

	cmd := command.New("xcrun", args...)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), out)
		}
		return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), err)
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		return err
	}
	return nil
}

// xcresulttoolExport exports a file with the given id at the given output path.
func xcresulttoolExport(xcresultPth, id, outputPth string) error {
	// xcrun xcresulttool export --path /Users/godrei/Develop/go/src/github.com/godrei/go-xcresult/_tmp/Test-Xcode11Test-2019.06.13_15-39-57-+0200.xcresult --id 0~Iiw5pvLjl5f4bgLtIhbQWMCDIhZUxfZh6Gk-SRKAFu8q0_kQkACtP3pfZsazzQstjz6IptG-fb1BahaasxL-3w== --output-path /Users/godrei/Develop/go/src/github.com/godrei/go-xcresult/_tmp/Test-Xcode11Test-2019.06.13_15-39-57-+0200.xcresult/test.jpg --type file
	args := []string{"xcresulttool", "export", "--path", xcresultPth, "--id", id, "--output-path", outputPth, "--type", "file"}
	cmd := command.New("xcrun", args...)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), out)
		}
		return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), err)
	}
	return nil
}
