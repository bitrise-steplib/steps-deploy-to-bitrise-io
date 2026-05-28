package uploaders

import (
	androidparser "github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	iosparser "github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

type Uploader struct {
	logger               log.Logger
	fileManager          fileutil.FileManager
	androidParser        *androidparser.Parser
	iosParser            *iosparser.Parser
	tracker              tracker
	useMultipartUpload   bool
	multipartConcurrency int
}

func New(
	logger log.Logger,
	fileManager fileutil.FileManager,
	androidParser *androidparser.Parser,
	iosParser *iosparser.Parser,
	useMultipartUpload bool,
	multipartConcurrency int,
) *Uploader {
	return &Uploader{
		logger:               logger,
		fileManager:          fileManager,
		androidParser:        androidParser,
		iosParser:            iosParser,
		tracker:              newTracker(env.NewRepository(), logger),
		useMultipartUpload:   useMultipartUpload,
		multipartConcurrency: multipartConcurrency,
	}
}

func (u *Uploader) Wait() {
	u.tracker.wait()
}

func (u *Uploader) upload(buildURL, token string, artifact ArtifactArgs, artifactType, contentType string, item *deployment.DeployableItem, buildArtifactMeta *AppDeploymentMetaData) ([]ArtifactURLs, error) {
	if u.useMultipartUpload {
		return u.uploadMultipart(buildURL, token, artifact, artifactType, contentType, item, buildArtifactMeta)
	}
	return u.uploadSingle(buildURL, token, artifact, artifactType, contentType, item, buildArtifactMeta)
}
