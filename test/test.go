package test

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	logV2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/retryhttp"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters"
	"github.com/hashicorp/go-retryablehttp"
)

// FileInfo ...
type FileInfo struct {
	FileName string `json:"filename"`
	FileSize int    `json:"filesize"`
}

// UploadURL ...
type UploadURL struct {
	FileName string `json:"filename"`
	URL      string `json:"upload_url"`
}

// UploadRequest ...
type UploadRequest struct {
	Name   string                    `json:"name"`
	Step   models.TestResultStepInfo `json:"step_info"`
	Assets []FileInfo                `json:"assets"`
	FileInfo
}

// UploadResponse ...
type UploadResponse struct {
	ID     string      `json:"id"`
	Assets []UploadURL `json:"assets"`
	UploadURL
}

// Result ...
type Result struct {
	Name       string
	XMLContent []byte
	ImagePaths []string
	StepInfo   models.TestResultStepInfo
}

// Results ...
type Results []Result

func httpCall(apiToken, method, url string, input io.Reader, output interface{}) error {
	if apiToken != "" {
		url = url + "/" + apiToken
	}
	req, err := retryablehttp.NewRequest(method, url, input)
	if err != nil {
		return err
	}

	logger := logV2.NewLogger()
	client := retryhttp.NewClient(logger)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close body: %s", err)
		}
	}()

	if resp.StatusCode < 200 || 299 < resp.StatusCode {
		bodyData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Warnf("Failed to read response: %s", err)
			return fmt.Errorf("unsuccessful status code: %d", resp.StatusCode)
		}
		return fmt.Errorf("unsuccessful status code: %d, response: %s", resp.StatusCode, bodyData)
	}

	if output != nil {
		return json.NewDecoder(resp.Body).Decode(&output)
	}
	return nil
}

func findImages(testDir string) (imageFilePaths []string) {
	for _, ext := range []string{".jpg", ".jpeg", ".png"} {
		if paths, err := filepath.Glob(filepath.Join(testDir, "*"+ext)); err == nil {
			imageFilePaths = append(imageFilePaths, paths...)
		}
		if paths, err := filepath.Glob(filepath.Join(testDir, "*"+strings.ToUpper(ext))); err == nil {
			imageFilePaths = append(imageFilePaths, paths...)
		}
	}
	return
}

