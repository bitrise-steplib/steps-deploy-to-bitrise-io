package junitxml

import (
	"encoding/xml"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
)

// TestReport ...
type TestReport struct {
	XMLName    xml.Name    `xml:"testsuites"`
	TestSuites []TestSuite `xml:"testsuite"`
}

// TestSuite ...
type TestSuite struct {
	XMLName   xml.Name   `xml:"testsuite"`
	Name      string     `xml:"name,attr"`
	Tests     int        `xml:"tests,attr"`
	Failures  int        `xml:"failures,attr"`
	Skipped   int        `xml:"skipped,attr"`
	Errors    int        `xml:"errors,attr"`
	Time      float64    `xml:"time,attr"`
	TestCases []TestCase `xml:"testcase"`
}

// TestCase ...
type TestCase struct {
	XMLName           xml.Name `xml:"testcase"`
	ConfigurationHash string   `xml:"configuration-hash,attr"`
	Name              string   `xml:"name,attr"`
	ClassName         string   `xml:"classname,attr"`
	Time              float64  `xml:"time,attr"`
	Failure           *Failure `xml:"failure,omitempty"`
	Skipped           *Skipped `xml:"skipped,omitempty"`
	Error             *Error   `xml:"error,omitempty"`
	SystemErr         string   `xml:"system-err,omitempty"`
}

// Failure ...
type Failure struct {
	XMLName xml.Name `xml:"failure,omitempty"`
	Message string   `xml:"message,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

// Skipped ...
type Skipped struct {
	XMLName xml.Name `xml:"skipped,omitempty"`
}

// Error ...
type Error struct {
	XMLName xml.Name `xml:"error,omitempty"`
	Message string   `xml:"message,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

func (report TestReport) Convert() testreport.TestReport {
	testSuites := make([]testreport.TestSuite, len(report.TestSuites))
	for i, suite := range report.TestSuites {
		testCases := make([]testreport.TestCase, len(suite.TestCases))
		for j, testCase := range suite.TestCases {
			var testCaseFailure *testreport.Failure
			if testCase.Failure != nil {
				testCaseFailure = &testreport.Failure{
					Message: testCase.Failure.Message,
					Value:   testCase.Failure.Value,
				}
			}

			var testCaseSkipped *testreport.Skipped
			if testCase.Skipped != nil {
				testCaseSkipped = &testreport.Skipped{}
			}

			var testCaseError *testreport.Error
			if testCase.Error != nil {
				testCaseError = &testreport.Error{
					Message: testCase.Error.Message,
					Value:   testCase.Error.Value,
				}
			}

			testCases[j] = testreport.TestCase{
				ConfigurationHash: testCase.ConfigurationHash,
				Name:              testCase.Name,
				ClassName:         testCase.ClassName,
				Time:              testCase.Time,
				Failure:           testCaseFailure,
				Skipped:           testCaseSkipped,
				Error:             testCaseError,
				SystemErr:         testCase.SystemErr,
			}
		}
		testSuites[i] = testreport.TestSuite{
			Name:      suite.Name,
			Tests:     suite.Tests,
			Failures:  suite.Failures,
			Skipped:   suite.Skipped,
			Errors:    suite.Errors,
			Time:      suite.Time,
			TestCases: testCases,
		}
	}

	return testreport.TestReport{
		TestSuites: testSuites,
	}
}
