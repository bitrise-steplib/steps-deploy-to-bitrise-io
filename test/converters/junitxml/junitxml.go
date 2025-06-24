package junitxml

import (
	"encoding/xml"
	"strings"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
	"github.com/pkg/errors"
)

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
	var mergedReport TestReport

	for _, result := range c.results {
		report, err := parseTestReport(result)
		if err != nil {
			return testreport.TestReport{}, err
		}

		mergedReport.TestSuites = append(mergedReport.TestSuites, report.TestSuites...)
	}

	return convertTestReport(mergedReport), nil
}

func parseTestReport(result resultReader) (TestReport, error) {
	data, err := result.ReadAll()
	if err != nil {
		return TestReport{}, err
	}

	var testReport TestReport
	testReportErr := xml.Unmarshal(data, &testReport)
	if testReportErr == nil {
		return testReport, nil
	}

	var testSuite TestSuite
	testSuiteErr := xml.Unmarshal(data, &testSuite)
	if testSuiteErr == nil {
		return TestReport{TestSuites: []TestSuite{testSuite}}, nil
	}

	return TestReport{}, errors.Wrap(errors.Wrap(testSuiteErr, string(data)), testReportErr.Error())
}

// merges Suites->Cases->Error and Suites->Cases->SystemErr field values into Suites->Cases->Failure field
// with 2 newlines and error category prefix
// the two newlines applied only if there is a failure message already
// this is required because our testing addon currently handles failure field properly
func convertTestReport(report TestReport) testreport.TestReport {
	convertedReport := testreport.TestReport{
		XMLName: report.XMLName,
	}

	for _, testSuit := range report.TestSuites {
		convertedTestSuite := convertTestSuit(testSuit)
		convertedReport.TestSuites = append(convertedReport.TestSuites, convertedTestSuite)
	}

	return convertedReport
}

func convertTestSuit(testSuit TestSuite) testreport.TestSuite {
	convertedTestSuite := testreport.TestSuite{
		XMLName:  testSuit.XMLName,
		Name:     testSuit.Name,
		Tests:    testSuit.Tests,
		Failures: testSuit.Failures + testSuit.Errors,
		Skipped:  testSuit.Skipped,
		Time:     testSuit.Time,
	}

	for _, testCase := range testSuit.TestCases {
		convertedTestCase := convertTestCase(testCase)
		convertedTestSuite.TestCases = append(convertedTestSuite.TestCases, convertedTestCase)
	}

	return convertedTestSuite
}

func convertTestCase(testCase TestCase) testreport.TestCase {
	convertedTestCase := testreport.TestCase{
		XMLName:           testCase.XMLName,
		ConfigurationHash: testCase.ConfigurationHash,
		Name:              testCase.Name,
		ClassName:         testCase.ClassName,
		Time:              testCase.Time,
	}

	if testCase.Skipped != nil {
		convertedTestCase.Skipped = &testreport.Skipped{
			XMLName: testCase.Skipped.XMLName,
		}
	}

	convertedTestCase.Failure = convertErrorsToFailure(testCase.Failure, testCase.Error, testCase.SystemErr)

	return convertedTestCase
}

func convertErrorsToFailure(failure *Failure, error *Error, systemErr string) *testreport.Failure {
	var messages []string

	if failure != nil {
		if len(strings.TrimSpace(failure.Message)) > 0 {
			messages = append(messages, failure.Message)
		}

		if len(strings.TrimSpace(failure.Value)) > 0 {
			messages = append(messages, failure.Value)
		}
	}

	if error != nil {
		if len(strings.TrimSpace(error.Message)) > 0 {
			messages = append(messages, "Error message:\n"+error.Message)
		}

		if len(strings.TrimSpace(error.Value)) > 0 {
			messages = append(messages, "Error value:\n"+error.Value)
		}
	}

	if len(systemErr) > 0 {
		messages = append(messages, "System error:\n"+systemErr)
	}

	if len(messages) > 0 {
		return &testreport.Failure{
			Value: strings.Join(messages, "\n\n"),
		}
	}
	return nil
}
