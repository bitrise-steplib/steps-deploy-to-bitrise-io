package uploaders

import (
	"encoding/json"
	"fmt"
)

// DeployFile ...
func DeployFile(pth, buildURL, token string) (ArtifactURLs, error) {
	return DeployFileWithMetaData(pth, buildURL, token, nil)
}

// TODO: handle files that are both Build Artifacts and Intermediate files at the same time
func DeployFileWithMetaData(pth, buildURL, token string, metaData *PipelineIntermediateFileMetaData) (ArtifactURLs, error) {
	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "file", "")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create file artifact, error: %s", err)
	}

	if err := uploadArtifact(uploadURL, pth, ""); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload file artifact, error: %s", err)
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, nil, metaData)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish file artifact, error: %s", err)
	}
	return artifactURLs, nil
}

func convertMetadata(metaData interface{}) (string, error) {
	if metaData == nil {
		return "", nil
	}

	bytes, err := json.Marshal(metaData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal meta: %s", err)
	}

	return string(bytes), nil
}
