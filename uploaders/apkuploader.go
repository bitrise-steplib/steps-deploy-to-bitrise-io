package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployAPK ...
func (u *Uploader) DeployAPK(item deployment.DeployableItem, artifacts []string, buildURL, token, notifyUserGroups, alwaysNotifyUserGroups, notifyEmails string, isEnablePublicPage bool) ([]ArtifactURLs, error) {
	pth := item.Path

	apkInfo, err := u.androidParser.ParseAPKData(pth)
	if err != nil {
		return nil, err
	}

	u.logger.Printf("APK metadata: %+v", printableAppInfo(apkInfo.AppInfo))

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
	buildArtifactMeta := AppDeploymentMetaData{
		AndroidArtifactInfo:    apkInfo,
		NotifyUserGroups:       notifyUserGroups,
		AlwaysNotifyUserGroups: alwaysNotifyUserGroups,
		NotifyEmails:           notifyEmails,
		IsEnablePublicPage:     isEnablePublicPage,
	}

	urLs, err := u.upload(buildURL, token, artifact, "android-apk", APKContentType, &item, &buildArtifactMeta)
	if err != nil {
		return nil, fmt.Errorf("failed apk deploy: %w", err)
	}

	return urLs, nil
}
