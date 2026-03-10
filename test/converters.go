// Package test contains the interface that is required to be a package a test result converter.
// It must be possible to set files from outside(for example if someone wants to use
// a pre-filtered files list), need to return Junit4 xml test result, and needs to have a
// Detect method to see if the converter can run with the files included in the test result dictionary.
// (So a converter can run only if the dir has a TestSummaries.plist file for example)
package test

import (
	"github.com/bitrise-io/go-android/v2/testresult/junitxml"
	"github.com/bitrise-io/go-steputils/v2/testreport"
	"github.com/bitrise-io/go-xcode/v2/testresult/xcresult"
	"github.com/bitrise-io/go-xcode/v2/testresult/xcresult3"
)

// Converter is the required interface a converter needs to match
type Converter interface {
	Setup(useOldXCResultExtractionMethod bool)
	Detect([]string) bool
	Convert() (testreport.TestReport, error)
}

var converterList = []Converter{
	&junitxml.Converter{},
	&xcresult.Converter{},
	&xcresult3.Converter{},
}

// AvailableConverters lists all supported converters
func AvailableConverters() []Converter {
	return converterList
}
