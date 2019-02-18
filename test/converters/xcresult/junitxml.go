package xcresult

import "encoding/xml"

// Junit4XML ...
type Junit4XML struct {
	XMLName    xml.Name `xml:"testsuites"`
	TestSuites []TestSuite
}

// TestSuite ...
type TestSuite struct {
	XMLName   xml.Name `xml:"testsuite"`
	Name      string   `xml:"name,attr"`
	Tests     int      `xml:"tests,attr"`
	Failures  int      `xml:"failures,attr"`
	Errors    int      `xml:"errors,attr"`
	Time      float64  `xml:"time,attr"`
	TestCases []TestCase
}

// TestCase ...
type TestCase struct {
	XMLName   xml.Name `xml:"testcase"`
	Name      string   `xml:"name,attr"`
	ClassName string   `xml:"classname,attr"`
	Time      float64  `xml:"time,attr"`
	Failure   string   `xml:"failure,omitempty"`
}
