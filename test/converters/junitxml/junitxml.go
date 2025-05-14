package junitxml

import (
	"encoding/xml"
	"os"
	"strings"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
	"github.com/pkg/errors"
)

type resultReader interface {
	ReadAll() ([]byte, error)
}

type fileReader struct {
	Filename string
}

func (r *fileReader) ReadAll() ([]byte, error) {
	return os.ReadFile(r.Filename)
}

type stringReader struct {
	Contents string
}

func (r *stringReader) ReadAll() ([]byte, error) {
	return []byte(r.Contents), nil
}

// Converter holds data of the converter
type Converter struct {
	results []resultReader
}

func (c *Converter) Setup(_ bool) {}

// Detect return true if the test results a Juni4 XML file
func (c *Converter) Detect(files []string) bool {
	c.results = nil
	for _, file := range files {
		if strings.HasSuffix(file, ".xml") || strings.HasSuffix(file, ".junit") {
			c.results = append(c.results, &fileReader{Filename: file})
		}
	}

	return len(c.results) > 0
}

// merges Suites->Cases->Error and Suites->Cases->SystemErr field values into Suites->Cases->Failure field
// with 2 newlines and error category prefix
// the two newlines applied only if there is a failure message already
// this is required because our testing addon currently handles failure field properly
func regroupErrors(suites []junit.TestSuite) []junit.TestSuite {
	for testSuiteIndex, suite := range suites {
		for testCaseIndex, tc := range suite.TestCases {
			var messages []string

			if tc.Failure != nil {
				if len(strings.TrimSpace(tc.Failure.Message)) > 0 {
					messages = append(messages, tc.Failure.Message)
				}

				if len(strings.TrimSpace(tc.Failure.Value)) > 0 {
					messages = append(messages, tc.Failure.Value)
				}
			}

			if tc.Error != nil {
				if len(strings.TrimSpace(tc.Error.Message)) > 0 {
					messages = append(messages, "Error message:\n"+tc.Error.Message)
				}

				if len(strings.TrimSpace(tc.Error.Value)) > 0 {
					messages = append(messages, "Error value:\n"+tc.Error.Value)
				}
			}

			if len(tc.SystemErr) > 0 {
				messages = append(messages, "System error:\n"+tc.SystemErr)
			}

			tc.Error, tc.SystemErr = nil, ""
			if messages != nil {
				tc.Failure = &junit.Failure{
					Value: strings.Join(messages, "\n\n"),
				}
			}

			suites[testSuiteIndex].Failures += suites[testSuiteIndex].Errors
			suites[testSuiteIndex].Errors = 0
			suites[testSuiteIndex].TestCases[testCaseIndex] = tc
		}
	}

	return suites
}

func parseTestSuites(result resultReader) ([]junit.TestSuite, error) {
	data, err := result.ReadAll()
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
func (c *Converter) XML() (junit.XML, error) {
	var xmlContent junit.XML

	for _, result := range c.results {
		testSuites, err := parseTestSuites(result)
		if err != nil {
			return junit.XML{}, err
		}

		xmlContent.TestSuites = append(xmlContent.TestSuites, testSuites...)
	}

	return xmlContent, nil
}
