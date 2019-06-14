package uploaders

import "fmt"

// DeployFile ...
func DeployFile(pth, buildURL, token, notifyUserGroups, notifyEmails, isEnablePublicPage string) (string, error) {
	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "file")
	if err != nil {
		return "", fmt.Errorf("failed to create file artifact, error: %s", err)
	}

	if err := uploadArtifact(uploadURL, pth, ""); err != nil {
		return "", fmt.Errorf("failed to upload file artifact, error: %s", err)
	}

	publicInstallPage, err := finishArtifact(buildURL, token, artifactID, "", "", "", "no")
	if err != nil {
		return "", fmt.Errorf("failed to finish file artifact, error: %s", err)
	}
	return publicInstallPage, nil
}
