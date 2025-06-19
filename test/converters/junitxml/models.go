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

func (testReport TestReport) Convert() testreport.TestReport {
	var report testreport.TestReport
	for _, suite := range testReport.TestSuites {
		testSuite := testreport.TestSuite{
			XMLName:  suite.XMLName,
			Name:     suite.Name,
			Tests:    suite.Tests,
			Failures: suite.Failures,
			Skipped:  suite.Skipped,
			Errors:   suite.Errors,
			Time:     suite.Time,
		}

		for _, tc := range suite.TestCases {
			testCase := testreport.TestCase{
				XMLName:           tc.XMLName,
				ConfigurationHash: tc.ConfigurationHash,
				Name:              tc.Name,
				ClassName:         tc.ClassName,
				Time:              tc.Time,
				SystemErr:         tc.SystemErr,
			}

			if tc.Failure != nil {
				testCase.Failure = &testreport.Failure{
					XMLName: tc.Failure.XMLName,
					Message: tc.Failure.Message,
					Value:   tc.Failure.Value,
				}
			}
			if tc.Skipped != nil {
				testCase.Skipped = &testreport.Skipped{
					XMLName: tc.Skipped.XMLName,
				}
			}
			if tc.Error != nil {
				testCase.Error = &testreport.Error{
					XMLName: tc.Failure.XMLName,
					Message: tc.Error.Message,
					Value:   tc.Error.Value,
				}
			}

			testSuite.TestCases = append(testSuite.TestCases, testCase)
		}

		report.TestSuites = append(report.TestSuites, testSuite)
	}

	return report
}