/*
ParseTestResults walks through the Test Deploy directory and parses all the Steps' test results.

The Test Deploy directory has the following directory structure:

	test_results ($BITRISE_TEST_DEPLOY_DIR)
	├── step_1_test_results ($BITRISE_TEST_RESULT_DIR)
	│		 ├── step-info.json
	│		 ├── test_run_1
	│		 │		 ├── UnitTest.xml
	│		 │		 └── test-info.json
	│		 └── test_run_2
	│		     ├── UITest.xml
	│		     └── test-info.json
	└── step_2_test_results ($BITRISE_TEST_RESULT_DIR)
	    ├── step-info.json
	    └── test_run
	        ├── results.xml
	        ├── screenshot_1.jpg
	        ├── screenshot_2.jpeg
	        ├── screenshot_3.png
	        └── test-info.json
*/
func ParseTestResults(testsRootDir string) (results Results, err error) {
	// read dirs in base tests dir
	// <root_tests_dir>

	testDirs, err := ioutil.ReadDir(testsRootDir)
	if err != nil {
		return nil, err
	}

	// find test results in each dir, skip if invalid test dir
	for _, testDir := range testDirs {
		// <root_tests_dir>/<test_dir>
		testDirPath := filepath.Join(testsRootDir, testDir.Name())

		if !testDir.IsDir() {
			log.Debugf("%s is not a directory", testDirPath)
			continue
		}

		// read unique test phase dirs
		testPhaseDirs, err := ioutil.ReadDir(testDirPath)
		if err != nil {
			return nil, err
		}

		// find step-info in dir, continue if no step-info.json as this file is only required if step has exported artifacts also
		// <root_tests_dir>/<test_dir>/step-info.json

		stepInfoPth := filepath.Join(testDirPath, "step-info.json")
		if isExists, err := pathutil.IsPathExists(stepInfoPth); err != nil {
			log.Warnf("Failed to check if step-info.json file exists in dir: %s: %s", testDirPath, err)
			continue
		} else if !isExists {
			continue
		}

		var stepInfo *models.TestResultStepInfo
		stepInfoFileContent, err := fileutil.ReadBytesFromFile(stepInfoPth)
		if err != nil {
			log.Warnf("Failed to read step-info.json file in dir: %s, error: %s", testDirPath, err)
			continue
		}

		if err := json.Unmarshal(stepInfoFileContent, &stepInfo); err != nil {
			log.Warnf("Failed to parse step-info.json file in dir: %s, error: %s", testDirPath, err)
			continue
		}

		for _, testPhaseDir := range testPhaseDirs {
			if !testPhaseDir.IsDir() {
				continue
			}

			// <root_tests_dir>/<test_dir>/<unique_dir>
			testPhaseDirPath := filepath.Join(testDirPath, testPhaseDir.Name())
			// read one level of file set only <root_tests_dir>/<test_dir>/<unique_dir>/files_to_get
			testFiles, err := filepath.Glob(filepath.Join(pathutil.EscapeGlobPath(testPhaseDirPath), "*"))
			if err != nil {
				return nil, err
			}

			// get the converter that can manage test type contained in the dir
			for _, converter := range converters.List() {
				log.Debugf("Running converter: %T", converter)

				// skip if couldn't find converter for content type
				detected := converter.Detect(testFiles)

				log.Debugf("known test result detected: %v", detected)

				if detected {
					// test-info.json file is required
					testInfoFileContent, err := fileutil.ReadBytesFromFile(filepath.Join(testPhaseDirPath, "test-info.json"))
					if err != nil {
						return nil, err
					}

					var testInfo struct {
						Name string `json:"test-name"`
					}
					if err := json.Unmarshal(testInfoFileContent, &testInfo); err != nil {
						return nil, err
					}

					junitXML, err := converter.XML()
					if err != nil {
						return nil, err
					}
					xmlData, err := xml.MarshalIndent(junitXML, "", " ")
					if err != nil {
						return nil, err
					}
					xmlData = append([]byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"), xmlData...)

					// so here I will have image paths, xml data, and step info per test dir in a bundle info
					images := findImages(testPhaseDirPath)

					log.Debugf("found images: %d", len(images))

					results = append(results, Result{
						Name:       testInfo.Name,
						XMLContent: xmlData,
						ImagePaths: images,
						StepInfo:   *stepInfo,
					})
				}
			}
		}
	}
	return results, nil
}

// Upload ...
func (results Results) Upload(apiToken, endpointBaseURL, appSlug, buildSlug string) error {
	for _, result := range results {
		log.Printf("Uploading: %s", result.Name)

		uploadReq := UploadRequest{
			FileInfo: FileInfo{
				FileName: "test_result.xml",
				FileSize: len(result.XMLContent),
			},
			Name: result.Name,
			Step: result.StepInfo,
		}
		for _, asset := range result.ImagePaths {
			fi, err := os.Stat(asset)
			if err != nil {
				return fmt.Errorf("failed to get file info for %s: %w", asset, err)
			}
			uploadReq.Assets = append(uploadReq.Assets, FileInfo{
				FileName: filepath.Base(asset),
				FileSize: int(fi.Size()),
			})
		}

		uploadRequestBodyData, err := json.Marshal(uploadReq)
		if err != nil {
			return fmt.Errorf("failed to json encode upload request: %w", err)
		}

		var (
			uploadResponse   UploadResponse
			uploadRequestURL = fmt.Sprintf("%s/apps/%s/builds/%s/test_reports", endpointBaseURL, appSlug, buildSlug)
		)
		if err := httpCall(apiToken, http.MethodPost, uploadRequestURL, bytes.NewReader(uploadRequestBodyData), &uploadResponse); err != nil {
			return fmt.Errorf("failed to initialise test result: %w", err)
		}

		if err := httpCall("", http.MethodPut, uploadResponse.URL, bytes.NewReader(result.XMLContent), nil); err != nil {
			return fmt.Errorf("failed to upload test result xml: %w", err)
		}

		for _, upload := range uploadResponse.Assets {
			for _, file := range result.ImagePaths {
				if filepath.Base(file) == upload.FileName {
					fi, err := os.Open(file)
					if err != nil {
						return fmt.Errorf("failed to open test result attachment (%s): %w", file, err)
					}
					if err := httpCall("", http.MethodPut, upload.URL, fi, nil); err != nil {
						return fmt.Errorf("failed to upload test result attachment (%s): %w", file, err)
					}
					break
				}
			}
		}

		var uploadPatchURL = fmt.Sprintf("%s/apps/%s/builds/%s/test_reports/%s", endpointBaseURL, appSlug, buildSlug, uploadResponse.ID)
		if err := httpCall(apiToken, http.MethodPatch, uploadPatchURL, strings.NewReader(`{"uploaded":true}`), nil); err != nil {
			return fmt.Errorf("failed to finalise test result: %w", err)
		}
	}

	return nil
}
