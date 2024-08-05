package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployAPK ...
func DeployAPK(item deployment.DeployableItem, artifacts []string, buildURL, token, notifyUserGroups, notifyEmails string, isEnablePublicPage bool) (ArtifactURLs, error) {
	pth := item.Path

	logger := NewLogger()
	parser := metaparser.New(logger, "")
	apkInfo, err := parser.ParseAPKData(pth)
	if err != nil {
		return ArtifactURLs{}, err
	}

	splitMeta, err := androidartifact.CreateSplitArtifactMeta(logger, pth, artifacts)
	if err != nil {
		logger.Errorf("Failed to create split meta, error: %s", err)
	} else {
		apkInfo.Artifact = androidartifact.Artifact(splitMeta)
	}

	// ---

	const APKContentType = "application/vnd.android.package-archive"
	artifact := ArtifactArgs{
		Path:     pth,
		FileSize: apkInfo.FileSizeBytes,
	}
	uploadURL, artifactID, err := createArtifact(buildURL, token, artifact, "android-apk", APKContentType)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create apk artifact: %s %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, artifact, APKContentType); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload apk artifact, error: %s", err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		AndroidArtifactInfo: apkInfo,
		NotifyUserGroups:    notifyUserGroups,
		NotifyEmails:        notifyEmails,
		IsEnablePublicPage:  isEnablePublicPage,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish apk artifact, error: %s", err)
	}

	return artifactURLs, nil
}
