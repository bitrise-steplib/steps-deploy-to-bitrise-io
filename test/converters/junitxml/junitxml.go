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
	for _, file := range h.files {
		if strings.HasSuffix(file, ".xml") {
			h.files = append(h.files, file)
		}
	}
	return len(h.files) > 0
}

func parseParseTestSuites(filePath string) ([]junit.TestSuite, error) {
	data, err := fileutil.ReadBytesFromFile(filePath)
	if err != nil {
		return nil, err
	}

	var testSuites junit.XML
	testSuitesError := xml.Unmarshal(data, &testSuites)
	if testSuitesError == nil {
		return testSuites.TestSuites, nil
	}

	var testSuite junit.TestSuite
	if err := xml.Unmarshal(data, &testSuite); err != nil {
		return nil, errors.Wrap(errors.Wrap(err, string(data)), testSuitesError.Error())
	}

	return []junit.TestSuite{testSuite}, nil
}

// XML returns the xml content bytes
func (h *Converter) XML() (junit.XML, error) {
	var xmlContent junit.XML

	for _, file := range h.files {
		testSuites, err := parseParseTestSuites(file)
		if err != nil {
			return junit.XML{}, err
		}

		xmlContent.TestSuites = append(xmlContent.TestSuites, testSuites...)
	}

	return xmlContent, nil
}
