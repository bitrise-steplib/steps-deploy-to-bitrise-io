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
	files       []string
	xmlFilePath string
}

// Detect return true if the test results a Juni4 XML file
func (h *Converter) Detect(files []string) bool {
	h.files = files
	for _, file := range h.files {
		if strings.HasSuffix(file, ".xml") {
			h.xmlFilePath = file
			return true
		}
	}
	return false
}

// XML returns the xml content bytes
func (h *Converter) XML() (junit.XML, error) {
	data, err := fileutil.ReadBytesFromFile(h.xmlFilePath)
	if err != nil {
		return junit.XML{}, err
	}

	var xmlContent junit.XML
	if err := xml.Unmarshal(data, &xmlContent); err != nil {
		return junit.XML{}, errors.Wrap(err, string(data))
	}
	return xmlContent, nil
}
