package uploaders

import (
	"encoding/json"
	"fmt"
	"github.com/bitrise-io/go-utils/pathutil"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/go-xcode/xcarchive"
)

// DeployXcarchive ...
func DeployXcarchive(pth, buildURL, token string) error {
	log.Printf("analyzing xcarchive")
	unzippedPth, err := xcarchive.UnzipXcarchive(pth)
	if err != nil {
		return err
	}

	ismacos, err := xcarchive.IsMacOS(filepath.Join(unzippedPth, pathutil.GetFileName(pth)))
	if err != nil {
		return fmt.Errorf("could not check if given project is macOS or not, error: %s", err)
	}
	if ismacos {
		log.Warnf("MacOS project found at path %s. Currently it's not supported, so won't be deployed", unzippedPth)
		return nil
	}

	infoPlistPth, err := xcarchive.GetEmbeddedInfoPlistPath(filepath.Join(unzippedPth, pathutil.GetFileName(pth)))
	if err != nil {
		return fmt.Errorf("failed to unwrap Info.plist from xcarchive, error: %s", err)
	}

	infoPlistData, err := plistutil.NewPlistDataFromFile(infoPlistPth)
	if err != nil {
		return fmt.Errorf("failed to parse Info.plist, error: %s", err)
	}

	appTitle, _ := infoPlistData.GetString("CFBundleName")
	bundleID, _ := infoPlistData.GetString("CFBundleIdentifier")
	version, _ := infoPlistData.GetString("CFBundleShortVersionString")
	buildNumber, _ := infoPlistData.GetString("CFBundleVersion")
	minOSVersion, _ := infoPlistData.GetString("MinimumOSVersion")
	deviceFamilyList, _ := infoPlistData.GetUInt64Array("UIDeviceFamily")

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
		return fmt.Errorf("failed to get xcarchive size, error: %s", err)
	}

	xcarchiveInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%f", fileSize),
		"app_info":        appInfo,
	}

	artifactInfoBytes, err := json.Marshal(xcarchiveInfoMap)
	if err != nil {
		return fmt.Errorf("failed to marshal xcarchive infos, error: %s", err)
	}

	log.Printf("  xcarchive infos: %v", appInfo)

	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "file")
	if err != nil {
		return fmt.Errorf("failed to create xcarchive artifact, error: %s", err)
	}

	if err := uploadArtifact(uploadURL, pth, ""); err != nil {
		return fmt.Errorf("failed to upload xcarchive artifact, error: %s", err)
	}

	_, err = finishArtifact(buildURL, token, artifactID, string(artifactInfoBytes), "", "", "false")
	if err != nil {
		return fmt.Errorf("failed to finish xcarchive artifact, error: %s", err)
	}
	return nil
}
