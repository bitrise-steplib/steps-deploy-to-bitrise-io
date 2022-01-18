package xcresult

import (
	"path/filepath"
	"strings"
	"unicode"

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

// by one of our issue reports, need to replace backspace char (U+0008) as it is an invalid character for xml unmarshaller
// the legal character ranges are here: https://www.w3.org/TR/REC-xml/#charsets
// so the exclusion will be:
/*
	\u0000 - \u0008
	\u000B
	\u000C
	\u000E - \u001F
	\u007F - \u0084
	\u0086 - \u009F
	\uD800 - \uDFFF

	Unicode range D800–DFFF is used as surrogate pair. Unicode and ISO/IEC 10646 do not assign characters to any of the code points in the D800–DFFF range, so an individual code value from a surrogate pair does not represent a character. (A couple of code points — the first from the high surrogate area (D800–DBFF), and the second from the low surrogate area (DC00–DFFF) — are used in UTF-16 to represent a character in supplementary planes)
	\uFDD0 - \uFDEF; \uFFFE; \uFFFF
*/
// These are non-characters in the standard, not assigned to anything; and have no meaning.
func filterIllegalChars(data []byte) (filtered []byte) {
	illegalCharFilter := func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}
	filtered = []byte(strings.Map(illegalCharFilter, string(data)))
	return
}

// XML ...
func (h *Converter) XML() (junit.XML, error) {
	data, err := fileutil.ReadBytesFromFile(h.testSummariesPlistPath)
	if err != nil {
		return junit.XML{}, err
	}

	data = filterIllegalChars(data)

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
			Skipped:  tests.SkippedCount(),
			Time:     tests.TotalTime(),
		}

		for _, test := range tests {
			failureMessage := test.Failure()

			var failure *junit.Failure
			if len(failureMessage) > 0 {
				failure = &junit.Failure{
					Value: failureMessage,
				}
			}

			var skipped *junit.Skipped
			if test.Skipped() {
				skipped = &junit.Skipped{}
			}

			testSuite.TestCases = append(testSuite.TestCases, junit.TestCase{
				Name:      test.TestName,
				ClassName: testID,
				Failure:   failure,
				Skipped:   skipped,
				Time:      test.Duration,
			})
		}

		xmlData.TestSuites = append(xmlData.TestSuites, testSuite)
	}

	return xmlData, nil
}
