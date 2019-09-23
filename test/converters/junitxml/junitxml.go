package junitxml

import (
	"encoding/xml"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
	"github.com/pkg/errors"
)

// Converter holds data of the converter
type Converter struct {
	files []string
}

// Detect return true if the test results a Juni4 XML file
func (h *Converter) Detect(files []string) bool {
	h.files = nil
	for _, file := range files {
		if strings.HasSuffix(file, ".xml") {
			h.files = append(h.files, file)
		}
	}
	return len(h.files) > 0
}

// merges Suites->Cases->Error and Suites->Cases->SystemErr field values into Suites->Cases->Failure field
// with 2 newlines and error category prefix
// the two newlines applied only if there is a failur emessage already
func regroupErrors(suites []junit.TestSuite) {
	for testSuiteIndex := range suites {
		ts := suites[testSuiteIndex]
		for testCaseIndex := range ts.TestCases {
			tc := suites[testSuiteIndex].TestCases[testCaseIndex]

			var messages []string

			if len(tc.Failure) > 0 {
				messages = append(messages, tc.Failure)
			}

			if len(tc.Error.Message) > 0 {
				messages = append(messages, "Error message:\n"+tc.Error.Message)
			}

			if len(tc.Error.Value) > 0 {
				messages = append(messages, "Error value:\n"+tc.Error.Value)
			}

			if len(tc.SystemErr) > 0 {
				messages = append(messages, "System Error:\n"+tc.SystemErr)
			}

			ts.Failures += ts.Errors

			suites[testSuiteIndex].TestCases[testCaseIndex] = tc

			tc.Failure, tc.Error.Message, tc.Error.Value, tc.SystemErr, ts.Errors = strings.Join(messages, "\n\n"), "", "", "", 0
		}
	}
}

func parseTestSuites(filePath string) ([]junit.TestSuite, error) {
	data, err := fileutil.ReadBytesFromFile(filePath)
	if err != nil {
		return nil, err
	}

	var testSuites junit.XML
	testSuitesError := xml.Unmarshal(data, &testSuites)
	if testSuitesError == nil {
		regroupErrors(testSuites.TestSuites)
		return testSuites.TestSuites, nil
	}

	var testSuite junit.TestSuite
	if err := xml.Unmarshal(data, &testSuite); err != nil {
		return nil, errors.Wrap(errors.Wrap(err, string(data)), testSuitesError.Error())
	}

	ts := []junit.TestSuite{testSuite}
	regroupErrors(ts)
	return ts, nil
}

// XML returns the xml content bytes
func (h *Converter) XML() (junit.XML, error) {
	var xmlContent junit.XML

	for _, file := range h.files {
		testSuites, err := parseTestSuites(file)
		if err != nil {
			return junit.XML{}, err
		}

		xmlContent.TestSuites = append(xmlContent.TestSuites, testSuites...)
	}

	return xmlContent, nil
}
