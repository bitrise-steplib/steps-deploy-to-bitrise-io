package junit

import (
	"encoding/xml"
	"reflect"
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
	Errors    int        `xml:"errors,attr"`
	Time      float64    `xml:"time,attr"`
	TestCases []TestCase `xml:"testcase"`
}

// TestCase ...
type TestCase struct {
	XMLName   xml.Name `xml:"testcase"`
	Name      string   `xml:"name,attr"`
	ClassName string   `xml:"classname,attr"`
	Time      float64  `xml:"time,attr"`
	Failure   string   `xml:"failure,omitempty"`
	Error     *Error   `xml:"error,omitempty"`
	SystemErr string   `xml:"system-err,omitempty"`
}

// Error ...
type Error struct {
	XMLName xml.Name `xml:"error,omitempty"`
	Message string   `xml:"message,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

// Equal ...
func (x XML) Equal(xml XML) bool {
	// store and clear TestSuite.TestSuites,
	// to make reflect.DeepEqual work as expected,
	// compare TestSuites later
	suitsA, suitsB := x.TestSuites, xml.TestSuites
	x.TestSuites, xml.TestSuites = nil, nil

	if !reflect.DeepEqual(x, xml) {
		return false
	}

	if len(suitsA) != len(suitsB) {
		return false
	}

	for _, suitA := range suitsA {
		found := false
		for j, suitB := range suitsB {
			if suitA.Equal(suitB) {
				found = true
				// remove already found cases to make sure
				// the slices are only different in order of elements
				copy(suitsB[j:], suitsB[j+1:])
				suitsB = suitsB[:len(suitsB)-1]

				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Equal ...
func (s TestSuite) Equal(ts TestSuite) bool {
	// store and clear TestSuite.TestCases,
	// to make reflect.DeepEqual work as expected,
	// compare TestCases later
	casesA, casesB := s.TestCases, ts.TestCases
	s.TestCases, ts.TestCases = nil, nil

	if !reflect.DeepEqual(s, ts) {
		return false
	}

	if len(casesA) != len(casesB) {
		return false
	}

	for _, caseA := range casesA {
		found := false
		for j, caseB := range casesB {
			if reflect.DeepEqual(caseA, caseB) {
				found = true
				// remove already found cases to make sure
				// the slices are only different in order of elements
				copy(casesB[j:], casesB[j+1:])
				casesB = casesB[:len(casesB)-1]

				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
