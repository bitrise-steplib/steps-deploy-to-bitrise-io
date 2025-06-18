package junitxml

import (
	"encoding/xml"
	"strings"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
	"github.com/pkg/errors"
)

// Converter holds data of the converter
type Converter struct {
	results []resultReader
}

func (c *Converter) Setup(_ bool) {}

// Detect return true if the test results a JUnit XML file
func (c *Converter) Detect(files []string) bool {
	c.results = nil
	for _, file := range files {
		if strings.HasSuffix(file, ".xml") || strings.HasSuffix(file, ".junit") {
			c.results = append(c.results, &fileReader{Filename: file})
		}
	}

	return len(c.results) > 0
}

// XML returns the xml content bytes
func (c *Converter) Convert() (junit.TestReport, error) {
	var xmlContent junit.TestReport

	for _, result := range c.results {
		testSuites, err := parseTestSuites(result)
		if err != nil {
			return junit.TestReport{}, err
		}

		xmlContent.TestSuites = append(xmlContent.TestSuites, testSuites...)
	}

	return xmlContent, nil
}

// merges Suites->Cases->Error and Suites->Cases->SystemErr field values into Suites->Cases->Failure field
// with 2 newlines and error category prefix
// the two newlines applied only if there is a failure message already
// this is required because our testing addon currently handles failure field properly
func regroupErrors(suites []junit.TestSuite) []junit.TestSuite {
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
				tc.Failure = &junit.Failure{
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

func parseTestSuites(result resultReader) ([]junit.TestSuite, error) {
	data, err := result.ReadAll()
	if err != nil {
		return nil, err
	}

	var testSuites junit.TestReport

	testSuitesError := xml.Unmarshal(data, &testSuites)
	if testSuitesError == nil {
		return regroupErrors(testSuites.TestSuites), nil
	}

	var testSuite junit.TestSuite
	if err := xml.Unmarshal(data, &testSuite); err != nil {
		return nil, errors.Wrap(errors.Wrap(err, string(data)), testSuitesError.Error())
	}

	return regroupErrors([]junit.TestSuite{testSuite}), nil
}
