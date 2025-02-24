package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployAPK ...
func (u *Uploader) DeployAPK(item deployment.DeployableItem, artifacts []string, buildURL, token, notifyUserGroups, alwaysNotifyUserGroups, notifyEmails string, isEnablePublicPage bool) (ArtifactURLs, error) {
	pth := item.Path

	apkInfo, err := u.androidParser.ParseAPKData(pth)
	if err != nil {
		return ArtifactURLs{}, err
	}

	u.logger.Printf("apk infos: %+v", printableAppInfo(apkInfo.AppInfo))

	splitMeta, err := androidartifact.CreateSplitArtifactMeta(u.logger, pth, artifacts)
	if err != nil {
		u.logger.Errorf("Failed to create split meta, error: %s", err)
	} else {
		apkInfo.Artifact = androidartifact.Artifact(splitMeta)
	}

	// ---

	const APKContentType = "application/vnd.android.package-archive"
	artifact := ArtifactArgs{
		Path:     pth,
		FileSize: apkInfo.FileSizeBytes,
	}
	uploadURL, artifactID, err := createArtifact(buildURL, token, artifact, "android-apk", APKContentType)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create apk artifact: %s %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, artifact, APKContentType); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload apk artifact, error: %s", err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		AndroidArtifactInfo:    apkInfo,
		NotifyUserGroups:       notifyUserGroups,
		AlwaysNotifyUserGroups: alwaysNotifyUserGroups,
		NotifyEmails:           notifyEmails,
		IsEnablePublicPage:     isEnablePublicPage,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish apk artifact, error: %s", err)
	}

	return artifactURLs, nil
}
