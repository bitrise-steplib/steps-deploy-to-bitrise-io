package junitxml

import (
	"encoding/xml"
)

// TestReport ...
type TestReport struct {
	XMLName    xml.Name    `xml:"testsuites"`
	TestSuites []TestSuite `xml:"testsuite"`
}

// TestSuite ...
type TestSuite struct {
	XMLName    xml.Name    `xml:"testsuite"`
	Name       string      `xml:"name,attr"`
	Tests      int         `xml:"tests,attr"`
	Failures   int         `xml:"failures,attr"`
	Errors     int         `xml:"errors,attr"`
	Skipped    int         `xml:"skipped,attr"`
	Assertions int         `xml:"assertions,attr,omitempty"`
	Time       float64     `xml:"time,attr"`
	Timestamp  string      `xml:"timestamp,attr,omitempty"`
	File       string      `xml:"file,attr,omitempty"`
	TestCases  []TestCase  `xml:"testcase,omitempty"`
	TestSuites []TestSuite `xml:"testsuite,omitempty"`
}

// TestCase ...
type TestCase struct {
	XMLName           xml.Name    `xml:"testcase"`
	ConfigurationHash string      `xml:"configuration-hash,attr,omitempty"`
	Name              string      `xml:"name,attr"`
	ClassName         string      `xml:"classname,attr"`
	Assertions        int         `xml:"assertions,attr,omitempty"`
	Time              float64     `xml:"time,attr"`
	File              string      `xml:"file,attr,omitempty"`
	Line              int         `xml:"line,attr,omitempty"`
	Failure           *Failure    `xml:"failure,omitempty"`
	Error             *Error      `xml:"error,omitempty"`
	Skipped           *Skipped    `xml:"skipped,omitempty"`
	Properties        *Properties `xml:"properties,omitempty"`
	SystemErr         string      `xml:"system-err,omitempty"`
	SystemOut         string      `xml:"system-out,omitempty"`

	FlakyFailures []FlakyFailure `xml:"flakyFailure,omitempty"`
	FlakyErrors   []FlakyError   `xml:"flakyError,omitempty"`
	RerunFailures []RerunFailure `xml:"rerunFailure,omitempty"`
	RerunErrors   []RerunError   `xml:"rerunError,omitempty"`
}

type FlakyFailure struct {
	XMLName   xml.Name `xml:"flakyFailure"`
	Type      string   `xml:"type,attr,omitempty"`
	Message   string   `xml:"message,attr,omitempty"`
	Value     string   `xml:",chardata"`
	SystemErr string   `xml:"system-err,omitempty"`
	SystemOut string   `xml:"system-out,omitempty"`
}

type FlakyError struct {
	XMLName   xml.Name `xml:"flakyError"`
	Type      string   `xml:"type,attr,omitempty"`
	Message   string   `xml:"message,attr,omitempty"`
	Value     string   `xml:",chardata"`
	SystemErr string   `xml:"system-err,omitempty"`
	SystemOut string   `xml:"system-out,omitempty"`
}

type RerunFailure struct {
	XMLName   xml.Name `xml:"rerunFailure"`
	Type      string   `xml:"type,attr,omitempty"`
	Message   string   `xml:"message,attr,omitempty"`
	Value     string   `xml:",chardata"`
	SystemErr string   `xml:"system-err,omitempty"`
	SystemOut string   `xml:"system-out,omitempty"`
}

type RerunError struct {
	XMLName   xml.Name `xml:"rerunError"`
	Type      string   `xml:"type,attr,omitempty"`
	Message   string   `xml:"message,attr,omitempty"`
	Value     string   `xml:",chardata"`
	SystemErr string   `xml:"system-err,omitempty"`
	SystemOut string   `xml:"system-out,omitempty"`
}

// Failure ...
type Failure struct {
	XMLName xml.Name `xml:"failure,omitempty"`
	Type    string   `xml:"type,attr,omitempty"`
	Message string   `xml:"message,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

// Error ...
type Error struct {
	XMLName xml.Name `xml:"error,omitempty"`
	Type    string   `xml:"type,attr,omitempty"`
	Message string   `xml:"message,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

// Skipped ...
type Skipped struct {
	XMLName xml.Name `xml:"skipped,omitempty"`
	Message string   `xml:"message,attr,omitempty"`
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
