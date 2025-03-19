package xcresult3

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
)

func isXcresulttoolAvailable() bool {
	if _, err := exec.LookPath("xcrun"); err != nil {
		return false
	}
	return command.New("xcrun", "--find", "xcresulttool").Run() == nil
}

func supportsNewExtractionMethods() (bool, error) {
	args := []string{"xcresulttool", "version"}
	cmd := command.New("xcrun", args...)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return true, fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), out)
		}
		return true, fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), err)
	}
	// xcresulttool version 23025, format version 3.53 (current)
	versionRegexp := regexp.MustCompile("xcresulttool version ([0-9]+)")

	matches := versionRegexp.FindStringSubmatch(out)
	if len(matches) < 2 {
		return true, fmt.Errorf("no version matches found in output: %s", out)
	}

	version, err := strconv.Atoi(matches[1])
	if err != nil {
		return true, fmt.Errorf("failed to convert version: %s", matches[1])
	}

	return version >= 23_021, nil // Xcode 16 beta3 has version 23021

	//fmt.Println("xcresulttool version:", version)
	//
	//return false, nil
}

// xcresulttoolGet performs xcrun xcresulttool get with --id flag defined if id provided and marshals the output into v.
func xcresulttoolGet(xcresultPth, id string, v interface{}) error {
	args := []string{"xcresulttool", "get"}

	supportsNewMethod, err := supportsNewExtractionMethods()
	if err != nil {
		return err
	}

	if supportsNewMethod {
		args = append(args, "test-results", "tests")
	} else {
		args = append(args, "--format", "json")
	}

	args = append(args, "--path", xcresultPth)

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
	args := []string{"xcresulttool", "export"}

	supportsNewMethod, err := supportsNewExtractionMethods()
	if err != nil {
		return err
	}

	if supportsNewMethod {
		args = append(args, "attachments")
	} else {
		args = append(args, "--type", "file")
	}

	args = append(args, "--path", xcresultPth)
	args = append(args, "--output-path", outputPth)

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
	return nil
}
