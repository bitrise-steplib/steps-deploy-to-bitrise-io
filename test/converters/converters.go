// Package converters contains the interface that is required to be a package a test result converter.
// It must be possible to set files from outside(for example if someone wants to use
// a pre-filtered files list), need to return Junit4 xml test result, and needs to have a
// Detect method to see if the converter can run with the files included in the test result dictionary.
// (So a converter can run only if the dir has a TestSummaries.plist file for example)
package converters

import (
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters/junitxml"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters/xcresult"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters/xcresult3"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
)

// Intf is the required interface a converter needs to match
type Intf interface {
	XML() (testreport.TestReport, error)
	Detect([]string) bool
	Setup(useOldXCResultExtractionMethod bool)
}

var converters = []Intf{
	&junitxml.Converter{},
	&xcresult.Converter{},
	&xcresult3.Converter{},
}

// List lists all supported converters
func List() []Intf {
	return converters
}
