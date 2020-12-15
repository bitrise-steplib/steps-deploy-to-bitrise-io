package uploaders

import "fmt"

// DeployFile ...
func DeployFile(pth, buildURL, token string) (ArtifactURLs, error) {
	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "file")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create file artifact, error: %s", err)
	}

	if err := uploadArtifact(uploadURL, pth, ""); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload file artifact, error: %s", err)
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, "", "", "", "no")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish file artifact, error: %s", err)
	}
	return artifactURLs, nil
}
