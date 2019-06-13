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
