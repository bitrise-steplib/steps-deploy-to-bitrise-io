package xcresult3

import (
	"fmt"
	"strconv"
	"strings"
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

	Issues Issues `json:"issues"`
}

// URL ...
type URL struct {
	Value string `json:"_value"`
}

// DocumentLocationInCreatingWorkspace ...
type DocumentLocationInCreatingWorkspace struct {
	URL URL `json:"url"`
}

// ProducingTarget ...
type ProducingTarget struct {
	Value string `json:"_value"`
}

// TestCaseName ...
type TestCaseName struct {
	Value string `json:"_value"`
}

// Message ...
type Message struct {
	Value string `json:"_value"`
}

// TestFailureSummary ...
type TestFailureSummary struct {
	DocumentLocationInCreatingWorkspace DocumentLocationInCreatingWorkspace `json:"documentLocationInCreatingWorkspace"`
	Message                             Message                             `json:"message"`
	ProducingTarget                     ProducingTarget                     `json:"producingTarget"`
	TestCaseName                        TestCaseName                        `json:"testCaseName"`
}

// TestFailureSummaries ...
type TestFailureSummaries struct {
	Values []TestFailureSummary `json:"_values"`
}

// Issues ...
type Issues struct {
	TestFailureSummaries TestFailureSummaries `json:"testFailureSummaries"`
}

// TestStatus ...
type TestStatus struct {
	Value string `json:"_value"`
}

// Subtests ...
type Subtests struct {
	Values []ActionTestSummaryGroup `json:"_values"`
}

// Duration ...
type Duration struct {
	Value string `json:"_value"`
}

// Identifier ...
type Identifier struct {
	Value string `json:"_value"`
}

// ActionTestSummaryGroup ...
type ActionTestSummaryGroup struct {
	Name Name `json:"name"`

	Identifier Identifier `json:"identifier"`

	Duration Duration `json:"duration"`

	TestStatus TestStatus `json:"testStatus"`

	SummaryRef struct {
		ID struct {
			Value string `json:"_value"`
		} `json:"id"`
	} `json:"summaryRef"`

	Subtests Subtests `json:"subtests"`
}

// Tests ...
type Tests struct {
	Values []ActionTestSummaryGroup `json:"_values"`
}

// Name ...
type Name struct {
	Value string `json:"_value"`
}

// ActionTestableSummary ...
type ActionTestableSummary struct {
	Name Name `json:"name"`

	Tests Tests `json:"tests"`
}

// TestableSummaries ...
type TestableSummaries struct {
	Values []ActionTestableSummary `json:"_values"`
}

// Summary ...
type Summary struct {
	TestableSummaries TestableSummaries `json:"testableSummaries"`
}

// Summaries ...
type Summaries struct {
	Values []Summary `json:"_values"`
}

// ActionTestPlanRunSummaries ...
type ActionTestPlanRunSummaries struct {
	Summaries Summaries `json:"summaries"`
}

// producingTargetAndTestCaseName unwraps the target and test case name from a given ActionTestSummaryGroup's Identifier.
func (g ActionTestSummaryGroup) producingTargetAndTestCaseName() (target string, testCase string) {
	// Xcode11TestUITests2/testFail()
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
		if i > -1 && i+1 < len(s.DocumentLocationInCreatingWorkspace.URL.Value) {
			return s.DocumentLocationInCreatingWorkspace.URL.Value[:i], s.DocumentLocationInCreatingWorkspace.URL.Value[i+1:]
		}
	}
	return
}
