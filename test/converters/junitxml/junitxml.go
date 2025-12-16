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
// this is required because our testing service currently handles failure field properly
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
	errors := 0
	skipped := 0

	flattenedTestCases := flattenGroupedTestCases(testSuite.TestCases)
	for _, testCase := range flattenedTestCases {
		convertedTestCase := convertTestCase(testCase)
		convertedTestSuite.TestCases = append(convertedTestSuite.TestCases, convertedTestCase)

		if convertedTestCase.Failure != nil {
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
	convertedTestSuite.Errors = errors
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

		flattenedTestCase := TestCase{
			XMLName:           testCase.XMLName,
			ConfigurationHash: testCase.ConfigurationHash,
			Name:              testCase.Name,
			ClassName:         testCase.ClassName,
		}

		for _, flakyFailure := range testCase.FlakyFailures {
			flattenedTestCase.Failure = convertToFailure(flakyFailure.Type, flakyFailure.Message, flakyFailure.SystemErr)
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

		for _, flakyError := range testCase.FlakyErrors {
			flattenedTestCase.Failure = convertToFailure(flakyError.Type, flakyError.Message, flakyError.SystemErr)
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

		for _, rerunfailure := range testCase.RerunFailures {
			flattenedTestCase.Failure = convertToFailure(rerunfailure.Type, rerunfailure.Message, rerunfailure.SystemErr)
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

		for _, rerunError := range testCase.RerunErrors {
			flattenedTestCase.Failure = convertToFailure(rerunError.Type, rerunError.Message, rerunError.SystemErr)
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

	}
	return flattenedTestCases
}

func convertToFailure(itemType, failureMessage, systemErr string) *Failure {
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

	if len(strings.TrimSpace(systemErr)) > 0 {
		if len(message) > 0 {
			message += "\n\n"
		}
		message += "System error:\n" + systemErr
	}

	if len(message) > 0 {
		return &Failure{
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
		Failure:           convertFailure(testCase.Failure),
		Error:             convertError(testCase.Error),
		Skipped:           convertSkipped(testCase.Skipped),
		SystemOut:         convertSystemOut(testCase.SystemOut),
		SystemErr:         convertSystemErr(testCase.SystemErr),
		Properties:        convertProperties(testCase.Properties),
	}

	if convertedTestCase.Error != nil {
		convertedTestCase.Failure = convertErrorToFailure(convertedTestCase.Error)
		convertedTestCase.Error = nil
	}

	enrichWithSystemOutputs(&convertedTestCase, testCase.SystemOut, testCase.SystemErr)

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

func convertSystemOut(systemOut string) *testreport.SystemOut {
	if systemOut == "" {
		return nil
	}

	return &testreport.SystemOut{
		XMLName: xml.Name{Local: "system-out"},
		Value:   systemOut,
	}
}

func convertSystemErr(systemErr string) *testreport.SystemErr {
	if systemErr == "" {
		return nil
	}

	return &testreport.SystemErr{
		XMLName: xml.Name{Local: "system-err"},
		Value:   systemErr,
	}
}

func convertSkipped(skipped *Skipped) *testreport.Skipped {
	if skipped == nil {
		return nil
	}

	var parts []string

	if len(strings.TrimSpace(skipped.Message)) > 0 {
		parts = append(parts, strings.TrimSpace(skipped.Message))
	}

	return &testreport.Skipped{
		XMLName: xml.Name{Local: "skipped"},
		Value:   strings.Join(parts, "\n\n"),
	}
}

func convertFailure(failure *Failure) *testreport.Failure {
	if failure == nil {
		return nil
	}

	var parts []string

	var attributes []string
	if len(strings.TrimSpace(failure.Type)) > 0 {
		attributes = append(attributes, failure.Type)
	}
	if len(strings.TrimSpace(failure.Message)) > 0 {
		attributes = append(attributes, failure.Message)
	}
	if len(attributes) > 0 {
		parts = append(parts, strings.Join(attributes, ": "))
	}

	if len(strings.TrimSpace(failure.Value)) > 0 {
		parts = append(parts, failure.Value)
	}

	return &testreport.Failure{
		XMLName: xml.Name{Local: "failure"},
		Value:   strings.Join(parts, "\n\n"),
	}
}

func convertError(error *Error) *testreport.Error {
	if error == nil {
		return nil
	}

	var parts []string

	var attributes []string
	if len(strings.TrimSpace(error.Type)) > 0 {
		attributes = append(attributes, error.Type)
	}
	if len(strings.TrimSpace(error.Message)) > 0 {
		attributes = append(attributes, error.Message)
	}
	if len(attributes) > 0 {
		parts = append(parts, strings.Join(attributes, ": "))
	}

	if len(strings.TrimSpace(error.Value)) > 0 {
		parts = append(parts, error.Value)
	}

	return &testreport.Error{
		XMLName: xml.Name{Local: "error"},
		Value:   strings.Join(parts, "\n\n"),
	}
}

func convertErrorToFailure(error *testreport.Error) *testreport.Failure {
	if error == nil {
		return nil
	}

	return &testreport.Failure{
		XMLName: xml.Name{Local: "failure"},
		Value:   error.Value,
	}
}

func enrichWithSystemOutputs(testCase *testreport.TestCase, systemOut, systemErr string) {
	testOutputs := []string{}

	if len(strings.TrimSpace(systemErr)) > 0 {
		testOutputs = append(testOutputs, "System error:\n"+systemErr)
	}

	if len(strings.TrimSpace(systemOut)) > 0 {
		testOutputs = append(testOutputs, "System output:\n"+systemOut)
	}

	if (len(testOutputs)) == 0 {
		return
	}

	combinedOutput := strings.Join(testOutputs, "\n\n")

	if testCase.Error != nil {
		if len(strings.TrimSpace(testCase.Error.Value)) > 0 {
			testCase.Error.Value = testCase.Error.Value + "\n\n" + combinedOutput
		} else {
			testCase.Error.Value = combinedOutput
		}
	} else if testCase.Failure != nil {
		if len(strings.TrimSpace(testCase.Failure.Value)) > 0 {
			testCase.Failure.Value = testCase.Failure.Value + "\n\n" + combinedOutput
		} else {
			testCase.Failure.Value = combinedOutput
		}
	} else if testCase.Skipped != nil {
		if len(strings.TrimSpace(testCase.Skipped.Value)) > 0 {
			testCase.Skipped.Value = testCase.Skipped.Value + "\n\n" + combinedOutput
		} else {
			testCase.Skipped.Value = combinedOutput
		}
	}
}
