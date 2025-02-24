package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployIPA ...
func (u *Uploader) DeployIPA(item deployment.DeployableItem, buildURL, token, notifyUserGroups, alwaysNotifyUserGroups, notifyEmails string, isEnablePublicPage bool) (ArtifactURLs, error) {
	pth := item.Path

	ipaInfo, err := u.iosParser.ParseIPAData(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to parse deployment info for %s: %w", pth, err)
	}

	if ipaInfo.ProvisioningInfo.IPAExportMethod == exportoptions.MethodAppStore {
		u.logger.Warnf("is_enable_public_page is set, but public download isn't allowed for app-store distributions")
		u.logger.Warnf("setting is_enable_public_page to false ...")
		isEnablePublicPage = false
	}

	u.logger.Printf("ipa infos: %+v", printableAppInfo(ipaInfo.AppInfo))

	const IPAContentType = "application/octet-stream ipa"
	artifact := ArtifactArgs{
		Path:     pth,
		FileSize: ipaInfo.FileSizeBytes,
	}
	uploadURL, artifactID, err := createArtifact(buildURL, token, artifact, "ios-ipa", IPAContentType)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create ipa artifact from %s: %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, artifact, IPAContentType); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload ipa (%s): %w", pth, err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		IOSArtifactInfo:        ipaInfo,
		NotifyUserGroups:       notifyUserGroups,
		AlwaysNotifyUserGroups: alwaysNotifyUserGroups,
		NotifyEmails:           notifyEmails,
		IsEnablePublicPage:     isEnablePublicPage,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish ipa (%s) upload: %w", pth, err)
	}

	return artifactURLs, nil
}
