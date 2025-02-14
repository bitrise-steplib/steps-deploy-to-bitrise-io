package uploaders

import (
	"errors"
	"fmt"

	"github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployXcarchive ...
func (u *Uploader) DeployXcarchive(item deployment.DeployableItem, buildURL, token string) (ArtifactURLs, error) {
	pth := item.Path

	xcarchiveInfo, err := u.iosParser.ParseXCArchiveData(pth)
	if err != nil {
		if errors.Is(err, metaparser.MacOSProjectIsNotSupported) {
			u.logger.Warnf("macOS archive deployment is not supported, skipping xcarchive")
		} else {
			return ArtifactURLs{}, fmt.Errorf("failed to parse deployment info for %s: %w", pth, err)
		}
	}

	u.logger.Printf("xcarchive infos: %+v", printableAppInfo(xcarchiveInfo.AppInfo))

	artifact := ArtifactArgs{
		Path:     pth,
		FileSize: xcarchiveInfo.FileSizeBytes,
	}
	uploadURL, artifactID, err := createArtifact(buildURL, token, artifact, "ios-xcarchive", "")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create xcarchive artifact from %s %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, artifact, ""); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload xcarchive (%s): %w", pth, err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		IOSArtifactInfo:        xcarchiveInfo,
		NotifyUserGroups:       "",
		AlwaysNotifyUserGroups: "",
		NotifyEmails:           "",
		IsEnablePublicPage:     false,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish xcarchive (%s) upload: %w", pth, err)
	}

	return artifactURLs, nil
}
