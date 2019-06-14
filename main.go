package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/ziputil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/uploaders"
)

var fileBaseNamesToSkip = []string{".DS_Store"}

// Config ...
type Config struct {
	BuildURL                   string `env:"build_url,required"`
	APIToken                   string `env:"build_api_token,required"`
	IsCompress                 string `env:"is_compress,opt[true,false]"`
	ZipName                    string `env:"zip_name"`
	DeployPath                 string `env:"deploy_path,required"`
	NotifyUserGroups           string `env:"notify_user_groups"`
	NotifyEmailList            string `env:"notify_email_list"`
	IsPublicPageEnabled        string `env:"is_enable_public_page,opt[true,false]"`
	PublicInstallPageMapFormat string `env:"public_install_page_url_map_format,required"`
	BuildSlug                  string `env:"BITRISE_BUILD_SLUG,required"`
	TestDeployDir              string `env:"BITRISE_TEST_DEPLOY_DIR,required"`
	AppSlug                    string `env:"BITRISE_APP_SLUG,required"`
	AddonAPIBaseURL            string `env:"addon_api_base_url,required"`
	AddonAPIToken              string `env:"addon_api_token"`
	DebugMode                  bool   `env:"debug_mode,opt[true,false]"`
}

// PublicInstallPage ...
type PublicInstallPage struct {
	File string
	URL  string
}

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

	filesToDeploy := []string{}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("__deploy-to-bitrise-io__")
	if err != nil {
		fail("Failed to create tmp dir, error: %s", err)
	}

	// Collect files to deploy
	isDeployPathDir, err := pathutil.IsDirExists(absDeployPth)
	if err != nil {
		fail("Failed to check if DeployPath (%s) is a directory or a file, error: %s", absDeployPth, err)
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
			fail("Failed to zip output dir, error: %s", err)
		}

		filesToDeploy = []string{tmpZipPath}
	} else {
		fmt.Println()
		log.Infof("Deploying the content of the Deploy directory separately")

		pattern := filepath.Join(absDeployPth, "*")
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

	publicInstallPages := make(map[string]string)

	for _, pth := range clearedFilesToDeploy {
		ext := filepath.Ext(pth)

		fmt.Println()

		switch ext {
		case ".ipa":
			log.Donef("Uploading ipa file: %s", pth)

			installPage, err := uploaders.DeployIPA(pth, config.BuildURL, config.APIToken, config.NotifyUserGroups, config.NotifyEmailList, config.IsPublicPageEnabled)
			if err != nil {
				fail("Deploy failed, error: %s", err)
			}

			if installPage != "" {
				publicInstallPages[filepath.Base(pth)] = installPage
			}
		case ".apk":
			log.Donef("Uploading apk file: %s", pth)

			installPage, err := uploaders.DeployAPK(pth, config.BuildURL, config.APIToken, config.NotifyUserGroups, config.NotifyEmailList, config.IsPublicPageEnabled)
			if err != nil {
				fail("Deploy failed, error: %s", err)
			}

			if installPage != "" {
				publicInstallPages[filepath.Base(pth)] = installPage
			}
		case ".aab":
			log.Donef("Uploading aab file: %s", pth)

			installPage, err := uploaders.DeployAPK(pth, config.BuildURL, config.APIToken, config.NotifyUserGroups, config.NotifyEmailList, config.IsPublicPageEnabled)
			if err != nil {
				fail("Deploy failed, error: %s", err)
			}

			if installPage != "" {
				publicInstallPages[filepath.Base(pth)] = installPage
			}
		default:
			log.Donef("Uploading file: %s", pth)

			installPage, err := uploaders.DeployFile(pth, config.BuildURL, config.APIToken, config.NotifyUserGroups, config.NotifyEmailList, config.IsPublicPageEnabled)
			if err != nil {
				fail("Deploy failed, error: %s", err)
			}

			if installPage != "" {
				publicInstallPages[filepath.Base(pth)] = installPage
			} else if config.IsPublicPageEnabled == "true" {
				log.Warnf("is_enable_public_page is set, but public download isn't allowed for this type of file")
			}
		}
	}

	fmt.Println()
	log.Donef("Success")
	log.Printf("You can find the Artifact on Bitrise, on the Build's page: %s", config.BuildURL)

	if len(publicInstallPages) > 0 {
		temp := template.New("Public Install Page template")
		var pages []PublicInstallPage
		for file, url := range publicInstallPages {
			pages = append(pages, PublicInstallPage{
				File: file,
				URL:  url,
			})
		}

		if err := tools.ExportEnvironmentWithEnvman("BITRISE_PUBLIC_INSTALL_PAGE_URL", pages[0].URL); err != nil {
			fail("Failed to export BITRISE_PUBLIC_INSTALL_PAGE_URL, error: %s", err)
		}
		log.Printf("The public install page url is now available in the Environment Variable: BITRISE_PUBLIC_INSTALL_PAGE_URL (value: %s)\n", pages[0].URL)

		temp, err := temp.Parse(config.PublicInstallPageMapFormat)
		if err != nil {
			fail("Error during parsing PublicInstallPageMap: ", err)
		}

		buf := new(bytes.Buffer)
		if err := temp.Execute(buf, pages); err != nil {
			fail("Execute: ", err)
		}

		if err := tools.ExportEnvironmentWithEnvman("BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP", buf.String()); err != nil {
			fail("Failed to export BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP, error: %s", err)
		}
		log.Printf("A map of deployed files and their public install page urls is now available in the Environment Variable: BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP (value: %s)", buf.String())
		log.Printf("")
	}

	// Deploy test files
	if config.AddonAPIToken != "" {
		fmt.Println()
		log.Infof("Upload test results")

		testResults, err := test.ParseTestResults(config.TestDeployDir)
		if err != nil {
			fail("Error during parsing test results: ", err)
		}

		log.Printf("- uploading (%d) test results", len(testResults))

		if err := testResults.Upload(config.AddonAPIToken, config.AddonAPIBaseURL, config.AppSlug, config.BuildSlug); err != nil {
			fail("Failed to upload test results: ", err)
		}

		log.Donef("Success")
	}
}

func validateGoTemplate(publicInstallPageMapFormat string) error {
	temp := template.New("Public Install Page Map template")

	_, err := temp.Parse(publicInstallPageMapFormat)
	return err
}
