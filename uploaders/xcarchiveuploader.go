package uploaders

import (
	"encoding/json"
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/ipa"
	"github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/go-xcode/profileutil"
	xcarchive "github.com/bitrise-io/go-xcode/xcarchive"
)

// DeployXcarchive ...
func DeployXcarchive(pth, buildURL, token string) error {
	log.Printf("analyzing xcarchive")

	infoPlistPth, err := xcarchive.UnwrapEmbeddedInfoPlist(pth)
	if err != nil {
		return fmt.Errorf("failed to unwrap Info.plist from xcarchive, error: %s", err)
	}

	infoPlistData, err := plistutil.NewPlistDataFromFile(infoPlistPth)
	if err != nil {
		return fmt.Errorf("failed to parse Info.plist, error: %s", err)
	}

	appTitle, _ := infoPlistData.GetString("CFBundleName")                // i
	bundleID, _ := infoPlistData.GetString("CFBundleIdentifier")          // o i
	version, _ := infoPlistData.GetString("CFBundleShortVersionString")   // o i
	buildNumber, _ := infoPlistData.GetString("CFBundleVersion")          // o i
	minOSVersion, _ := infoPlistData.GetString("MinimumOSVersion")        // i
	deviceFamilyList, _ := infoPlistData.GetUInt64Array("UIDeviceFamily") // i

	appInfo := map[string]interface{}{
		"app_title":          appTitle,
		"bundle_id":          bundleID,
		"version":            version,
		"build_number":       buildNumber,
		"min_OS_version":     minOSVersion,
		"device_family_list": deviceFamilyList,
	}

	log.Printf("  xcarchive infos: %v", appInfo)

	// ---

	provisioningProfilePth, err := ipa.UnwrapEmbeddedMobileProvision(pth)
	if err != nil {
		return fmt.Errorf("failed to unwrap embedded.mobilprovision from xcarchive, error: %s", err)
	}

	provisioningProfileInfo, err := profileutil.NewProvisioningProfileInfoFromFile(provisioningProfilePth)
	if err != nil {
		return fmt.Errorf("failed to parse embedded.mobilprovision, error: %s", err)
	}

	teamName := provisioningProfileInfo.TeamName
	creationDate := provisioningProfileInfo.CreationDate
	provisionsAlldevices := provisioningProfileInfo.ProvisionsAllDevices
	expirationDate := provisioningProfileInfo.ExpirationDate
	deviceUDIDList := provisioningProfileInfo.ProvisionedDevices
	profileName := provisioningProfileInfo.Name

	provisioningInfo := map[string]interface{}{
		"creation_date":          creationDate,
		"expire_date":            expirationDate,
		"device_UDID_list":       deviceUDIDList,
		"team_name":              teamName,
		"profile_name":           profileName,
		"provisions_all_devices": provisionsAlldevices,
	}

	// ---

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return fmt.Errorf("failed to get xcarchive size, error: %s", err)
	}

	xcarchiveInfoMap := map[string]interface{}{
		"file_size_bytes":   fmt.Sprintf("%f", fileSize),
		"app_info":          appInfo,
		"provisioning_info": provisioningInfo,
	}

	artifactInfoBytes, err := json.Marshal(xcarchiveInfoMap)
	if err != nil {
		return fmt.Errorf("failed to marshal xcarchive infos, error: %s", err)
	}

	// ---

	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "ios-xcarchive")
	if err != nil {
		return fmt.Errorf("failed to create xcarchive artifact, error: %s", err)
	}

	if err := uploadArtifact(uploadURL, pth, ""); err != nil {
		return fmt.Errorf("failed to upload xcarchive artifact, error: %s", err)
	}
	return nil
}
