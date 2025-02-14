package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/envman/envman"
	androidparser "github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-android/v2/metaparser/bundletool"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-steputils/v2/secretkeys"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/errorutil"
	"github.com/bitrise-io/go-utils/v2/exitcode"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	loggerV2 "github.com/bitrise-io/go-utils/v2/log"
	pathutil2 "github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-utils/ziputil"
	iosparser "github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/fileredactor"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/report"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/uploaders"
)

var fileBaseNamesToSkip = []string{".DS_Store"}

// Config ...
type Config struct {
	PipelineIntermediateFiles     string `env:"pipeline_intermediate_files"`
	BuildURL                      string `env:"build_url,required"`
	APIToken                      string `env:"build_api_token,required"`
	IsCompress                    bool   `env:"is_compress,opt[true,false]"`
	ZipName                       string `env:"zip_name"`
	DeployPath                    string `env:"deploy_path"`
	NotifyUserGroups              string `env:"notify_user_groups"`
	AlwaysNotifyUserGroups        string `env:"always_notify_user_groups"`
	NotifyEmailList               string `env:"notify_email_list"`
	IsPublicPageEnabled           bool   `env:"is_enable_public_page,opt[true,false]"`
	PublicInstallPageMapFormat    string `env:"public_install_page_url_map_format,required"`
	PermanentDownloadURLMapFormat string `env:"permanent_download_url_map_format,required"`
	DetailsPageURLMapFormat       string `env:"details_page_url_map_format,required"`
	BuildSlug                     string `env:"BITRISE_BUILD_SLUG,required"`
	TestDeployDir                 string `env:"BITRISE_TEST_DEPLOY_DIR,required"`
	AppSlug                       string `env:"BITRISE_APP_SLUG,required"`
	AddonAPIBaseURL               string `env:"addon_api_base_url,required"`
	AddonAPIToken                 string `env:"addon_api_token"`
	FilesToRedact                 string `env:"files_to_redact"`
	DebugMode                     bool   `env:"debug_mode,opt[true,false]"`
	BundletoolVersion             string `env:"bundletool_version,required"`
	UploadConcurrency             string `env:"BITRISE_DEPLOY_UPLOAD_CONCURRENCY"`
	HTMLReportDir                 string `env:"BITRISE_HTML_REPORT_DIR"`
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
	DetailsPageURLs       map[string]string
}

const zippedXcarchiveExt = ".xcarchive.zip"

func fail(logger loggerV2.Logger, format string, v ...interface{}) {
	logger.Errorf(format, v...)
	os.Exit(int(exitcode.Failure))
}

