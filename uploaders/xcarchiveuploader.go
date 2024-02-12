package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	xcarchiveV2 "github.com/bitrise-io/go-xcode/xcarchive/v2"
	"github.com/bitrise-io/go-xcode/zipreader"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployXcarchive ...
func DeployXcarchive(item deployment.DeployableItem, buildURL, token string) (ArtifactURLs, error) {
	pth := item.Path

	reader, err := zipreader.OpenZip(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to open ipa file %s, error: %s", pth, err)
	}
	xcarchiveReader := xcarchiveV2.NewXcarchiveZipReader(*reader)
	isMacos := xcarchiveReader.IsMacOS()
	if isMacos {
		log.Warnf("macOS archive deployment is not supported, skipping file: %s", pth)
		return ArtifactURLs{}, nil // MacOS project is not supported, so won't be deployed.
	}
	archiveInfoPlist, err := xcarchiveReader.InfoPlist()
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to parse archive Info.plist from %s: %s", pth, err)
	}

	iosXCArchiveReader := xcarchiveV2.NewIOSXcarchiveZipReader(*reader)
	appInfoPlist, err := iosXCArchiveReader.AppInfoPlist()
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to parse application Info.plist from %s: %s", pth, err)
	}

	appTitle, _ := appInfoPlist.GetString("CFBundleName")
	bundleID, _ := appInfoPlist.GetString("CFBundleIdentifier")
	version, _ := appInfoPlist.GetString("CFBundleShortVersionString")
	buildNumber, _ := appInfoPlist.GetString("CFBundleVersion")
	minOSVersion, _ := appInfoPlist.GetString("MinimumOSVersion")
	deviceFamilyList, _ := appInfoPlist.GetUInt64Array("UIDeviceFamily")
	scheme, _ := archiveInfoPlist.GetString("SchemeName")

	appInfo := map[string]interface{}{
		"app_title":          appTitle,
		"bundle_id":          bundleID,
		"version":            version,
		"build_number":       buildNumber,
		"min_OS_version":     minOSVersion,
		"device_family_list": deviceFamilyList,
	}

	// ---

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to get xcarchive size, error: %s", err)
	}

	xcarchiveInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%f", fileSize),
		"app_info":        appInfo,
		"scheme":          scheme,
	}

	log.Printf("xcarchive infos: %v", appInfo)

	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "ios-xcarchive", "")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create xcarchive artifact: %s %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, pth, ""); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload xcarchive artifact, error: %s", err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		ArtifactInfo:       xcarchiveInfoMap,
		NotifyUserGroups:   "",
		NotifyEmails:       "",
		IsEnablePublicPage: false,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish xcarchive artifact, error: %s", err)
	}
	return artifactURLs, nil
}
