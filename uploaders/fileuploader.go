package uploaders

import (
	"fmt"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployFile ...
func DeployFile(item deployment.DeployableItem, buildURL, token string) (ArtifactURLs, error) {
	fileSize, err := fileSizeInBytes(item.Path)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("get file size: %s", err)
	}
	artifact := ArtifactArgs{
		Path:     item.Path,
		FileSize: fileSize,
	}
	uploadURL, artifactID, err := createArtifact(buildURL, token, artifact, "file", "")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("create file artifact: %s %w", artifact.Path, err)
	}

	if err := UploadArtifact(uploadURL, artifact, ""); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload file artifact, error: %s", err)
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, nil, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish file artifact, error: %s", err)
	}
	return artifactURLs, nil
}
