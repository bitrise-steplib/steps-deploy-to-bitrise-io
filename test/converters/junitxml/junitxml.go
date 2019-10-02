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
// the two newlines applied only if there is a failure message already
// this is required because our testing addon currently handles failure field properly
func regroupErrors(suites []junit.TestSuite) []junit.TestSuite {
	for testSuiteIndex := range suites {
		for testCaseIndex := range suites[testSuiteIndex].TestCases {
			tc := suites[testSuiteIndex].TestCases[testCaseIndex]

			var messages []string

			if len(tc.Failure) > 0 {
				messages = append(messages, tc.Failure)
			}

			if tc.Error != nil {
				if len(tc.Error.Message) > 0 {
					messages = append(messages, "Error message:\n"+tc.Error.Message)
				}

				if len(tc.Error.Value) > 0 {
					messages = append(messages, "Error value:\n"+tc.Error.Value)
				}
			}

			if len(tc.SystemErr) > 0 {
				messages = append(messages, "System error:\n"+tc.SystemErr)
			}

			tc.Failure, tc.Error, tc.SystemErr = strings.Join(messages, "\n\n"), nil, ""

			suites[testSuiteIndex].Failures += suites[testSuiteIndex].Errors
			suites[testSuiteIndex].Errors = 0
			suites[testSuiteIndex].TestCases[testCaseIndex] = tc
		}
	}
	return suites
}

func parseTestSuites(filePath string) ([]junit.TestSuite, error) {
	data, err := fileutil.ReadBytesFromFile(filePath)
	if err != nil {
		return nil, err
	}

	var testSuites junit.XML
	testSuitesError := xml.Unmarshal(data, &testSuites)
	if testSuitesError == nil {
		return regroupErrors(testSuites.TestSuites), nil
	}

	var testSuite junit.TestSuite
	if err := xml.Unmarshal(data, &testSuite); err != nil {
		return nil, errors.Wrap(errors.Wrap(err, string(data)), testSuitesError.Error())
	}

	return regroupErrors([]junit.TestSuite{testSuite}), nil
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
