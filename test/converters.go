// Package test wires together the available test result converters.
package test

import (
	"github.com/bitrise-io/go-android/v2/testresult/junitxml"
	"github.com/bitrise-io/go-steputils/v2/testreport"
	"github.com/bitrise-io/go-xcode/v2/testresult/xcresult"
	"github.com/bitrise-io/go-xcode/v2/testresult/xcresult3"
)

var converterList = []testreport.Converter{
	&junitxml.Converter{},
	&xcresult.Converter{},
	&xcresult3.Converter{},
}

func AvailableConverters() []testreport.Converter {
	return converterList
}
