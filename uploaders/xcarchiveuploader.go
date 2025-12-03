package uploaders

import (
	"errors"
	"fmt"

	"github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployXcarchive ...
func (u *Uploader) DeployXcarchive(item deployment.DeployableItem, buildURL, token string) ([]ArtifactURLs, error) {
	pth := item.Path

	xcarchiveInfo, err := u.iosParser.ParseXCArchiveData(pth)
	if err != nil {
		if errors.Is(err, metaparser.MacOSProjectIsNotSupported) {
			return nil, fmt.Errorf("macOS archive deployment is not supported: %w", err)
		} else {
			return nil, fmt.Errorf("failed to parse deployment info for %s: %w", pth, err)
		}
	}

	u.logger.Printf("xcarchive infos: %+v", printableAppInfo(xcarchiveInfo.AppInfo))

	artifact := ArtifactArgs{
		Path:     pth,
		FileSize: xcarchiveInfo.FileSizeBytes,
	}
	buildArtifactMeta := AppDeploymentMetaData{
		IOSArtifactInfo:        xcarchiveInfo,
		NotifyUserGroups:       "",
		AlwaysNotifyUserGroups: "",
		NotifyEmails:           "",
		IsEnablePublicPage:     false,
	}

	urLs, err := u.upload(buildURL, token, artifact, "ios-xcarchive", "", &item, &buildArtifactMeta)
	if err != nil {
		return nil, fmt.Errorf("failed xcarchive deploy: %w", err)
	}

	return urLs, nil
}
