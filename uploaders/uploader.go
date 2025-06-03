package uploaders

import (
	"fmt"

	androidparser "github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	iosparser "github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

type Uploader struct {
	logger        log.Logger
	fileManager   fileutil.FileManager
	androidParser *androidparser.Parser
	iosParser     *iosparser.Parser
	tracker       tracker
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
		tracker:       newTracker(env.NewRepository(), logger),
	}
}

func (u *Uploader) Wait() {
	u.tracker.wait()
}

func (u *Uploader) upload(buildURL, token string, artifact ArtifactArgs, artifactType, contentType string, item *deployment.DeployableItem, buildArtifactMeta *AppDeploymentMetaData) ([]ArtifactURLs, error) {
	uploadTasks, err := createArtifact(buildURL, token, artifact, artifactType, contentType, item.ArchiveAsArtifact, item.IntermediateFileMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to create artifact (%s): %w", artifact.Path, err)
	}

	var artifactURLs []ArtifactURLs
	useIntermediateFileURLs := true
	if item.ArchiveAsArtifact && item.IntermediateFileMeta != nil {
		// If an item is both a Build Artifact and an Intermediate File,
		// only use the artifact URLs of the Build Artifact's upload task.
		useIntermediateFileURLs = false
	}
	for _, task := range uploadTasks {
		details, err := UploadArtifact(task.URL, artifact, contentType)

		var transferType = Artifact
		if task.IsIntermediate {
			transferType = Intermediate
		}

		u.tracker.logFileTransfer(transferType, details, err, item.ArchiveAsArtifact, item.IsIntermediateFile())

		if err != nil {
			return nil, fmt.Errorf("failed to upload artifact (%s): %w", artifact.Path, err)
		}

		urls, err := finishArtifact(buildURL, token, task.Identifier(), buildArtifactMeta)
		if err != nil {
			return nil, fmt.Errorf("failed to finish artifact upload (%s): %w", artifact.Path, err)
		}

		if !task.IsIntermediate || useIntermediateFileURLs {
			artifactURLs = append(artifactURLs, urls)
		}
	}

	return artifactURLs, nil
}
