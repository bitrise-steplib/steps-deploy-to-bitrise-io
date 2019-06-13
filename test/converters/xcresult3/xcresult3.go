package xcresult3

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
)

// ActionsInvocationRecord ...
type ActionsInvocationRecord struct {
	Actions struct {
		Values []struct {
			ActionResult struct {
				TestsRef struct {
					ID struct {
						Value string `json:"_value"`
					} `json:"id"`
				} `json:"testsRef"`
			} `json:"actionResult"`
		} `json:"_values"`
	} `json:"actions"`

	Issues struct {
		TestFailureSummaries struct {
			Values []TestFailureSummary `json:"_values"`
		} `json:"testFailureSummaries"`
	} `json:"issues"`
}

// TestFailureSummary ...
type TestFailureSummary struct {
	DocumentLocationInCreatingWorkspace struct {
		URL struct {
			Value string `json:"_value"`
		} `json:"url"`
	} `json:"documentLocationInCreatingWorkspace"`
	Message struct {
		Value string `json:"_value"`
	} `json:"message"`
	ProducingTarget struct {
		Value string `json:"_value"`
	} `json:"producingTarget"`
	TestCaseName struct {
		Value string `json:"_value"`
	} `json:"testCaseName"`
}

// ActionTestSummaryGroup ...
type ActionTestSummaryGroup struct {
	Name struct {
		Value string `json:"_value"`
	} `json:"name"`

	Identifier struct {
		Value string `json:"_value"`
	} `json:"identifier"`

	Duration struct {
		Value string `json:"_value"`
	} `json:"duration"`

	TestStatus struct {
		Value string `json:"_value"`
	} `json:"testStatus"`

	SummaryRef struct {
		ID struct {
			Value string `json:"_value"`
		} `json:"id"`
	} `json:"summaryRef"`

	Subtests struct {
		Values []ActionTestSummaryGroup `json:"_values"`
	} `json:"subtests"`
}

// ActionTestableSummary ...
type ActionTestableSummary struct {
	Name struct {
		Value string `json:"_value"`
	} `json:"name"`

	Tests struct {
		Values []ActionTestSummaryGroup `json:"_values"`
	} `json:"tests"`
}

// ActionTestPlanRunSummaries ...
type ActionTestPlanRunSummaries struct {
	Summaries struct {
		Values []struct {
			TestableSummaries struct {
				Values []ActionTestableSummary `json:"_values"`
			} `json:"testableSummaries"`
		} `json:"_values"`
	} `json:"summaries"`
}

// producingTargetAndTestCaseName unwraps the target and test case name from a given ActionTestSummaryGroup's Identifier.
func (g ActionTestSummaryGroup) producingTargetAndTestCaseName() (target string, testCase string) {
	// Xcode11TestUITests2\/testFail()
	if g.Identifier.Value != "" {
		s := strings.Split(g.Identifier.Value, "/")
		if len(s) == 2 {
			target, testCase = s[0], s[0]+"."+s[1]
		}
	}
	return
}

// testsWithStatus returns ActionTestSummaryGroup with TestStatus defined.
func (g ActionTestSummaryGroup) testsWithStatus() (tests []ActionTestSummaryGroup) {
	if g.TestStatus.Value != "" {
		tests = append(tests, g)
	}

	for _, subtest := range g.Subtests.Values {
		tests = append(tests, subtest.testsWithStatus()...)
	}
	return
}

// tests returns ActionTestSummaryGroup mapped by the container TestableSummary name.
func (s ActionTestPlanRunSummaries) tests() map[string][]ActionTestSummaryGroup {
	summaryGroupsByName := map[string][]ActionTestSummaryGroup{}

	for _, summary := range s.Summaries.Values {
		for _, testableSummary := range summary.TestableSummaries.Values {
			// test suit
			name := testableSummary.Name.Value

			var tests []ActionTestSummaryGroup
			for _, test := range testableSummary.Tests.Values {
				tests = append(tests, test.testsWithStatus()...)
			}

			summaryGroupsByName[name] = tests
		}
	}

	return summaryGroupsByName
}

func (s ActionTestPlanRunSummaries) failuresCount(testableSummaryName string) (failure int) {
	testsByCase := s.tests()
	tests := testsByCase[testableSummaryName]
	for _, test := range tests {
		if test.TestStatus.Value == "Failure" {
			failure++
		}
	}
	return
}

func (s ActionTestPlanRunSummaries) totalTime(testableSummaryName string) (time float64) {
	testsByCase := s.tests()
	tests := testsByCase[testableSummaryName]
	for _, test := range tests {
		if test.Duration.Value != "" {
			d, err := strconv.ParseFloat(test.Duration.Value, 64)
			if err == nil {
				time += d
			}
		}
	}
	return
}

// failure returns the ActionTestSummaryGroup's failure reason from the ActionsInvocationRecord.
func (r ActionsInvocationRecord) failure(test ActionTestSummaryGroup) string {
	target, testCase := test.producingTargetAndTestCaseName()
	for _, failureSummary := range r.Issues.TestFailureSummaries.Values {
		if failureSummary.ProducingTarget.Value == target && failureSummary.TestCaseName.Value == testCase {
			file, line := failureSummary.fileAndLineNumber()
			return fmt.Sprintf("%s:%s - %s", file, line, failureSummary.Message.Value)
		}
	}
	return ""
}

// fileAndLineNumber unwraps the file path and line number descriptor from a given ActionTestSummaryGroup's.
func (s TestFailureSummary) fileAndLineNumber() (file string, line string) {
	// file:\/\/\/Users\/bitrisedeveloper\/Develop\/ios\/Xcode11Test\/Xcode11TestUITests\/Xcode11TestUITests.swift#CharacterRangeLen=0&EndingLineNumber=42&StartingLineNumber=42
	if s.DocumentLocationInCreatingWorkspace.URL.Value != "" {
		i := strings.LastIndex(s.DocumentLocationInCreatingWorkspace.URL.Value, "#")
		if i > -1 {
			return s.DocumentLocationInCreatingWorkspace.URL.Value[:i], s.DocumentLocationInCreatingWorkspace.URL.Value[i:]
		}
	}
	return
}

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

// Parse parses the given xcresult file's ActionsInvocationRecord and the list of ActionTestPlanRunSummaries.
func Parse(pth string) (*ActionsInvocationRecord, []ActionTestPlanRunSummaries, error) {
	var r ActionsInvocationRecord
	if err := xcresulttoolGet(pth, "", &r); err != nil {
		return nil, nil, err
	}

	var summaries []ActionTestPlanRunSummaries
	for _, action := range r.Actions.Values {
		refID := action.ActionResult.TestsRef.ID.Value
		var s ActionTestPlanRunSummaries
		if err := xcresulttoolGet(pth, refID, &s); err != nil {
			return nil, nil, err
		}
		summaries = append(summaries, s)
	}
	return &r, summaries, nil
}
