package junitxml

import (
	"encoding/xml"
	"strings"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
	"github.com/pkg/errors"
)

// Converter holds data of the converter
type Converter struct {
	results []resultReader
}

func (c *Converter) Setup(_ bool) {}

func (c *Converter) Detect(files []string) bool {
	c.results = nil
	for _, file := range files {
		if strings.HasSuffix(file, ".xml") || strings.HasSuffix(file, ".junit") {
			c.results = append(c.results, &fileReader{Filename: file})
		}
	}

	return len(c.results) > 0
}

func (c *Converter) Convert() (testreport.TestReport, error) {
	var report TestReport

	for _, result := range c.results {
		testSuites, err := parseTestSuites(result)
		if err != nil {
			return testreport.TestReport{}, err
		}

		report.TestSuites = append(report.TestSuites, testSuites...)
	}

	return report.Convert(), nil
}

func parseTestSuites(result resultReader) ([]TestSuite, error) {
	data, err := result.ReadAll()
	if err != nil {
		return nil, err
	}

	var testSuites TestReport

	testSuitesError := xml.Unmarshal(data, &testSuites)
	if testSuitesError == nil {
		return regroupErrors(testSuites.TestSuites), nil
	}

	var testSuite TestSuite
	if err := xml.Unmarshal(data, &testSuite); err != nil {
		return nil, errors.Wrap(errors.Wrap(err, string(data)), testSuitesError.Error())
	}

	return regroupErrors([]TestSuite{testSuite}), nil
}

// merges Suites->Cases->Error and Suites->Cases->SystemErr field values into Suites->Cases->Failure field
// with 2 newlines and error category prefix
// the two newlines applied only if there is a failure message already
// this is required because our testing addon currently handles failure field properly
func regroupErrors(suites []TestSuite) []TestSuite {
	for testSuiteIndex, suite := range suites {
		for testCaseIndex, tc := range suite.TestCases {
			var messages []string

			if tc.Failure != nil {
				if len(strings.TrimSpace(tc.Failure.Message)) > 0 {
					messages = append(messages, tc.Failure.Message)
				}

				if len(strings.TrimSpace(tc.Failure.Value)) > 0 {
					messages = append(messages, tc.Failure.Value)
				}
			}

			if tc.Error != nil {
				if len(strings.TrimSpace(tc.Error.Message)) > 0 {
					messages = append(messages, "Error message:\n"+tc.Error.Message)
				}

				if len(strings.TrimSpace(tc.Error.Value)) > 0 {
					messages = append(messages, "Error value:\n"+tc.Error.Value)
				}
			}

			if len(tc.SystemErr) > 0 {
				messages = append(messages, "System error:\n"+tc.SystemErr)
			}

			tc.Error, tc.SystemErr = nil, ""
			if messages != nil {
				tc.Failure = &Failure{
					Value: strings.Join(messages, "\n\n"),
				}
			}

			suites[testSuiteIndex].Failures += suites[testSuiteIndex].Errors
			suites[testSuiteIndex].Errors = 0
			suites[testSuiteIndex].TestCases[testCaseIndex] = tc
		}
	}

	return suites
}
