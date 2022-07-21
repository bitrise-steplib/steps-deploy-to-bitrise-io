package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/envman/envman"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/ziputil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/uploaders"
)

var fileBaseNamesToSkip = []string{".DS_Store"}

// Config ...
type Config struct {
	PipelineIntermediateFiles     string `env:"pipeline_intermediate_files"`
	BuildURL                      string `env:"build_url,required"`
	APIToken                      string `env:"build_api_token,required"`
	IsCompress                    string `env:"is_compress,opt[true,false]"`
	ZipName                       string `env:"zip_name"`
	DeployPath                    string `env:"deploy_path"`
	NotifyUserGroups              string `env:"notify_user_groups"`
	NotifyEmailList               string `env:"notify_email_list"`
	IsPublicPageEnabled           bool   `env:"is_enable_public_page,opt[true,false]"`
	PublicInstallPageMapFormat    string `env:"public_install_page_url_map_format,required"`
	PermanentDownloadURLMapFormat string `env:"permanent_download_url_map_format,required"`
	BuildSlug                     string `env:"BITRISE_BUILD_SLUG,required"`
	TestDeployDir                 string `env:"BITRISE_TEST_DEPLOY_DIR,required"`
	AppSlug                       string `env:"BITRISE_APP_SLUG,required"`
	AddonAPIBaseURL               string `env:"addon_api_base_url,required"`
	AddonAPIToken                 string `env:"addon_api_token"`
	DebugMode                     bool   `env:"debug_mode,opt[true,false]"`
	BundletoolVersion             string `env:"bundletool_version,required"`
}

// PublicInstallPage ...
type PublicInstallPage struct {
	File string
	URL  string
}

// ArtifactURLCollection ...
type ArtifactURLCollection struct {
	PublicInstallPageURLs map[string]string
	PermanentDownloadURLs map[string]string
}

const zippedXcarchiveExt = ".xcarchive.zip"

func fail(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	var config Config
	if err := stepconf.Parse(&config); err != nil {
		fail("Issue with input: %s", err)
	}

	if err := validateGoTemplate(config.PublicInstallPageMapFormat); err != nil {
		fail("PublicInstallPageMapFormat - %s", err)
	}

	stepconf.Print(config)
	fmt.Println()
	log.SetEnableDebugLog(config.DebugMode)

	absDeployPth, err := pathutil.AbsPath(config.DeployPath)
	if err != nil {
		fail("Failed to expand path: %s, error: %s", config.DeployPath, err)
	}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("__deploy-to-bitrise-io__")
	if err != nil {
		fail("Failed to create tmp dir, error: %s", err)
	}

	filesToDeploy, err := collectFilesToDeploy(absDeployPth, config, tmpDir)
	if err != nil {
		fail("%s", err)
	}
	clearedFilesToDeploy := clearDeployFiles(filesToDeploy)

	collector := deployment.NewCollector(deployment.DefaultIsDirFunction, ziputil.ZipDir, tmpDir)
	finalDeployableItems, err := collector.FinalListOfDeployableItems(clearedFilesToDeploy, config.PipelineIntermediateFiles)
	if err != nil {
		fail("%s", err)
	}

	if len(finalDeployableItems) == 0 {
		fmt.Println()
		log.Infof("No deployment files were defined. Please check the deploy_path and pipeline_intermediate_files inputs.")
		log.Donef("Success")

		return
	}

	fmt.Println()
	log.Infof("List of files to deploy")

	logDeployFiles(finalDeployableItems)

	fmt.Println()
	log.Infof("Deploying files")

	artifactURLCollection, err := deploy(finalDeployableItems, config)
	if err != nil {
		fail("%s", err)
	}
	fmt.Println()
	log.Donef("Success")
	log.Printf("You can find the Artifact on Bitrise, on the Build's page: %s", config.BuildURL)

	if err := exportInstallPages(artifactURLCollection, config); err != nil {
		fail("%s", err)
	}
	deployTestResults(config)
}

func exportInstallPages(artifactURLCollection ArtifactURLCollection, config Config) error {
	if len(artifactURLCollection.PublicInstallPageURLs) > 0 {
		pages := mapURLsToInstallPages(artifactURLCollection.PublicInstallPageURLs)

		if err := tools.ExportEnvironmentWithEnvman("BITRISE_PUBLIC_INSTALL_PAGE_URL", pages[0].URL); err != nil {
			return fmt.Errorf("failed to export BITRISE_PUBLIC_INSTALL_PAGE_URL: %s", err)
		}
		log.Printf("The public install page url is now available in the Environment Variable: BITRISE_PUBLIC_INSTALL_PAGE_URL (value: %s)\n", pages[0].URL)

		value, err := exportMapEnvironment("Public Install Page template", config.PublicInstallPageMapFormat, "PublicInstallPageMap", "BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP", pages)
		if err != nil {
			return fmt.Errorf("failed to export BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP, error: %s", err)
		}
		log.Printf("A map of deployed files and their public install page urls is now available in the Environment Variable: BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP (value: %s)", value)
		log.Printf("")
	}
	if len(artifactURLCollection.PermanentDownloadURLs) > 0 {
		pages := mapURLsToInstallPages(artifactURLCollection.PermanentDownloadURLs)
		value, err := exportMapEnvironment("Permanent Download URL template", config.PermanentDownloadURLMapFormat, "PermanentDownloadURLMap", "BITRISE_PERMANENT_DOWNLOAD_URL_MAP", pages)
		if err != nil {
			return fmt.Errorf("failed to export BITRISE_PERMANENT_DOWNLOAD_URL_MAP: %s", err)
		}
		log.Printf("A map of deployed files and their permanent download urls is now available in the Environment Variable: BITRISE_PERMANENT_DOWNLOAD_URL_MAP (value: %s)", value)
		log.Printf("")
	}
	return nil
}

