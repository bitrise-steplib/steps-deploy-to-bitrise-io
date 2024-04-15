package uploaders

import (
	"fmt"

	logV2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/v2/zip"
	"github.com/bitrise-io/go-xcode/v2/ziputil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployIPA ...
func DeployIPA(item deployment.DeployableItem, buildURL, token, notifyUserGroups, notifyEmails string, isEnablePublicPage bool) (ArtifactURLs, error) {
	logger := logV2.NewLogger()
	pth := item.Path
	defaultZipReaderFactory := func() (ziputil.ReadCloser, error) {
		return ziputil.NewDefaultRead(pth, logger)
	}

	appInfo, provisioningInfo, err := readIPADeploymentMeta(defaultZipReaderFactory, logger)
	if err != nil {
		if !ziputil.IsErrFormat(err) {
			return ArtifactURLs{}, fmt.Errorf("failed to parse deployment info for %s: %w", pth, err)
		}

		logger.Warnf("Default zip reader failed to extract ipa file (%s): %s", pth, err)
		logger.Warnf("Continue with fallback zip reader...")

		dittoZipReaderFactory := func() (ziputil.ReadCloser, error) {
			return ziputil.NewDittoReader(pth, logger)
		}

		appInfo, provisioningInfo, err = readIPADeploymentMeta(dittoZipReaderFactory, logger)
		if err != nil {
			return ArtifactURLs{}, fmt.Errorf("failed to parse deployment info for %s: %w", pth, err)
		}
	}

	if provisioningInfo["ipa_export_method"] == exportoptions.MethodAppStore {
		logger.Warnf("is_enable_public_page is set, but public download isn't allowed for app-store distributions")
		logger.Warnf("setting is_enable_public_page to false ...")
		isEnablePublicPage = false
	}

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to size of %s: %w", pth, err)
	}

	ipaInfoMap := map[string]interface{}{
		"file_size_bytes":   fmt.Sprintf("%f", fileSize),
		"app_info":          appInfo,
		"provisioning_info": provisioningInfo,
	}

	const IPAContentType = "application/octet-stream ipa"
	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "ios-ipa", IPAContentType)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create ipa artifact from %s: %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, pth, IPAContentType); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload ipa (%s): %w", pth, err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		ArtifactInfo:       ipaInfoMap,
		NotifyUserGroups:   notifyUserGroups,
		NotifyEmails:       notifyEmails,
		IsEnablePublicPage: isEnablePublicPage,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish ipa (%s) upload: %w", pth, err)
	}

	return artifactURLs, nil
}

type zipReaderFactory func() (ziputil.ReadCloser, error)

func readIPADeploymentMeta(zipFactory zipReaderFactory, logger logV2.Logger) (map[string]interface{}, map[string]interface{}, error) {
	zipReader, err := zipFactory()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open ipa: %w", err)
	}
	defer func() {
		if err := zipReader.Close(); err != nil {
			logger.Warnf("Failed to close ipa: %s", err)
		}
	}()

	ipaReader := zip.NewIPAReader(zipReader)
	infoPlist, err := ipaReader.AppInfoPlist()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unwrap Info.plist from ipa: %w", err)
	}

	appTitle, _ := infoPlist.GetString("CFBundleName")
	bundleID, _ := infoPlist.GetString("CFBundleIdentifier")
	version, _ := infoPlist.GetString("CFBundleShortVersionString")
	buildNumber, _ := infoPlist.GetString("CFBundleVersion")
	minOSVersion, _ := infoPlist.GetString("MinimumOSVersion")
	deviceFamilyList, _ := infoPlist.GetUInt64Array("UIDeviceFamily")

	appInfo := map[string]interface{}{
		"app_title":          appTitle,
		"bundle_id":          bundleID,
		"version":            version,
		"build_number":       buildNumber,
		"min_OS_version":     minOSVersion,
		"device_family_list": deviceFamilyList,
	}

	logger.Printf("ipa infos: %v", appInfo)

	provisioningProfileInfo, err := ipaReader.ProvisioningProfileInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read profile info from ipa: %w", err)
	}

	teamName := provisioningProfileInfo.TeamName
	creationDate := provisioningProfileInfo.CreationDate
	provisionsAllDevices := provisioningProfileInfo.ProvisionsAllDevices
	expirationDate := provisioningProfileInfo.ExpirationDate
	deviceUDIDList := provisioningProfileInfo.ProvisionedDevices
	profileName := provisioningProfileInfo.Name
	exportMethod := provisioningProfileInfo.ExportType

	provisioningInfo := map[string]interface{}{
		"creation_date":          creationDate,
		"expire_date":            expirationDate,
		"device_UDID_list":       deviceUDIDList,
		"team_name":              teamName,
		"profile_name":           profileName,
		"provisions_all_devices": provisionsAllDevices,
		"ipa_export_method":      exportMethod,
	}

	return appInfo, provisioningInfo, nil
}
