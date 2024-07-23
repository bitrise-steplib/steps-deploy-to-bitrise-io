package uploaders

import (
	"fmt"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/androidartifact"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/androidsignature"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/bundletool"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"

	"github.com/bitrise-io/go-utils/log"
)

// DeployAAB ...
func DeployAAB(item deployment.DeployableItem, artifacts []string, buildURL, token string, bt bundletool.Path) (ArtifactURLs, error) {
	pth := item.Path
	aabInfo, err := androidartifact.GetAABInfo(bt, pth)
	if err != nil {
		return ArtifactURLs{}, err
	}

	appInfo := map[string]interface{}{
		"package_name":    aabInfo.PackageName,
		"version_code":    aabInfo.VersionCode,
		"version_name":    aabInfo.VersionName,
		"app_name":        aabInfo.AppName,
		"min_sdk_version": aabInfo.MinSDKVersion,
	}

	log.Printf("aab infos: %v", appInfo)

	if aabInfo.PackageName == "" {
		log.Warnf("Package name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent)
	}

	if aabInfo.VersionCode == "" {
		log.Warnf("Version code is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent)
	}

	if aabInfo.VersionName == "" {
		log.Warnf("Version name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent)
	}

	if aabInfo.MinSDKVersion == "" {
		log.Warnf("Min SDK version is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent)
	}

	if aabInfo.AppName == "" {
		log.Warnf("App name is undefined, AndroidManifest.xml package content:\n%s", aabInfo.RawPackageContent)
	}

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to get apk size, error: %s", err)
	}

	info := androidartifact.ParseArtifactPath(pth)

	aabInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%d", fileSize),
		"app_info":        appInfo,
		"module":          info.Module,
		"product_flavour": info.ProductFlavour,
		"build_type":      info.BuildType,
	}

	signature, err := androidsignature.Read(pth)
	if err != nil {
		log.Warnf("Failed to read signature: %s", err)
	}
	aabInfoMap["signed_by"] = signature

	splitMeta, err := androidartifact.CreateSplitArtifactMeta(pth, artifacts)
	if err != nil {
		log.Warnf("Failed to create split meta, error: %s", err)
	} else {
		aabInfoMap["apk"] = splitMeta.APK
		aabInfoMap["aab"] = splitMeta.AAB
		aabInfoMap["split"] = splitMeta.Split
		aabInfoMap["universal"] = splitMeta.UniversalApk
	}

	// ---

	const AABContentType = "application/octet-stream aab"
	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "android-apk", AABContentType)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create apk artifact: %s %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, pth, AABContentType); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload apk artifact, error: %s", err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		ArtifactInfo:       aabInfoMap,
		NotifyUserGroups:   "",
		NotifyEmails:       "",
		IsEnablePublicPage: false,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish apk artifact, error: %s", err)
	}

	return artifactURLs, nil
}
