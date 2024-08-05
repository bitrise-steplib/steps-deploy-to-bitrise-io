package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"
	"github.com/bitrise-io/go-android/v2/metaparser/bundletool"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployAAB ...
func DeployAAB(item deployment.DeployableItem, artifacts []string, buildURL, token string, bt bundletool.Path) (ArtifactURLs, error) {
	pth := item.Path

	logger := NewLogger()
	parser := metaparser.New(logger, bt)
	aabInfo, err := parser.ParseAABData(pth)
	if err != nil {
		return ArtifactURLs{}, err
	}

	logger.Printf("aab infos: %v", aabInfo.AppInfo)

	if aabInfo.AppInfo.PackageName == "" {
		logger.Warnf("Package name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.AppInfo.RawPackageContent)
	}

	if aabInfo.AppInfo.VersionCode == "" {
		logger.Warnf("Version code is undefined, AndroidManifest.xml package content:\n%s", aabInfo.AppInfo.RawPackageContent)
	}

	if aabInfo.AppInfo.VersionName == "" {
		logger.Warnf("Version name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.AppInfo.RawPackageContent)
	}

	if aabInfo.AppInfo.MinSDKVersion == "" {
		logger.Warnf("Min SDK version is undefined, AndroidManifest.xml package content:\n%s", aabInfo.AppInfo.RawPackageContent)
	}

	if aabInfo.AppInfo.AppName == "" {
		logger.Warnf("App name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.AppInfo.RawPackageContent)
	}

	splitMeta, err := androidartifact.CreateSplitArtifactMeta(nil, pth, artifacts)
	if err != nil {
		logger.Warnf("Failed to create split meta, error: %s", err)
	} else {
		aabInfo.Artifact = androidartifact.Artifact(splitMeta)
	}

	// ---

	const AABContentType = "application/octet-stream aab"
	artifact := ArtifactArgs{
		Path:     pth,
		FileSize: aabInfo.FileSizeBytes,
	}
	uploadURL, artifactID, err := createArtifact(buildURL, token, artifact, "android-apk", AABContentType)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create apk artifact: %s %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, artifact, AABContentType); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload apk artifact, error: %s", err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		AndroidArtifactInfo: aabInfo,
		NotifyUserGroups:    "",
		NotifyEmails:        "",
		IsEnablePublicPage:  false,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish apk artifact, error: %s", err)
	}

	return artifactURLs, nil
}
