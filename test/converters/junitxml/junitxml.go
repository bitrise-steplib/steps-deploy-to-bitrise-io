package junitxml

import (
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
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
func (h *Converter) XML() ([]byte, error) {
	return fileutil.ReadBytesFromFile(h.xmlFilePath)
}
