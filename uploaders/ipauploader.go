package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployIPA ...
func DeployIPA(item deployment.DeployableItem, buildURL, token, notifyUserGroups, notifyEmails string, isEnablePublicPage bool) (ArtifactURLs, error) {
	logger := log.NewLogger()
	pth := item.Path
	fileManager := fileutil.NewFileManager()
	parser := metaparser.New(logger, fileManager)

	ipaInfo, err := parser.ParseIPAData(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to parse deployment info for %s: %w", pth, err)
	}

	if ipaInfo.ProvisioningInfo.IPAExportMethod == exportoptions.MethodAppStore {
		logger.Warnf("is_enable_public_page is set, but public download isn't allowed for app-store distributions")
		logger.Warnf("setting is_enable_public_page to false ...")
		isEnablePublicPage = false
	}

	logger.Printf("ipa infos: %v", ipaInfo)

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
		IOSArtifactInfo:    ipaInfo,
		NotifyUserGroups:   notifyUserGroups,
		NotifyEmails:       notifyEmails,
		IsEnablePublicPage: isEnablePublicPage,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish ipa (%s) upload: %w", pth, err)
	}

	return artifactURLs, nil
}
