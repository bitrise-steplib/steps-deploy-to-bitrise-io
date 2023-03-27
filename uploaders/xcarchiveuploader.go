package uploaders

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/xcarchive"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployXcarchive ...
func DeployXcarchive(item deployment.DeployableItem, buildURL, token string) (ArtifactURLs, error) {
	pth := item.Path
	unzippedPth, err := xcarchive.UnzipXcarchive(pth)
	if err != nil {
		return ArtifactURLs{}, err
	}

	archivePth := filepath.Join(unzippedPth, pathutil.GetFileName(pth))
	isMacos, err := xcarchive.IsMacOS(archivePth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("could not check if given project is macOS or not, error: %s", err)
	} else if isMacos {
		log.Warnf("macOS archive deployment is not supported, skipping file: %s", archivePth)
		return ArtifactURLs{}, nil // MacOS project is not supported, so won't be deployed.
	}

	iosArchive, err := xcarchive.NewIosArchive(archivePth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to parse iOS XcArchive from %s. Error: %s", archivePth, err)
	}

	appTitle, _ := iosArchive.Application.InfoPlist.GetString("CFBundleName")
	bundleID := iosArchive.Application.BundleIdentifier()
	version, _ := iosArchive.Application.InfoPlist.GetString("CFBundleShortVersionString")
	buildNumber, _ := iosArchive.Application.InfoPlist.GetString("CFBundleVersion")
	minOSVersion, _ := iosArchive.Application.InfoPlist.GetString("MinimumOSVersion")
	deviceFamilyList, _ := iosArchive.Application.InfoPlist.GetUInt64Array("UIDeviceFamily")
	scheme, _ := iosArchive.InfoPlist.GetString("SchemeName")

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

	if err := uploadArtifact(uploadURL, pth, ""); err != nil {
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
