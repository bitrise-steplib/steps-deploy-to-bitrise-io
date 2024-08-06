package uploaders

import (
	androidparser "github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	iosparser "github.com/bitrise-io/go-xcode/v2/metaparser"
)

type Uploader struct {
	logger        log.Logger
	fileManager   fileutil.FileManager
	androidParser *androidparser.Parser
	iosParser     *iosparser.Parser
}

func New(
	logger log.Logger,
	fileManager fileutil.FileManager,
	androidParser *androidparser.Parser,
	iosParser *iosparser.Parser,
) *Uploader {
	return &Uploader{
		logger:        logger,
		fileManager:   fileManager,
		androidParser: androidParser,
		iosParser:     iosParser,
	}
}
