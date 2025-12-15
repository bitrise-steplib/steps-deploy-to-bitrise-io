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
	XMLName    xml.Name    `xml:"testsuite"`
	Name       string      `xml:"name,attr"`
	Tests      int         `xml:"tests,attr"`
	Failures   int         `xml:"failures,attr"`
	Errors     int         `xml:"errors,attr"`
	Skipped    int         `xml:"skipped,attr"`
	Time       float64     `xml:"time,attr"`
	TestCases  []TestCase  `xml:"testcase"`
	TestSuites []TestSuite `xml:"testsuite"`
}

type TestCase struct {
	XMLName xml.Name `xml:"testcase"`
	// ConfigurationHash is used to distinguish the same test case runs,
	// performed with different build configurations (e.g., Debug vs. Release) or different devices/simulators
	ConfigurationHash string      `xml:"configuration-hash,attr"`
	Name              string      `xml:"name,attr"`
	ClassName         string      `xml:"classname,attr"`
	Time              float64     `xml:"time,attr"`
	Error             *Error      `xml:"error,omitempty"`
	Failure           *Failure    `xml:"failure,omitempty"`
	Skipped           *Skipped    `xml:"skipped,omitempty"`
	Properties        *Properties `xml:"properties,omitempty"`
	SystemOut         *SystemOut  `xml:"system-out,omitempty"`
	SystemErr         *SystemErr  `xml:"system-err,omitempty"`
}

type Error struct {
	XMLName xml.Name `xml:"error,omitempty"`
	Value   string   `xml:",chardata"`
}

type Failure struct {
	XMLName xml.Name `xml:"failure,omitempty"`
	Value   string   `xml:",chardata"`
}

type Skipped struct {
	XMLName xml.Name `xml:"skipped,omitempty"`
	Value   string   `xml:",chardata"`
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

type SystemOut struct {
	XMLName xml.Name `xml:"system-out,omitempty"`
	Value   string   `xml:",chardata"`
}

type SystemErr struct {
	XMLName xml.Name `xml:"system-err,omitempty"`
	Value   string   `xml:",chardata"`
}