func mapURLsToInstallPages(URLs map[string]string) []PublicInstallPage {
	var pages []PublicInstallPage
	for file, url := range URLs {
		pages = append(pages, PublicInstallPage{
			File: file,
			URL:  url,
		})
	}
	return pages
}

func exportMapEnvironment(templateName string, format string, formatName string, outputKey string, pages []PublicInstallPage) (string, error) {
	var maxEnvLength int

	if configs, err := envman.GetConfigs(); err != nil {
		maxEnvLength = 20 * 1024
	} else {
		maxEnvLength = configs.EnvBytesLimitInKB * 1024
	}

	temp := template.New(templateName)
	temp, err := temp.Parse(format)
	if err != nil {
		return "", fmt.Errorf("error during parsing %s: %s", formatName, err)
	}

	value, logWarning, err := applyTemplateWithMaxSize(temp, pages, maxEnvLength)
	if err != nil {
		return "", err
	}

	if logWarning {
		log.Warnf("too many artifacts, not all urls has been written to output: %s", outputKey)
	}

	return value, tools.ExportEnvironmentWithEnvman(outputKey, value)
}

func applyTemplateWithMaxSize(temp *template.Template, pages []PublicInstallPage, maxEnvLength int) (string, bool, error) {
	var value string
	var logWarning bool
	for {
		buf := new(bytes.Buffer)
		if err := temp.Execute(buf, pages); err != nil {
			return "", false, fmt.Errorf("execute: %s", err)
		}
		value = buf.String()
		if len(value) <= maxEnvLength || len(pages) < 1 {
			break
		}
		logWarning = true
		pages = pages[:len(pages)-1]
	}
	return value, logWarning, nil
}

func logDeployFiles(files []deployment.DeployableItem) {
	for _, file := range files {
		message := fmt.Sprintf("- %s", file.Path)

		if file.PipelineMeta != nil {
			message += " (pipeline intermediate file)"
		}

		log.Printf(message)
	}
}

func clearDeployFiles(filesToDeploy []string) []string {
	var clearedFilesToDeploy []string
	for _, pth := range filesToDeploy {
		for _, fileBaseNameToSkip := range fileBaseNamesToSkip {
			if filepath.Base(pth) == fileBaseNameToSkip {
				log.Warnf("skipping: %s", pth)
			} else {
				clearedFilesToDeploy = append(clearedFilesToDeploy, pth)
			}

		}
	}
	return clearedFilesToDeploy
}

func collectFilesToDeploy(absDeployPth string, config Config, tmpDir string) (filesToDeploy []string, err error) {
	isDeployPathDir, err := pathutil.IsDirExists(absDeployPth)
	if err != nil {
		return nil, fmt.Errorf("failed to check if DeployPath (%s) is a directory or a file, error: %s", absDeployPth, err)
	}

	if !isDeployPathDir {
		fmt.Println()
		log.Infof("Deploying single file")

		filesToDeploy = []string{absDeployPth}
	} else if config.IsCompress == "true" {
		fmt.Println()
		log.Infof("Deploying compressed Deploy directory")

		zipName := filepath.Base(absDeployPth)
		if config.ZipName != "" {
			zipName = config.ZipName
		}
		tmpZipPath := filepath.Join(tmpDir, zipName+".zip")

		if err := ziputil.ZipDir(absDeployPth, tmpZipPath, true); err != nil {
			return nil, fmt.Errorf("failed to zip output dir, error: %s", err)
		}

		filesToDeploy = []string{tmpZipPath}
	} else {
		fmt.Println()
		log.Infof("Deploying the content of the Deploy directory separately")

		pattern := filepath.Join(absDeployPth, "*")
		pths, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to list files in DeployPath, error: %s", err)
		}

		for _, pth := range pths {
			if isDir, err := pathutil.IsDirExists(pth); err != nil {
				return nil, fmt.Errorf("failed to check if path (%s) is a directory or a file, error: %s", pth, err)
			} else if !isDir {
				filesToDeploy = append(filesToDeploy, pth)
			}
		}
	}

	return filesToDeploy, nil
}