func main() {
	logger := loggerV2.NewLogger() // TODO: replace v1 logger with v2 logger all around the code

	var config Config
	if err := stepconf.Parse(&config); err != nil {
		fail(logger, "Issue with input: %s", err)
	}

	stepconf.Print(config)

	log.SetEnableDebugLog(config.DebugMode)
	logger.EnableDebugLog(config.DebugMode)

	if err := validateGoTemplate(config.PublicInstallPageMapFormat); err != nil {
		fail(logger, "PublicInstallPageMapFormat - %s", err)
	}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("__deploy-to-bitrise-io__")
	if err != nil {
		fail(logger, "Failed to create tmp dir, error: %s", err)
	}

	logger.Println()
	logger.Infof("Collecting files to redact...")

	pathModifier := pathutil2.NewPathModifier()
	pathChecker := pathutil2.NewPathChecker()
	pathProcessor := fileredactor.NewFilePathProcessor(pathModifier, pathChecker)
	filePaths, err := pathProcessor.ProcessFilePaths(config.FilesToRedact)
	if err != nil {
		fail(logger, errorutil.FormattedError(fmt.Errorf("failed to collect file paths to redact: %w", err)))
	}

	if len(filePaths) > 0 {
		logger.Printf("List of files to redact:")
		for _, path := range filePaths {
			logger.Printf("- %s", path)
		}

		fileManager := fileutil.NewFileManager()
		redactor := fileredactor.NewFileRedactor(fileManager)
		secrets := loadSecrets()
		err = redactor.RedactFiles(filePaths, secrets)
		if err != nil {
			fail(logger, errorutil.FormattedError(fmt.Errorf("failed to redact files: %w", err)))
		}
	} else {
		logger.Printf("No files to redact...")
	}

	logger.Println()
	logger.Infof("Collecting files to deploy...")

	var deployableItems []deployment.DeployableItem
	if strings.TrimSpace(config.DeployPath) != "" {
		absDeployPth, err := pathutil.AbsPath(config.DeployPath)
		if err != nil {
			fail(logger, "Failed to expand path: %s, error: %s", config.DeployPath, err)
		}

		filesToDeploy, err := collectFilesToDeploy(absDeployPth, config, tmpDir, logger)
		if err != nil {
			fail(logger, "%s", err)
		}
		clearedFilesToDeploy := clearDeployFiles(filesToDeploy, logger)
		deployableItems = deployment.ConvertPaths(clearedFilesToDeploy)
	}

	if strings.TrimSpace(config.PipelineIntermediateFiles) != "" {
		zipComparator := deployment.NewZipComparator(deployment.DefaultReadZipFunction)
		repository := env.NewRepository()
		collector := deployment.NewCollector(zipComparator, deployment.DefaultIsDirFunction, ziputil.ZipDir, repository, tmpDir)
		deployableItems, err = collector.AddIntermediateFiles(deployableItems, config.PipelineIntermediateFiles)
		if err != nil {
			fail(logger, "%s", err)
		}
	}

	if len(deployableItems) == 0 {
		logger.Printf("No deployment files were defined. Please check the deploy_path and pipeline_intermediate_files inputs.")
	} else {
		logger.Printf("List of files to deploy (%d):", len(deployableItems))
		logDeployFiles(deployableItems, logger)

		logger.Println()
		logger.Infof("Deploying files...")
		artifactURLCollection, errors := deploy(deployableItems, config, logger)
		if len(errors) > 0 {
			logger.Println()

			var errMessage string
			for _, err := range errors {
				errMessage += errorutil.FormattedError(err)
			}

			fail(logger, "%s", errMessage)
		}

		logger.Donef("Success")
		logger.Printf("You can find the Build Artifact on the Build's page: %s", config.BuildURL)

		if err := exportInstallPages(artifactURLCollection, config, logger); err != nil {
			fail(logger, "%s", err)
		}
	}

	if config.AddonAPIToken != "" {
		deployTestResults(config, logger)
	}

	if config.HTMLReportDir != "" {
		deployHTMLReports(config, logger)
	}
}

func deployHTMLReports(config Config, logger loggerV2.Logger) {
	logger.Println()
	logger.Infof("Deploying html reports...")

	concurrency := determineConcurrency(Config{})
	uploader := report.NewHTMLReportUploader(config.HTMLReportDir, config.BuildURL, config.APIToken, concurrency, logger)

	uploadErrors := uploader.DeployReports()
	if 0 < len(uploadErrors) {
		logger.Errorf("Failed to upload html reports:")
		for _, err := range uploadErrors {
			logger.Errorf("- %w", err)
		}
	} else {
		logger.Donef("Successful html report upload")
	}
}

