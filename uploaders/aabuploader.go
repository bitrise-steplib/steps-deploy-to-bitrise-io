package uploaders

import (
	"fmt"

	metaparser "github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"
	"github.com/bitrise-io/go-android/v2/metaparser/bundletool"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployAAB ...
func DeployAAB(item deployment.DeployableItem, artifacts []string, buildURL, token string, bt bundletool.Path) (ArtifactURLs, error) {
	pth := item.Path
	aabInfo, err := metaparser.ParseAABData(pth, bt)
	if err != nil {
		return ArtifactURLs{}, err
	}

	for _, warning := range aabInfo.Warnings {
		log.Warnf(warning)
	}

	splitMeta, err := androidartifact.CreateSplitArtifactMeta(pth, artifacts)
	if err != nil {
		log.Warnf("Failed to create split meta, error: %s", err)
	} else {
		aabInfo.Artifact = androidartifact.Artifact(splitMeta)
	}

	// ---

	const AABContentType = "application/octet-stream aab"
	artifact := ArtifactArgs{
		Path:     pth,
		FileSize: aabInfo.FileSizeBytes,
	}
	uploadURL, artifactID, err := createArtifact(buildURL, token, artifact, "android-apk", AABContentType)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create apk artifact: %s %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, artifact, AABContentType); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload apk artifact, error: %s", err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		ArtifactInfo:       aabInfo,
		NotifyUserGroups:   "",
		NotifyEmails:       "",
		IsEnablePublicPage: false,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish apk artifact, error: %s", err)
	}

	return artifactURLs, nil
}
