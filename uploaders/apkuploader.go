package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/androidartifact"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployAPK ...
func DeployAPK(item deployment.DeployableItem, artifacts []string, buildURL, token, notifyUserGroups, notifyEmails string, isEnablePublicPage bool) (ArtifactURLs, error) {
	pth := item.Path
	apkInfo, err := androidartifact.GetAPKInfo(pth)
	if err != nil {
		return ArtifactURLs{}, err
	}

	appInfo := map[string]interface{}{
		"app_name":        apkInfo.AppName,
		"package_name":    apkInfo.PackageName,
		"version_code":    apkInfo.VersionCode,
		"version_name":    apkInfo.VersionName,
		"min_sdk_version": apkInfo.MinSDKVersion,
	}

	log.Printf("apk infos: %v", appInfo)

	if apkInfo.PackageName == "" {
		log.Warnf("Package name is undefined, AndroidManifest.xml package content:\n%s", apkInfo.RawPackageContent)
	}

	if apkInfo.VersionCode == "" {
		log.Warnf("Version code is undefined, AndroidManifest.xml package content:\n%s", apkInfo.RawPackageContent)
	}

	if apkInfo.VersionName == "" {
		log.Warnf("Version name is undefined, AndroidManifest.xml package content:\n%s", apkInfo.RawPackageContent)
	}

	// ---

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to get apk size, error: %s", err)
	}

	info := androidartifact.ParseArtifactPath(pth)

	apkInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%f", fileSize),
		"app_info":        appInfo,
		"module":          info.Module,
		"product_flavour": info.ProductFlavour,
		"build_type":      info.BuildType,
	}
	splitMeta, err := androidartifact.CreateSplitArtifactMeta(pth, artifacts)
	if err != nil {
		log.Errorf("Failed to create split meta, error: %s", err)
	} else {
		apkInfoMap["aab"] = splitMeta.AAB
		apkInfoMap["apk"] = splitMeta.APK
		apkInfoMap["split"] = splitMeta.Split
		apkInfoMap["universal"] = splitMeta.UniversalApk
	}

	// ---

	const APKContentType = "application/vnd.android.package-archive"
	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "android-apk", APKContentType)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create apk artifact: %s %w", pth, err)
	}

	if err := uploadArtifact(uploadURL, pth, APKContentType); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload apk artifact, error: %s", err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		ArtifactInfo:       apkInfoMap,
		NotifyUserGroups:   notifyUserGroups,
		NotifyEmails:       notifyEmails,
		IsEnablePublicPage: isEnablePublicPage,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish apk artifact, error: %s", err)
	}

	return artifactURLs, nil
}