func deployTestResults(config Config) {
	if config.AddonAPIToken != "" {
		fmt.Println()
		log.Infof("Upload test results")

		testResults, err := test.ParseTestResults(config.TestDeployDir)
		if err != nil {
			log.Warnf("error during parsing test results: ", err)
		} else {
			log.Printf("- uploading (%d) test results", len(testResults))

			if err := testResults.Upload(config.AddonAPIToken, config.AddonAPIBaseURL, config.AppSlug, config.BuildSlug); err != nil {
				log.Warnf("Failed to upload test results: ", err)
			} else {
				log.Donef("Success")
			}
		}
	}
}

func findAPKsAndAABs(items []deployment.DeployableItem) (apks []deployment.DeployableItem, aabs []deployment.DeployableItem, others []deployment.DeployableItem) {
	for _, item := range items {
		switch getFileType(item.Path) {
		case ".apk":
			apks = append(apks, item)
		case ".aab":
			aabs = append(aabs, item)
		default:
			others = append(others, item)
		}
	}
	return
}

func deploy(deployableItems []deployment.DeployableItem, config Config) (ArtifactURLCollection, error) {
	apks, aabs, others := findAPKsAndAABs(deployableItems)

	var androidArtifacts []string
	for _, artifacts := range append(apks, aabs...) {
		androidArtifacts = append(androidArtifacts, artifacts.Path)
	}

	artifactURLCollection := ArtifactURLCollection{
		PublicInstallPageURLs: map[string]string{},
		PermanentDownloadURLs: map[string]string{},
	}
	for _, apk := range apks {
		log.Donef("Uploading apk file: %s", apk)

		artifactURLs, err := uploaders.DeployAPK(apk, androidArtifacts, config.BuildURL, config.APIToken, config.NotifyUserGroups, config.NotifyEmailList, config.IsPublicPageEnabled)
		if err != nil {
			return ArtifactURLCollection{}, fmt.Errorf("deploy failed, error: %s", err)
		}

		fillURLMaps(artifactURLCollection, artifactURLs, apk.Path, config.IsPublicPageEnabled)
	}

	for _, item := range append(aabs, others...) {
		pth := item.Path
		fileType := getFileType(pth)
		fmt.Println()

		switch fileType {
		case ".ipa":
			log.Donef("Uploading ipa file: %s", pth)

			artifactURLs, err := uploaders.DeployIPA(item, config.BuildURL, config.APIToken, config.NotifyUserGroups, config.NotifyEmailList, config.IsPublicPageEnabled)
			if err != nil {
				return ArtifactURLCollection{}, fmt.Errorf("deploy failed, error: %s", err)
			}

			fillURLMaps(artifactURLCollection, artifactURLs, pth, config.IsPublicPageEnabled)
		case ".aab":
			log.Donef("Uploading aab file: %s", pth)

			artifactURLs, err := uploaders.DeployAAB(item, androidArtifacts, config.BuildURL, config.APIToken, config.BundletoolVersion)
			if err != nil {
				return ArtifactURLCollection{}, fmt.Errorf("deploy failed, error: %s", err)
			}

			fillURLMaps(artifactURLCollection, artifactURLs, pth, false)
		case zippedXcarchiveExt:
			log.Donef("Uploading xcarchive file: %s", pth)

			artifactURLs, err := uploaders.DeployXcarchive(item, config.BuildURL, config.APIToken)
			if err != nil {
				return ArtifactURLCollection{}, fmt.Errorf("deploy failed, error: %s", err)
			}
			fillURLMaps(artifactURLCollection, artifactURLs, pth, false)
		default:
			log.Donef("Uploading file: %s", pth)

			artifactURLs, err := uploaders.DeployFile(item, config.BuildURL, config.APIToken)
			if err != nil {
				return ArtifactURLCollection{}, fmt.Errorf("deploy failed, error: %s", err)
			}

			fillURLMaps(artifactURLCollection, artifactURLs, pth, config.IsPublicPageEnabled)
		}
	}
	return artifactURLCollection, nil
}

func fillURLMaps(artifactURLCollection ArtifactURLCollection, artifactURLs uploaders.ArtifactURLs, apk string, tryPublic bool) {
	if tryPublic && artifactURLs.PublicInstallPageURL != "" {
		artifactURLCollection.PublicInstallPageURLs[filepath.Base(apk)] = artifactURLs.PublicInstallPageURL
	}
	if artifactURLs.PermanentDownloadURL != "" {
		artifactURLCollection.PermanentDownloadURLs[filepath.Base(apk)] = artifactURLs.PermanentDownloadURL
	}
}

func getFileType(pth string) string {
	if strings.HasSuffix(pth, zippedXcarchiveExt) {
		return zippedXcarchiveExt
	}
	return filepath.Ext(pth)
}

func validateGoTemplate(publicInstallPageMapFormat string) error {
	temp := template.New("Public Install Page Map template")

	_, err := temp.Parse(publicInstallPageMapFormat)
	return err
}
