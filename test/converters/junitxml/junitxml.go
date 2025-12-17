package junitxml

import (
	"encoding/xml"
	"strings"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
	errorPkg "github.com/pkg/errors"
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

	return TestReport{}, errorPkg.Wrap(errorPkg.Wrap(testSuiteErr, string(data)), testReportErr.Error())
}

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
		XMLName:    testSuite.XMLName,
		Name:       testSuite.Name,
		Time:       testSuite.Time,
		Assertions: testSuite.Assertions,
		Timestamp:  testSuite.Timestamp,
		File:       testSuite.File,
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
		if convertedTestCase.Error != nil {
			errors++
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
			flattenedTestCase.Failure = &Failure{
				Type:    flakyFailure.Type,
				Message: flakyFailure.Message,
				Value:   flakyFailure.Value,
			}
			flattenedTestCase.SystemErr = flakyFailure.SystemErr
			flattenedTestCase.SystemOut = flakyFailure.SystemOut
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

		flattenedTestCase.Failure = nil
		for _, flakyError := range testCase.FlakyErrors {
			flattenedTestCase.Error = &Error{
				Type:    flakyError.Type,
				Message: flakyError.Message,
				Value:   flakyError.Value,
			}
			flattenedTestCase.SystemErr = flakyError.SystemErr
			flattenedTestCase.SystemOut = flakyError.SystemOut
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

		flattenedTestCase.Error = nil
		for _, rerunfailure := range testCase.RerunFailures {
			flattenedTestCase.Failure = &Failure{
				Type:    rerunfailure.Type,
				Message: rerunfailure.Message,
				Value:   rerunfailure.Value,
			}
			flattenedTestCase.SystemErr = rerunfailure.SystemErr
			flattenedTestCase.SystemOut = rerunfailure.SystemOut
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

		flattenedTestCase.Failure = nil
		for _, rerunError := range testCase.RerunErrors {
			flattenedTestCase.Error = &Error{
				Type:    rerunError.Type,
				Message: rerunError.Message,
				Value:   rerunError.Value,
			}
			flattenedTestCase.SystemErr = rerunError.SystemErr
			flattenedTestCase.SystemOut = rerunError.SystemOut
			flattenedTestCases = append(flattenedTestCases, flattenedTestCase)
		}

	}
	return flattenedTestCases
}

func convertTestCase(testCase TestCase) testreport.TestCase {
	convertedTestCase := testreport.TestCase{
		XMLName:           testCase.XMLName,
		ConfigurationHash: testCase.ConfigurationHash,
		Name:              testCase.Name,
		ClassName:         testCase.ClassName,
		Time:              testCase.Time,
		Assertions:        testCase.Assertions,
		File:              testCase.File,
		Line:              testCase.Line,
		Failure:           convertFailure(testCase.Failure),
		Error:             convertError(testCase.Error),
		Skipped:           convertSkipped(testCase.Skipped),
		Properties:        convertProperties(testCase.Properties),
		SystemOut:         convertSystemOut(testCase.SystemOut),
		SystemErr:         convertSystemErr(testCase.SystemErr),
	}

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
	if len(strings.TrimSpace(systemOut)) == 0 {
		return nil
	}

	return &testreport.SystemOut{
		XMLName: xml.Name{Local: "system-out"},
		Value:   strings.TrimSpace(systemOut),
	}
}

func convertSystemErr(systemErr string) *testreport.SystemErr {
	if len(strings.TrimSpace(systemErr)) == 0 {
		return nil
	}

	return &testreport.SystemErr{
		XMLName: xml.Name{Local: "system-err"},
		Value:   strings.TrimSpace(systemErr),
	}
}

func convertSkipped(skipped *Skipped) *testreport.Skipped {
	if skipped == nil {
		return nil
	}

	return &testreport.Skipped{
		XMLName: xml.Name{Local: "skipped"},
		Message: skipped.Message,
		Value:   strings.TrimSpace(skipped.Value),
	}
}

func convertFailure(failure *Failure) *testreport.Failure {
	if failure == nil {
		return nil
	}

	return &testreport.Failure{
		XMLName: xml.Name{Local: "failure"},
		Type:    failure.Type,
		Message: failure.Message,
		Value:   strings.TrimSpace(failure.Value),
	}
}

func convertError(error *Error) *testreport.Error {
	if error == nil {
		return nil
	}

	return &testreport.Error{
		XMLName: xml.Name{Local: "error"},
		Type:    error.Type,
		Message: error.Message,
		Value:   strings.TrimSpace(error.Value),
	}
}
