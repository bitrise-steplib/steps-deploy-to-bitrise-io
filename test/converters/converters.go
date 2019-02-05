package converters

import (
	"github.com/bitrise-io/steps-deploy-to-bitrise-io/test/converters/junitxml"
	"github.com/bitrise-io/steps-deploy-to-bitrise-io/test/converters/xcresult"
)

type Intf interface {
	SetFiles([]string)
	XML() ([]byte, error)
	Detect() bool
}

var handlers = []Intf{
	&junitxml.Handler{},
	&xcresult.Handler{},
}

func List() []Intf {
	return handlers
}
