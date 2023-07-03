package uploaders

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/androidartifact"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/androidsignature"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/bundletool"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployAAB ...
func DeployAAB(item deployment.DeployableItem, artifacts []string, buildURL, token, bundletoolVersion string) (ArtifactURLs, error) {
	pth := item.Path
	r, err := bundletool.New(bundletoolVersion)
	if err != nil {
		return ArtifactURLs{}, err
	}
	cmd := r.Command("dump", "manifest", "--bundle", pth)

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return ArtifactURLs{}, fmt.Errorf("command failed with exit status %d (%s): %s", exitErr.ExitCode(), cmd.PrintableCommandArgs(), out)
		}
		return ArtifactURLs{}, fmt.Errorf("executing command failed (%s): %w", cmd.PrintableCommandArgs(), err)
	}

	packageName, versionCode, versionName := androidartifact.ParsePackageInfos(out, true)

	appInfo := map[string]interface{}{
		"package_name": packageName,
		"version_code": versionCode,
		"version_name": versionName,
	}

	log.Printf("aab infos: %v", appInfo)

	if packageName == "" {
		log.Warnf("Package name is undefined, AndroidManifest.xml package content:\n%s", out)
	}

	if versionCode == "" {
		log.Warnf("Version code is undefined, AndroidManifest.xml package content:\n%s", out)
	}

	if versionName == "" {
		log.Warnf("Version name is undefined, AndroidManifest.xml package content:\n%s", out)
	}

	// ---

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to get apk size, error: %s", err)
	}

	info := androidartifact.ParseArtifactPath(pth)

	aabInfoMap := map[string]interface{}{
		"file_size_bytes": fmt.Sprintf("%f", fileSize),
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

	if err := uploadArtifact(uploadURL, pth, AABContentType); err != nil {
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
