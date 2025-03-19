package model3

import (
	"fmt"
	"strings"
	"time"
)

func Convert(data *TestData) (*TestSummary, error) {
	summary := TestSummary{}

	for _, testPlanNode := range data.TestNodes {
		if testPlanNode.Type != TestNodeTypeTestPlan {
			return nil, fmt.Errorf("test plan expected but got: %s", testPlanNode.Type)
		}

		testPlan := TestPlan{Name: testPlanNode.Name}

		for _, testBundleNode := range testPlanNode.Children {
			if testBundleNode.Type != TestNodeTypeUnitTestBundle && testBundleNode.Type != TestNodeTypeUITestBundle {
				return nil, fmt.Errorf("test bundle expected but got: %s", testBundleNode.Type)
			}

			testBundle := TestBundle{Name: testBundleNode.Name}

			for _, testSuiteNode := range testBundleNode.Children {
				if testSuiteNode.Type != TestNodeTypeTestSuite {
					return nil, fmt.Errorf("test suite expected but got: %s", testSuiteNode.Type)
				}

				testSuite := TestSuite{Name: testSuiteNode.Name}

				for _, testCaseNode := range testSuiteNode.Children {
					if testCaseNode.Type != TestNodeTypeTestCase {
						return nil, fmt.Errorf("test case expected but got: %s", testCaseNode.Type)
					}

					testCase := TestCase{
						Name:      testCaseNode.Name,
						ClassName: strings.Split(testCaseNode.Identifier, "/")[0],
						Time:      extractDuration(testCaseNode.Duration),
						Result:    testCaseNode.Result,
						Message:   extractFailureMessage(testCaseNode),
					}
					testSuite.TestCases = append(testSuite.TestCases, testCase)
				}

				testBundle.TestSuites = append(testBundle.TestSuites, testSuite)
			}

			testPlan.TestBundles = append(testPlan.TestBundles, testBundle)
		}

		summary.TestPlans = append(summary.TestPlans, testPlan)
	}

	return &summary, nil
}

func extractDuration(text string) time.Duration {
	// Duration is in the format "123.456789s"
	duration, err := time.ParseDuration(text)
	if err != nil {
		return 0
	}

	return duration
}

func extractFailureMessage(testNode TestNode) string {
	childrenCount := len(testNode.Children)
	if childrenCount == 0 {
		return ""
	}

	lastNode := testNode.Children[childrenCount-1]
	if lastNode.Type == TestNodeTypeRepetition {
		return extractFailureMessage(lastNode)
	}

	failureMessage := ""

	for _, child := range testNode.Children {
		if child.Type == TestNodeTypeFailureMessage {
			// The failure message appears in the Name field and not in the Details field.
			failureMessage += child.Name
		}
	}

	return failureMessage
}
