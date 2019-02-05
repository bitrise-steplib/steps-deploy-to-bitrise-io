package xcresult

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"

	"howett.net/plist"

	"github.com/bitrise-io/go-utils/fileutil"
)

//
// TestSummaries.plist

type TestSummaryPlist struct {
	FormatVersion     string
	TestableSummaries []TestableSummary
}

func collapseSubtestTree(data Subtests) (tests Subtests) {
	for _, test := range data {
		if len(test.Subtests) > 0 {
			tests = append(tests, collapseSubtestTree(test.Subtests)...)
		}
		if test.TestStatus != "" {
			tests = append(tests, test)
		}
	}
	return
}

func (summaryPlist TestSummaryPlist) Tests() map[string]Subtests {
	tests := map[string]Subtests{}
	var subTests Subtests
	for _, testableSummary := range summaryPlist.TestableSummaries {
		for _, test := range testableSummary.Tests {
			subTests = append(subTests, collapseSubtestTree(test.Subtests)...)
		}
	}
	for _, test := range subTests {
		testID := strings.Split(test.TestIdentifier, "/")[0]
		tests[testID] = append(tests[testID], test)
	}
	return tests
}

type TestableSummary struct {
	TargetName      string
	TestKind        string
	TestName        string
	TestObjectClass string
	Tests           []Test
}

type FailureSummary struct {
	FileName           string
	LineNumber         int
	Message            string
	PerformanceFailure bool
}

type Subtest struct {
	Duration         float64
	TestStatus       string
	TestIdentifier   string
	TestName         string
	TestObjectClass  string
	Subtests         Subtests
	FailureSummaries []FailureSummary
}

func (st Subtest) Failure() (message string) {
	prefix := ""
	for _, failure := range st.FailureSummaries {
		message += fmt.Sprintf("%s%s:%d - %s", prefix, failure.FileName, failure.LineNumber, failure.Message)
		prefix = "\n"
	}
	return
}

type Subtests []Subtest

func (sts Subtests) FailuresCount() (count int) {
	for _, test := range sts {
		if len(test.FailureSummaries) > 0 {
			count++
		}
	}
	return count
}

func (sts Subtests) TotalTime() (time float64) {
	for _, test := range sts {
		time += test.Duration
	}
	return time
}

type Test struct {
	Subtests Subtests
}

//
////

// Junit4 XML

type Junit4XML struct {
	XMLName    xml.Name `xml:"testsuites"`
	TestSuites []TestSuite
}

type TestSuite struct {
	XMLName   xml.Name `xml:"testsuite"`
	Name      string   `xml:"name,attr"`
	Tests     int      `xml:"tests,attr"`
	Failures  int      `xml:"failures,attr"`
	Errors    int      `xml:"errors,attr"`
	Time      float64  `xml:"time,attr"`
	TestCases []TestCase
}

type TestCase struct {
	XMLName   xml.Name `xml:"testcase"`
	Name      string   `xml:"name,attr"`
	ClassName string   `xml:"classname,attr"`
	Time      float64  `xml:"time,attr"`
	Failure   string   `xml:"failure,omitempty"`
}

//
////

type Handler struct {
	files                  []string
	testSummariesPlistPath string
}

func (h *Handler) SetFiles(files []string) {
	h.files = files
}
func (h *Handler) Detect() bool {
	for _, file := range h.files {
		if strings.HasSuffix(file, ".xcresult") {
			h.testSummariesPlistPath = filepath.Join(file, "TestSummaries.plist")
			return true
		}
	}
	return false
}

func (h *Handler) XML() ([]byte, error) {
	data, err := fileutil.ReadBytesFromFile(h.testSummariesPlistPath)
	if err != nil {
		return nil, err
	}

	var plistData TestSummaryPlist
	if _, err := plist.Unmarshal(data, &plistData); err != nil {
		return nil, err
	}

	var xmlData Junit4XML
	for testID, tests := range plistData.Tests() {
		testSuite := TestSuite{
			Name:     testID,
			Tests:    len(tests),
			Failures: tests.FailuresCount(),
			Time:     tests.TotalTime(),
		}

		for _, test := range tests {
			testSuite.TestCases = append(testSuite.TestCases, TestCase{
				Name:      test.TestName,
				ClassName: testID,
				Failure:   test.Failure(),
				Time:      test.Duration,
			})
		}

		xmlData.TestSuites = append(xmlData.TestSuites, testSuite)
	}

	xmlOutputData, err := xml.MarshalIndent(xmlData, "", " ")
	return append([]byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"), xmlOutputData...), err
}
