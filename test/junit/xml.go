package junit

import (
	"encoding/xml"
)

// XML ...
type XML struct {
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
