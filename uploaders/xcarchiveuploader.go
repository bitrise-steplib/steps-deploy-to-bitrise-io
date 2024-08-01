package uploaders

import (
	"fmt"

	logV2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/artifacts"
	"github.com/bitrise-io/go-xcode/v2/zip"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployXcarchive ...
func DeployXcarchive(item deployment.DeployableItem, buildURL, token string) (ArtifactURLs, error) {
	logger := logV2.NewLogger()
	pth := item.Path

	appInfo, scheme, err := readXCArchiveDeploymentMeta(pth, logger)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to parse deployment info for %s: %w", pth, err)
	}

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to get size of %s: %w", pth, err)
	}

	xcarchiveInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%d", fileSize),
		"app_info":        appInfo,
		"scheme":          scheme,
	}

	logger.Printf("xcarchive infos: %v", appInfo)

	artifact := ArtifactArgs{
		Path:     pth,
		FileSize: fileSize,
	}
	uploadURL, artifactID, err := createArtifact(buildURL, token, artifact, "ios-xcarchive", "")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create xcarchive artifact from %s %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, artifact, ""); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload xcarchive (%s): %w", pth, err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		ArtifactInfo:       xcarchiveInfoMap,
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

func readXCArchiveDeploymentMeta(xcarchivePth string, logger logV2.Logger) (map[string]interface{}, string, error) {
	reader, err := zip.NewDefaultReader(xcarchivePth, logger)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			logger.Warnf("%s", err)
		}
	}()

	xcarchiveReader := artifacts.NewXCArchiveReader(reader)
	isMacos := xcarchiveReader.IsMacOS()
	if isMacos {
		logger.Warnf("macOS archive deployment is not supported, skipping xcarchive")
		return nil, "", nil // MacOS project is not supported, so won't be deployed.
	}
	archiveInfoPlist, err := xcarchiveReader.InfoPlist()
	if err != nil {
		return nil, "", fmt.Errorf("failed to unwrap Info.plist from xcarchive: %w", err)
	}

	iosXCArchiveReader := artifacts.NewIOSXCArchiveReader(reader)
	appInfoPlist, err := iosXCArchiveReader.AppInfoPlist()
	if err != nil {
		return nil, "", fmt.Errorf("failed to unwrap application Info.plist from xcarchive: %w", err)
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

	return appInfo, scheme, nil
}