func loadSecrets() []string {
	envRepository := env.NewRepository()
	keys := secretkeys.NewManager().Load(envRepository)

	var values []string
	for _, key := range keys {
		value := envRepository.Get(key)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func exportInstallPages(artifactURLCollection ArtifactURLCollection, config Config, logger loggerV2.Logger) error {
	if len(artifactURLCollection.PublicInstallPageURLs) > 0 {
		pages := mapURLsToInstallPages(artifactURLCollection.PublicInstallPageURLs)

		if err := tools.ExportEnvironmentWithEnvman("BITRISE_PUBLIC_INSTALL_PAGE_URL", pages[0].URL); err != nil {
			return fmt.Errorf("failed to export BITRISE_PUBLIC_INSTALL_PAGE_URL: %s", err)
		}
		logger.Printf("The public install page url is now available in the Environment Variable: BITRISE_PUBLIC_INSTALL_PAGE_URL (value: %s)\n", pages[0].URL)

		value, err := exportMapEnvironment("Public Install Page template", config.PublicInstallPageMapFormat, "PublicInstallPageMap", "BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP", pages, logger)
		if err != nil {
			return fmt.Errorf("failed to export BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP, error: %s", err)
		}
		logger.Printf("A map of deployed files and their public install page urls is now available in the Environment Variable: BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP (value: %s)", value)
	}
	if len(artifactURLCollection.PermanentDownloadURLs) > 0 {
		pages := mapURLsToInstallPages(artifactURLCollection.PermanentDownloadURLs)
		value, err := exportMapEnvironment("Permanent Download URL template", config.PermanentDownloadURLMapFormat, "PermanentDownloadURLMap", "BITRISE_PERMANENT_DOWNLOAD_URL_MAP", pages, logger)
		if err != nil {
			return fmt.Errorf("failed to export BITRISE_PERMANENT_DOWNLOAD_URL_MAP: %s", err)
		}
		logger.Printf("A map of deployed files and their permanent download urls is now available in the Environment Variable: BITRISE_PERMANENT_DOWNLOAD_URL_MAP (value: %s)", value)
	}
	if len(artifactURLCollection.DetailsPageURLs) > 0 {
		pages := mapURLsToInstallPages(artifactURLCollection.DetailsPageURLs)

		if err := tools.ExportEnvironmentWithEnvman("BITRISE_ARTIFACT_DETAILS_PAGE_URL", pages[0].URL); err != nil {
			return fmt.Errorf("failed to export BITRISE_ARTIFACT_DETAILS_PAGE_URL: %s", err)
		}
		log.Printf("The artifact details page url is now available in the Environment Variable: BITRISE_ARTIFACT_DETAILS_PAGE_URL (value: %s)\n", pages[0].URL)

		value, err := exportMapEnvironment("Details Page URL template", config.DetailsPageURLMapFormat, "DetailsPageURLMap", "BITRISE_ARTIFACT_DETAILS_PAGE_URL_MAP", pages, logger)
		if err != nil {
			return fmt.Errorf("failed to export BITRISE_ARTIFACT_DETAILS_PAGE_URL_MAP, error: %s", err)
		}
		log.Printf("A map of deployed files and their details page urls is now available in the Environment Variable: BITRISE_ARTIFACT_DETAILS_PAGE_URL_MAP (value: %s)", value)
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

func exportMapEnvironment(templateName string, format string, formatName string, outputKey string, pages []PublicInstallPage, logger loggerV2.Logger) (string, error) {
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
		logger.Warnf("too many artifacts, not all urls has been written to output: %s", outputKey)
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

func logDeployFiles(files []deployment.DeployableItem, logger loggerV2.Logger) {
	for _, file := range files {
		message := fmt.Sprintf("- %s", file.Path)

		if file.IntermediateFileMeta != nil {
			message += " (pipeline intermediate file)"
		}

		logger.Printf("%s", message)
	}
}

func clearDeployFiles(filesToDeploy []string, logger loggerV2.Logger) []string {
	var clearedFilesToDeploy []string
	for _, pth := range filesToDeploy {
		for _, fileBaseNameToSkip := range fileBaseNamesToSkip {
			if filepath.Base(pth) == fileBaseNameToSkip {
				logger.Warnf("skipping: %s", pth)
			} else {
				clearedFilesToDeploy = append(clearedFilesToDeploy, pth)
			}

		}
	}
	return clearedFilesToDeploy
}

func collectFilesToDeploy(absDeployPth string, config Config, tmpDir string, logger loggerV2.Logger) (filesToDeploy []string, err error) {
	pathExists, err := pathutil.IsPathExists(absDeployPth)
	if err != nil {
		return nil, fmt.Errorf("failed to check if %s exists: %s", absDeployPth, err)
	}
	if !pathExists {
		logger.Warnf("Nothing to deploy at %s", absDeployPth)
		return
	}

	isDeployPathDir, err := pathutil.IsDirExists(absDeployPth)
	if err != nil {
		return nil, fmt.Errorf("failed to check if %s is a directory or a file: %s", absDeployPth, err)
	}

	if !isDeployPathDir {
		logger.Printf("Build Artifact deployment mode: deploying single file")

		filesToDeploy = []string{absDeployPth}
	} else if config.IsCompress {
		logger.Printf("Build Artifact deployment mode: deploying compressed deploy directory")

		entries, err := os.ReadDir(absDeployPth)
		if err != nil {
			return nil, fmt.Errorf("read contents of %s: %s", absDeployPth, err)
		}
		if len(entries) == 0 {
			logger.Donef("Directory is empty, nothing to compress")
			return nil, nil
		}

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
		logger.Printf("Build Artifact deployment mode: deploying the content of the deploy directory")

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

func stepNameWithIndex(stepInfo models.TestResultStepInfo) string {
	name := stepInfo.Title
	if len(name) == 0 {
		name = stepInfo.ID + "@" + stepInfo.Version
	}
	return fmt.Sprintf("%d. Step (%s)", stepInfo.Number, name)
}

func deployTestResults(config Config, logger loggerV2.Logger) {
	logger.Println()
	logger.Infof("Collecting test results...")
	testResults, err := test.ParseTestResults(config.TestDeployDir, logger)
	if err != nil {
		logger.Warnf("Failed to parse test results: %s", err)
		return
	}
	if len(testResults) == 0 {
		logger.Printf("No test results found")
		return
	}

	for i, result := range testResults {
		if i == 0 {
			logger.Printf("List of test results:")
		}
		if len(result.ImagePaths) > 0 {
			logger.Printf("- %s (generated by the %s) with %d attachment(s):", result.Name, stepNameWithIndex(result.StepInfo), len(result.ImagePaths))
			for _, pth := range result.ImagePaths {
				logger.Printf("  - %s", pth)
			}
		} else {
			logger.Printf("- %s (generated by the %s)", result.Name, stepNameWithIndex(result.StepInfo))
		}
	}

	logger.Println()
	logger.Infof("Deploying test results...")
	if err := testResults.Upload(config.AddonAPIToken, config.AddonAPIBaseURL, config.AppSlug, config.BuildSlug, logger); err != nil {
		logger.Warnf("Failed to deploy test results: %s", err)
	} else {
		logger.Donef("Success")
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

func deploy(deployableItems []deployment.DeployableItem, config Config, logger loggerV2.Logger) (ArtifactURLCollection, []error) {
	apks, aabs, others := findAPKsAndAABs(deployableItems)

	var androidArtifacts []string
	for _, artifacts := range append(apks, aabs...) {
		androidArtifacts = append(androidArtifacts, artifacts.Path)
	}

	artifactURLCollection := ArtifactURLCollection{
		PublicInstallPageURLs: map[string]string{},
		PermanentDownloadURLs: map[string]string{},
		DetailsPageURLs:       map[string]string{},
	}
	var err error
	var errorCollection []error
	var wg sync.WaitGroup

	concurrency := determineConcurrency(config)
	jobs := make(chan bool, concurrency)
	combinedItems := append(append(apks, aabs...), others...)
	mapLock := &sync.RWMutex{}
	errLock := &sync.RWMutex{}

	var bTool bundletool.Path
	if len(aabs) > 0 {
		bTool, err = bundletool.New(config.BundletoolVersion)
		if err != nil {
			errorCollection = handleDeploymentFailureError(err, errorCollection, logger)
		}
	}
	fileManager := fileutil.NewFileManager()
	uploader := uploaders.New(
		logger,
		fileManager,
		androidparser.New(uploaders.NewLogger(), bTool, fileManager),
		iosparser.New(logger, fileManager),
	)

	for _, item := range combinedItems {
		wg.Add(1)

		go func(item deployment.DeployableItem) {
			defer wg.Done()

			jobs <- true

			artifactURLs, err := deploySingleItem(logger, uploader, item, config, androidArtifacts)
			if err != nil {
				errLock.Lock()
				errorCollection = handleDeploymentFailureError(err, errorCollection, logger)
				errLock.Unlock()
			} else {
				fillURLMaps(mapLock, artifactURLCollection, artifactURLs, item.Path, config.IsPublicPageEnabled)
			}

			<-jobs
		}(item)
	}

	wg.Wait()

	return artifactURLCollection, errorCollection
}

func deploySingleItem(logger loggerV2.Logger, uploader *uploaders.Uploader, item deployment.DeployableItem, config Config, androidArtifacts []string) (uploaders.ArtifactURLs, error) {
	pth := item.Path
	fileType := getFileType(pth)

	defer logger.Println()

	switch fileType {
	case ".apk":
		logger.Printf("Deploying apk file: %s", pth)

		return uploader.DeployAPK(item, androidArtifacts, config.BuildURL, config.APIToken, config.NotifyUserGroups, config.AlwaysNotifyUserGroups, config.NotifyEmailList, config.IsPublicPageEnabled)
	case ".aab":
		logger.Printf("Deploying aab file: %s", pth)

		return uploader.DeployAAB(item, androidArtifacts, config.BuildURL, config.APIToken)
	case ".ipa":
		logger.Printf("Deploying ipa file: %s", pth)

		return uploader.DeployIPA(item, config.BuildURL, config.APIToken, config.NotifyUserGroups, config.AlwaysNotifyUserGroups, config.NotifyEmailList, config.IsPublicPageEnabled)
	case zippedXcarchiveExt:
		logger.Printf("Deploying xcarchive file: %s", pth)

		return uploader.DeployXcarchive(item, config.BuildURL, config.APIToken)
	default:
		return uploader.DeployFile(item, config.BuildURL, config.APIToken)
	}
}

func handleDeploymentFailureError(err error, errorCollection []error, logger loggerV2.Logger) []error {
	logger.Errorf("%s", errorutil.FormattedError(err))
	err = fmt.Errorf("deploy failed, error: %w", err)
	errorCollection = append(errorCollection, err)
	return errorCollection
}

func fillURLMaps(lock *sync.RWMutex, artifactURLCollection ArtifactURLCollection, artifactURLs uploaders.ArtifactURLs, path string, tryPublic bool) {
	lock.Lock()
	defer lock.Unlock()

	if tryPublic && artifactURLs.PublicInstallPageURL != "" {
		artifactURLCollection.PublicInstallPageURLs[filepath.Base(path)] = artifactURLs.PublicInstallPageURL
	}
	if artifactURLs.PermanentDownloadURL != "" {
		artifactURLCollection.PermanentDownloadURLs[filepath.Base(path)] = artifactURLs.PermanentDownloadURL
	}
	if artifactURLs.DetailsPageURL != "" {
		artifactURLCollection.DetailsPageURLs[filepath.Base(path)] = artifactURLs.DetailsPageURL
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

func determineConcurrency(config Config) int {
	if config.UploadConcurrency == "" {
		return 1
	}

	value, err := strconv.Atoi(config.UploadConcurrency)
	if err != nil {
		return 1
	}

	if value < 1 {
		return 1
	}

	if value > 20 {
		return 20
	}

	return value
}
