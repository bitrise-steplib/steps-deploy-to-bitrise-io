package uploaders

import (
	"fmt"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployFile ...
func DeployFile(item deployment.DeployableItem, buildURL, token string) (ArtifactURLs, error) {
	pth := item.Path
	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "file", "")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create file artifact: %s %w", pth, err)
	}

	if err := uploadArtifact(uploadURL, pth, ""); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload file artifact, error: %s", err)
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, nil, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish file artifact, error: %s", err)
	}
	return artifactURLs, nil
}
