package xcresult

import (
	"path/filepath"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
	"howett.net/plist"
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
		if filepath.Ext(file) == ".xcresult" {
			testSummariesPlistPath := filepath.Join(file, "TestSummaries.plist")
			if exist, err := pathutil.IsPathExists(testSummariesPlistPath); err != nil || exist == false {
				continue
			}

			h.testSummariesPlistPath = testSummariesPlistPath
			return true
		}
	}
	return false
}

// XML ...
func (h *Converter) XML() (junit.XML, error) {
	data, err := fileutil.ReadBytesFromFile(h.testSummariesPlistPath)
	if err != nil {
		return junit.XML{}, err
	}

	var plistData TestSummaryPlist
	if _, err := plist.Unmarshal(data, &plistData); err != nil {
		return junit.XML{}, err
	}

	var xmlData junit.XML
	for testID, tests := range plistData.Tests() {
		testSuite := junit.TestSuite{
			Name:     testID,
			Tests:    len(tests),
			Failures: tests.FailuresCount(),
			Time:     tests.TotalTime(),
		}

		for _, test := range tests {
			testSuite.TestCases = append(testSuite.TestCases, junit.TestCase{
				Name:      test.TestName,
				ClassName: testID,
				Failure:   test.Failure(),
				Time:      test.Duration,
			})
		}

		xmlData.TestSuites = append(xmlData.TestSuites, testSuite)
	}

	return xmlData, nil
}
