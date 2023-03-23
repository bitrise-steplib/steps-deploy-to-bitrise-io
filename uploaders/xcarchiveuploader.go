package uploaders

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/xcarchive"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

func DeployXcarchive(item deployment.DeployableItem, buildURL, token string) (ArtifactURLs, error) {
	uploadURL, artifactID, err := createArtifact(buildURL, token, item.Path, "ios-xcarchive", "")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create xcarchive artifact: %s %w", item.Path, err)
	}

	if err := uploadArtifact(uploadURL, item.Path, ""); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload xcarchive artifact, error: %s", err)
	}

	archiveMetadata, err := parseMetadata(item)
	if err != nil {
		log.Warnf("Metadata parsing failed, we are going to deploy the file as a plain artifact without metadata")
		log.Printf("Parsing error: %s", err)
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, archiveMetadata, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish xcarchive artifact, error: %s", err)
	}
	return artifactURLs, nil
}

func parseMetadata(item deployment.DeployableItem) (*AppDeploymentMetaData, error) {
	unzippedPth, err := xcarchive.UnzipXcarchive(item.Path)
	if err != nil {
		return nil, err
	}

	archivePth := filepath.Join(unzippedPth, pathutil.GetFileName(item.Path))
	isMacos, err := xcarchive.IsMacOS(archivePth)
	if err != nil {
		return nil, fmt.Errorf("could not check if given archive is macOS or not, error: %s", err)
	} else if isMacos {
		return nil, fmt.Errorf("macOS archive parsing is not supported")
	}

	iosArchive, err := xcarchive.NewIosArchive(archivePth)
	if err != nil {
		return nil, fmt.Errorf("failed to parse iOS xcarchive from %s. Error: %s", archivePth, err)
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

	fileSize, err := fileSizeInBytes(item.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get xcarchive size: %s", err)
	}

	xcarchiveInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%f", fileSize),
		"app_info":        appInfo,
		"scheme":          scheme,
	}

	log.Printf("xcarchive metadata: %+v", appInfo)

	return &AppDeploymentMetaData{
		ArtifactInfo:       xcarchiveInfoMap,
		NotifyUserGroups:   "",
		NotifyEmails:       "",
		IsEnablePublicPage: false,
	}, nil
}
