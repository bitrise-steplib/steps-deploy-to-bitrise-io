package uploaders

import (
	"fmt"
	"time"

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
	tasks, err := createArtifact(buildURL, token, artifact, artifactType, contentType, item.ArchiveAsArtifact, item.IntermediateFileMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to create artifact (%s): %w", artifact.Path, err)
	}

	useIntermediateFileURLs := true
	if item.ArchiveAsArtifact && item.IntermediateFileMeta != nil && len(tasks) > 1 {
		// When the item is both a Build Artifact and an Intermediate File,
		// only surface the artifact URLs from the Build Artifact task.
		useIntermediateFileURLs = false
	}

	var artifactURLs []ArtifactURLs
	for _, task := range tasks {
		start := time.Now()
		parts, uploadErr := uploadAllParts(artifact.Path, artifact.FileSize, task.PartSize, task.PartURLs)

		transferType := Artifact
		if task.IsIntermediate {
			transferType = Intermediate
		}
		details := TransferDetails{
			Size:     artifact.FileSize,
			Duration: time.Since(start),
			Hostname: extractHost(task.PartURLs[0]),
		}
		u.tracker.logFileTransfer(transferType, details, uploadErr, item.ArchiveAsArtifact, item.IsIntermediateFile())

		if uploadErr != nil {
			if _, abortErr := finishArtifact(buildURL, token, task.Identifier(), false, nil, nil); abortErr != nil {
				u.logger.Warnf("Failed to abort multipart upload for artifact %d: %s", task.ID, abortErr)
			}
			return nil, fmt.Errorf("failed to upload artifact parts (%s): %w", artifact.Path, uploadErr)
		}

		urls, err := finishArtifact(buildURL, token, task.Identifier(), true, parts, buildArtifactMeta)
		if err != nil {
			return nil, fmt.Errorf("failed to finish artifact upload (%s): %w", artifact.Path, err)
		}

		if !task.IsIntermediate || useIntermediateFileURLs {
			artifactURLs = append(artifactURLs, urls)
		}
	}

	return artifactURLs, nil
}
