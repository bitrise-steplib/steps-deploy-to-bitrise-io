package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/ziputil"
	"github.com/bitrise-io/steps-deploy-to-bitrise-io/uploaders"
	"github.com/bitrise-tools/go-steputils/input"
	"github.com/bitrise-tools/go-steputils/tools"
)

var fileBaseNamesToSkip = []string{".DS_Store"}

// ConfigsModel ...
type ConfigsModel struct {
	BuildURL            string
	APIToken            string
	IsCompress          string
	DeployPath          string
	NotifyUserGroups    string
	NotifyEmailList     string
	IsPublicPageEnabled string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		BuildURL:            os.Getenv("build_url"),
		APIToken:            os.Getenv("build_api_token"),
		IsCompress:          os.Getenv("is_compress"),
		DeployPath:          os.Getenv("deploy_path"),
		NotifyUserGroups:    os.Getenv("notify_user_groups"),
		NotifyEmailList:     os.Getenv("notify_email_list"),
		IsPublicPageEnabled: os.Getenv("is_enable_public_page"),
	}
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfNotEmpty(configs.BuildURL); err != nil {
		return fmt.Errorf("BuildURL - %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.APIToken); err != nil {
		return fmt.Errorf("APIToken - %s", err)
	}
	if err := input.ValidateWithOptions(configs.IsCompress, "true", "false"); err != nil {
		return fmt.Errorf("IsCompress - %s", err)
	}
	if err := input.ValidateIfPathExists(configs.DeployPath); err != nil {
		return fmt.Errorf("DeployPath - %s", err)
	}
	if err := input.ValidateWithOptions(configs.IsPublicPageEnabled, "true", "false"); err != nil {
		return fmt.Errorf("IsPublicPageEnabled - %s", err)
	}
	return nil
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- BuildURL: %s", configs.BuildURL)
	log.Printf("- APIToken: %s", configs.APIToken)
	log.Printf("- IsCompress: %s", configs.IsCompress)
	log.Printf("- DeployPath: %s", configs.DeployPath)
	log.Printf("- NotifyUserGroups: %s", configs.NotifyUserGroups)
	log.Printf("- NotifyEmailList: %s", configs.NotifyEmailList)
	log.Printf("- IsPublicPageEnabled: %s", configs.IsPublicPageEnabled)
}

func fail(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		fail("Issue with input: %s", err)
	}

	filesToDeploy := []string{}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("__deploy-to-bitrise-io__")
	if err != nil {
		fail("Failed to create tmp dir, error: %s", err)
	}

	// Collect files to deploy
	isDeployPathDir, err := pathutil.IsDirExists(configs.DeployPath)
	if err != nil {
		fail("Failed to check if DeployPath (%s) is a directory or a file, error: %s", configs.DeployPath, err)
	}

	if !isDeployPathDir {
		fmt.Println()
		log.Infof("Deploying single file")

		filesToDeploy = []string{configs.DeployPath}
	} else if configs.IsCompress == "true" {
		fmt.Println()
		log.Infof("Deploying compressed Deploy directory")

		dirName := filepath.Base(configs.DeployPath)
		tmpZipPath := filepath.Join(tmpDir, dirName+".zip")

		if err := ziputil.ZipDir(configs.DeployPath, tmpZipPath, true); err != nil {
			fail("Failed to zip output dir, error: %s", err)
		}

		filesToDeploy = []string{tmpZipPath}
	} else {
		fmt.Println()
		log.Infof("Deploying the content of the Deploy directory separately")

		pattern := filepath.Join(configs.DeployPath, "*")
		pths, err := filepath.Glob(pattern)
		if err != nil {
			fail("Failed to list files in DeployPath, error: %s", err)
		}

		for _, pth := range pths {
			if isDir, err := pathutil.IsDirExists(pth); err != nil {
				fail("Failed to check if path (%s) is a directory or a file, error: %s", pth, err)
			} else if !isDir {
				filesToDeploy = append(filesToDeploy, pth)
			}
		}
	}

	clearedFilesToDeploy := []string{}
	for _, pth := range filesToDeploy {
		for _, fileBaseNameToSkip := range fileBaseNamesToSkip {
			if filepath.Base(pth) == fileBaseNameToSkip {
				log.Warnf("skipping: %s", pth)
			} else {
				clearedFilesToDeploy = append(clearedFilesToDeploy, pth)
			}

		}
	}

	fmt.Println()
	log.Infof("List of files to deploy")
	for _, pth := range clearedFilesToDeploy {
		log.Printf("- %s", pth)
	}
	// ---

	// Deploy files
	fmt.Println()
	log.Infof("Deploying files")

	publicInstallPage := ""

	for _, pth := range clearedFilesToDeploy {
		ext := filepath.Ext(pth)

		fmt.Println()

		switch ext {
		case ".ipa":
			log.Donef("Uploading ipa file: %s", pth)

			installPage, err := uploaders.DeployIPA(pth, configs.BuildURL, configs.APIToken, configs.NotifyUserGroups, configs.NotifyEmailList, configs.IsPublicPageEnabled)
			if err != nil {
				fail("Deploy failed, error: %s", err)
			}

			if installPage != "" {
				publicInstallPage = installPage
			}
		case ".apk":
			log.Donef("Uploading apk file: %s", pth)

			installPage, err := uploaders.DeployAPK(pth, configs.BuildURL, configs.APIToken, configs.NotifyUserGroups, configs.NotifyEmailList, configs.IsPublicPageEnabled)
			if err != nil {
				fail("Deploy failed, error: %s", err)
			}

			if installPage != "" {
				publicInstallPage = installPage
			}
		default:
			log.Donef("Uploading file: %s", pth)

			installPage, err := uploaders.DeployFile(pth, configs.BuildURL, configs.APIToken, configs.NotifyUserGroups, configs.NotifyEmailList, configs.IsPublicPageEnabled)
			if err != nil {
				fail("Deploy failed, error: %s", err)
			}

			if installPage != "" {
				publicInstallPage = installPage
			} else if configs.IsPublicPageEnabled == "true" {
				log.Warnf("is_enable_public_page is set, but public download isn't allowed for this type of file")
			}
		}
	}

	fmt.Println()
	log.Donef("Success")
	log.Printf("You can find the Artifact on Bitrise, on the Build's page: %s", configs.BuildURL)

	if publicInstallPage != "" {
		if err := tools.ExportEnvironmentWithEnvman("BITRISE_PUBLIC_INSTALL_PAGE_URL", publicInstallPage); err != nil {
			fail("Failed to export BITRISE_PUBLIC_INSTALL_PAGE_URL, error: %s", err)
		}
		log.Printf("The public install page url is now available in the Environment Variable: BITRISE_PUBLIC_INSTALL_PAGE_URL (value: %s)", publicInstallPage)
	}
	// --

}
