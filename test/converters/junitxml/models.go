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
	XMLName           xml.Name    `xml:"testcase"`
	ConfigurationHash string      `xml:"configuration-hash,attr"`
	Name              string      `xml:"name,attr"`
	ClassName         string      `xml:"classname,attr"`
	Time              float64     `xml:"time,attr"`
	Failure           *Failure    `xml:"failure,omitempty"`
	Properties        *Properties `xml:"properties,omitempty"`
	Skipped           *Skipped    `xml:"skipped,omitempty"`
	Error             *Error      `xml:"error,omitempty"`
	SystemErr         string      `xml:"system-err,omitempty"`
	SystemOut         string      `xml:"system-out,omitempty"`

	FlakyFailures []FlakyFailure `xml:"flakyFailure,omitempty"`
	FlakyErrors   []FlakyError   `xml:"flakyError,omitempty"`
	RerunFailures []RerunFailure `xml:"rerunFailure,omitempty"`
	RerunErrors   []RerunError   `xml:"rerunError,omitempty"`
}

type FlakyFailure struct {
	XMLName   xml.Name `xml:"flakyFailure"`
	Message   string   `xml:"message,attr"`
	Type      string   `xml:"type,attr"`
	SystemErr string   `xml:"system-err,omitempty"`
	SystemOut string   `xml:"system-out,omitempty"`
}

type FlakyError struct {
	XMLName   xml.Name `xml:"flakyError"`
	Message   string   `xml:"message,attr"`
	Type      string   `xml:"type,attr"`
	SystemErr string   `xml:"system-err,omitempty"`
	SystemOut string   `xml:"system-out,omitempty"`
}

type RerunFailure struct {
	XMLName   xml.Name `xml:"rerunFailure"`
	Message   string   `xml:"message,attr"`
	Type      string   `xml:"type,attr"`
	SystemErr string   `xml:"system-err,omitempty"`
	SystemOut string   `xml:"system-out,omitempty"`
}

type RerunError struct {
	XMLName   xml.Name `xml:"rerunError"`
	Message   string   `xml:"message,attr"`
	Type      string   `xml:"type,attr"`
	SystemErr string   `xml:"system-err,omitempty"`
	SystemOut string   `xml:"system-out,omitempty"`
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

type Property struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

type Properties struct {
	XMLName  xml.Name   `xml:"properties"`
	Property []Property `xml:"property"`
}
