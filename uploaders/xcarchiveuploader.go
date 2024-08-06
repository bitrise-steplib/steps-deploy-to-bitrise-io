package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployXcarchive ...
func DeployXcarchive(item deployment.DeployableItem, buildURL, token string) (ArtifactURLs, error) {
	logger := log.NewLogger()
	pth := item.Path
	fileManager := fileutil.NewFileManager()
	parser := metaparser.New(logger, fileManager)

	xcarchiveInfo, err := parser.ParseXCArchiveData(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to parse deployment info for %s: %w", pth, err)
	}

	logger.Printf("xcarchive infos: %v", xcarchiveInfo)

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
		IOSArtifactInfo:    xcarchiveInfo,
		NotifyUserGroups:   "",
		NotifyEmails:       "",
		IsEnablePublicPage: false,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish xcarchive (%s) upload: %w", pth, err)
	}

	return artifactURLs, nil
}
