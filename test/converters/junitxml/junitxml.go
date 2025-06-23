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

	return convertToReport(mergedReport), nil
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
func convertToReport(report TestReport) testreport.TestReport {
	convertedReport := testreport.TestReport{
		XMLName: report.XMLName,
	}

	for _, suite := range report.TestSuites {
		convertedTestSuite := testreport.TestSuite{
			XMLName:  suite.XMLName,
			Name:     suite.Name,
			Tests:    suite.Tests,
			Failures: suite.Failures,
			Skipped:  suite.Skipped,
			Errors:   0,
			Time:     suite.Time,
		}

		for _, tc := range suite.TestCases {
			convertedTestCase := testreport.TestCase{
				XMLName:           tc.XMLName,
				ConfigurationHash: tc.ConfigurationHash,
				Name:              tc.Name,
				ClassName:         tc.ClassName,
				Time:              tc.Time,
			}

			if tc.Skipped != nil {
				convertedTestCase.Skipped = &testreport.Skipped{
					XMLName: tc.Skipped.XMLName,
				}
			}

			// Converting Error and SystemErr fields into Failure field
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

			if len(messages) > 0 {
				convertedTestCase.Failure = &testreport.Failure{
					Value: strings.Join(messages, "\n\n"),
				}
			}

			convertedTestSuite.Failures += suite.Errors
			convertedTestSuite.TestCases = append(convertedTestSuite.TestCases, convertedTestCase)
		}

		convertedReport.TestSuites = append(convertedReport.TestSuites, convertedTestSuite)
	}

	return convertedReport
}
