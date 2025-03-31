package model3

import (
	"fmt"
	"strings"
	"time"
)

func Convert(data *TestData) (*TestSummary, []string, error) {
	var warnings []string
	summary := TestSummary{}

	for _, testPlanNode := range data.TestNodes {
		if testPlanNode.Type != TestNodeTypeTestPlan {
			return nil, warnings, fmt.Errorf("test plan expected but got: %s", testPlanNode.Type)
		}

		testPlan := TestPlan{Name: testPlanNode.Name}

		for _, testBundleNode := range testPlanNode.Children {
			if testBundleNode.Type != TestNodeTypeUnitTestBundle && testBundleNode.Type != TestNodeTypeUITestBundle {
				return nil, warnings, fmt.Errorf("test bundle expected but got: %s", testBundleNode.Type)
			}

			testBundle := TestBundle{Name: testBundleNode.Name}

			for _, testSuiteNode := range testBundleNode.Children {
				if testSuiteNode.Type != TestNodeTypeTestSuite {
					return nil, warnings, fmt.Errorf("test suite expected but got: %s", testSuiteNode.Type)
				}

				testSuite := TestSuite{Name: testSuiteNode.Name}

				for _, testCaseNode := range testSuiteNode.Children {
					if testCaseNode.Type != TestNodeTypeTestCase {
						return nil, warnings, fmt.Errorf("test case expected but got: %s", testCaseNode.Type)
					}

					className := strings.Split(testCaseNode.Identifier, "/")[0]
					if className == "" {
						// In rare cases the identifier is an empty string so we need to use the test suite name which is the
						// same as the first part of the identifier in normal cases.
						className = testSuiteNode.Name
					}

					message, failureMessageWarnings := extractFailureMessage(testCaseNode)

					if len(failureMessageWarnings) > 0 {
						warnings = append(warnings, failureMessageWarnings...)
					}

					testCase := TestCase{
						Name:      testCaseNode.Name,
						ClassName: className,
						Time:      extractDuration(testCaseNode.Duration),
						Result:    testCaseNode.Result,
						Message:   message,
					}
					testSuite.TestCases = append(testSuite.TestCases, testCase)
				}

				testBundle.TestSuites = append(testBundle.TestSuites, testSuite)
			}

			testPlan.TestBundles = append(testPlan.TestBundles, testBundle)
		}

		summary.TestPlans = append(summary.TestPlans, testPlan)
	}

	return &summary, warnings, nil
}

func extractDuration(text string) time.Duration {
	// Duration is in the format "123.456789s"
	duration, err := time.ParseDuration(text)
	if err != nil {
		return 0
	}

	return duration
}

func extractFailureMessage(testNode TestNode) (string, []string) {
	childrenCount := len(testNode.Children)
	if childrenCount == 0 {
		return "", nil
	}

	lastNode := testNode.Children[childrenCount-1]
	if lastNode.Type == TestNodeTypeRepetition {
		return extractFailureMessage(lastNode)
	}

	var warnings []string
	failureMessage := ""

	for _, child := range testNode.Children {
		if child.Type == TestNodeTypeFailureMessage {
			// The failure message appears in the Name field and not in the Details field.
			if child.Name == "" {
				warnings = append(warnings, fmt.Sprintf("'%s' type has empty name field", child.Type))
			}
			if child.Details != "" {
				warnings = append(warnings, fmt.Sprintf("'%s' type has unexpected details field", child.Type))
			}

			failureMessage += child.Name
		}
	}

	return failureMessage, warnings
}
