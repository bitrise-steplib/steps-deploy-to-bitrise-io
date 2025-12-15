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

// convertTestReport converts the JUnit XML test report to the internal test report format.
// It preserves the distinction between test failures and test execution errors:
// - Failure elements remain as Failure in the output
// - Error elements are converted to Error in the output
// - SystemErr and SystemOut are preserved in their respective fields
func convertTestReport(report TestReport) testreport.TestReport {
	convertedReport := testreport.TestReport{
		XMLName: report.XMLName,
	}

	for _, testSuite := range report.TestSuites {
		convertedTestSuite := convertTestSuite(testSuite)
		convertedReport.TestSuites = append(convertedReport.TestSuites, convertedTestSuite)
	}

	return convertedReport
}

func convertTestSuite(testSuite TestSuite) testreport.TestSuite {
	convertedTestSuite := testreport.TestSuite{
		XMLName: testSuite.XMLName,
		Name:    testSuite.Name,
		Time:    testSuite.Time,
	}

	tests := 0
	failures := 0
	skipped := 0

	flattenedTestCases := flattenGroupedTestCases(testSuite.TestCases)
	for _, testCase := range flattenedTestCases {
		convertedTestCase := convertTestCase(testCase)
		convertedTestSuite.TestCases = append(convertedTestSuite.TestCases, convertedTestCase)

		if convertedTestCase.Failure != nil || convertedTestCase.Error != nil {
			failures++
		}
		if convertedTestCase.Skipped != nil {
			skipped++
		}
		tests++
	}

	for _, childSuite := range testSuite.TestSuites {
		convertedChildSuite := convertTestSuite(childSuite)
		
		convertedTestSuite.TestSuites = append(convertedTestSuite.TestSuites, convertedChildSuite)
	}

	convertedTestSuite.Tests = tests
	convertedTestSuite.Failures = failures
	convertedTestSuite.Skipped = skipped

	return convertedTestSuite
}

func flattenGroupedTestCases(testCases []TestCase) []TestCase {
	var flattenedTestCases []TestCase
	for _, testCase := range testCases {
		flattenedTestCases = append(flattenedTestCases, testCase)

		if len(testCase.FlakyFailures) == 0 && len(testCase.FlakyErrors) == 0 &&
			len(testCase.RerunFailures) == 0 && len(testCase.RerunErrors) == 0 {
			continue
		}

		for _, flakyFailure := range testCase.FlakyFailures {
			flattenedTestCase := TestCase{
				XMLName:           testCase.XMLName,
				ConfigurationHash: testCase.ConfigurationHash,
				Name:              testCase.Name,
				ClassName:         testCase.ClassName,
				Failure:           convertToFailure(flakyFailure.Type, flakyFailure.Message),
				SystemErr:         flakyFailure.SystemErr,
			}
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

		for _, flakyError := range testCase.FlakyErrors {
			flattenedTestCase := TestCase{
				XMLName:           testCase.XMLName,
				ConfigurationHash: testCase.ConfigurationHash,
				Name:              testCase.Name,
				ClassName:         testCase.ClassName,
				Error:             convertToError(flakyError.Type, flakyError.Message),
				SystemErr:         flakyError.SystemErr,
			}
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

		for _, rerunFailure := range testCase.RerunFailures {
			flattenedTestCase := TestCase{
				XMLName:           testCase.XMLName,
				ConfigurationHash: testCase.ConfigurationHash,
				Name:              testCase.Name,
				ClassName:         testCase.ClassName,
				Failure:           convertToFailure(rerunFailure.Type, rerunFailure.Message),
				SystemErr:         rerunFailure.SystemErr,
			}
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

		for _, rerunError := range testCase.RerunErrors {
			flattenedTestCase := TestCase{
				XMLName:           testCase.XMLName,
				ConfigurationHash: testCase.ConfigurationHash,
				Name:              testCase.Name,
				ClassName:         testCase.ClassName,
				Error:             convertToError(rerunError.Type, rerunError.Message),
				SystemErr:         rerunError.SystemErr,
			}
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

	}
	return flattenedTestCases
}

func convertToFailure(itemType, failureMessage string) *Failure {
	var message string
	if len(strings.TrimSpace(itemType)) > 0 {
		message = itemType
	}
	if len(strings.TrimSpace(failureMessage)) > 0 {
		if len(message) > 0 {
			message += ": "
		}
		message += failureMessage
	}

	if len(message) > 0 {
		return &Failure{
			Value: message,
		}
	}
	return nil
}

func convertToError(itemType, errorMessage string) *Error {
	var message string
	if len(strings.TrimSpace(itemType)) > 0 {
		message = itemType
	}
	if len(strings.TrimSpace(errorMessage)) > 0 {
		if len(message) > 0 {
			message += ": "
		}
		message += errorMessage
	}

	if len(message) > 0 {
		return &Error{
			Value: message,
		}
	}
	return nil
}

func convertTestCase(testCase TestCase) testreport.TestCase {
	convertedTestCase := testreport.TestCase{
		XMLName:           testCase.XMLName,
		ConfigurationHash: testCase.ConfigurationHash,
		Name:              testCase.Name,
		ClassName:         testCase.ClassName,
		Time:              testCase.Time,
		Properties:        convertProperties(testCase.Properties),
		SystemErr:         testCase.SystemErr,
		SystemOut:         testCase.SystemOut,
	}

	if testCase.Skipped != nil {
		convertedTestCase.Skipped = &testreport.Skipped{
			XMLName: testCase.Skipped.XMLName,
		}
	}

	convertedTestCase.Failure = convertFailure(testCase.Failure)
	convertedTestCase.Error = convertError(testCase.Error)

	return convertedTestCase
}

func convertProperties(properties *Properties) *testreport.Properties {
	var convertedProperties *testreport.Properties
	if properties != nil && len(properties.Property) > 0 {
		convertedProperties = &testreport.Properties{
			XMLName: properties.XMLName,
		}
		for _, property := range properties.Property {
			convertedProperties.Property = append(convertedProperties.Property, testreport.Property{
				XMLName: property.XMLName,
				Name:    property.Name,
				Value:   property.Value,
			})
		}
	}
	return convertedProperties
}

func convertFailure(failure *Failure) *testreport.Failure {
	if failure == nil {
		return nil
	}

	var messages []string
	if len(strings.TrimSpace(failure.Message)) > 0 {
		messages = append(messages, failure.Message)
	}

	if len(strings.TrimSpace(failure.Value)) > 0 {
		messages = append(messages, failure.Value)
	}

	if len(messages) > 0 {
		return &testreport.Failure{
			XMLName: xml.Name{Local: "failure"},
			Value:   strings.Join(messages, "\n\n"),
		}
	}
	return nil
}

func convertError(error *Error) *testreport.Error {
	if error == nil {
		return nil
	}

	var messages []string
	if len(strings.TrimSpace(error.Message)) > 0 {
		messages = append(messages, error.Message)
	}

	if len(strings.TrimSpace(error.Value)) > 0 {
		messages = append(messages, error.Value)
	}

	if len(messages) > 0 {
		return &testreport.Error{
			XMLName: xml.Name{Local: "error"},
			Value:   strings.Join(messages, "\n\n"),
		}
	}
	return nil
}
