package testreport

import (
	"encoding/xml"
)

// TestReport is the internal test report structure used to present test results.
type TestReport struct {
	XMLName    xml.Name    `xml:"testsuites"`
	TestSuites []TestSuite `xml:"testsuite"`
}

type TestSuite struct {
	XMLName   xml.Name   `xml:"testsuite"`
	Name      string     `xml:"name,attr"`
	Tests     int        `xml:"tests,attr"`
	Failures  int        `xml:"failures,attr"`
	Skipped   int        `xml:"skipped,attr"`
	Time      float64    `xml:"time,attr"`
	TestCases []TestCase `xml:"testcase"`
}

type TestCase struct {
	XMLName xml.Name `xml:"testcase"`
	// ConfigurationHash is used to distinguish the same test case runs,
	// performed with different build configurations (e.g., Debug vs. Release) or different devices/simulators
	ConfigurationHash string  `xml:"configuration-hash,attr"`
	Name              string  `xml:"name,attr"`
	ClassName         string  `xml:"classname,attr"`
	Time              float64 `xml:"time,attr"`
	// TODO: Currently a JUnit report's TestCase.Error and TestCase.SystemErr is merged into TestCase.Failure.Value field,
	// this way test execution errors are not distinguished from test failures.
	Failure    *Failure    `xml:"failure,omitempty"`
	Skipped    *Skipped    `xml:"skipped,omitempty"`
	Properties *Properties `xml:"properties,omitempty"`
}

type Failure struct {
	XMLName xml.Name `xml:"failure,omitempty"`
	Value   string   `xml:",chardata"`
}

type Skipped struct {
	XMLName xml.Name `xml:"skipped,omitempty"`
}

type Property struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

type Properties struct {
	XMLName  xml.Name   `xml:"properties"`
	Property []Property `xml:"property"`
}
