package xcresult

import (
	"encoding/xml"
	"path/filepath"
	"strings"

	"howett.net/plist"

	"github.com/bitrise-io/go-utils/fileutil"
)

// Converter ...
type Converter struct {
	files                  []string
	testSummariesPlistPath string
}

// Detect ...
func (h *Converter) Detect(files []string) bool {
	h.files = files
	for _, file := range h.files {
		if strings.HasSuffix(file, ".xcresult") {
			h.testSummariesPlistPath = filepath.Join(file, "TestSummaries.plist")
			return true
		}
	}
	return false
}

// XML ...
func (h *Converter) XML() ([]byte, error) {
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
