package test

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	logV2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/retryhttp"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testasset"
	"github.com/hashicorp/go-retryablehttp"
)

// maxTotalXMLSize limits the total size of all XML files uploaded in a single run
const maxTotalXMLSize = 100 * 1024 * 1024 // 100 MiB

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
	Name            string
	XMLContent      []byte
	AttachmentPaths []string
	StepInfo        models.TestResultStepInfo
}

// Results ...
type Results []Result

func httpCall(apiToken, method, url string, input io.Reader, output interface{}, logger logV2.Logger) error {
	if apiToken != "" {
		url = url + "/" + apiToken
	}
	req, err := retryablehttp.NewRequest(method, url, input)
	if err != nil {
		return err
	}

	client := retryhttp.NewClient(logger)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warnf("Failed to close body: %s", err)
		}
	}()

	if resp.StatusCode < 200 || 299 < resp.StatusCode {
		bodyData, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Warnf("Failed to read response: %s", err)
			return fmt.Errorf("unsuccessful status code: %d", resp.StatusCode)
		}
		return fmt.Errorf("unsuccessful status code: %d, response: %s", resp.StatusCode, bodyData)
	}

	if output != nil {
		return json.NewDecoder(resp.Body).Decode(&output)
	}
	return nil
}

func findSupportedAttachments(testDir string, logger logV2.Logger) (attachmentPaths []string) {
	err := filepath.WalkDir(testDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if testasset.IsSupportedAssetType(path) {
			attachmentPaths = append(attachmentPaths, path)
		}

		return nil
	})

	if err != nil {
		logger.Warnf("Failed to walk test dir (%s): %s", testDir, err)
		return nil
	}

	return
}

/*
ParseTestResults walks through the Test Deploy directory and parses all the Steps' test results.

The Test Deploy directory has the following directory structure:

	test_results ($BITRISE_TEST_DEPLOY_DIR)
	├── step_1_test_results ($BITRISE_TEST_RESULT_DIR)
	│	├── step-info.json
	│	├── test_run_1
	│	│	├── UnitTest.xml
	│	│	└── test-info.json
	│	└── test_run_2
	│		├── UITest.xml
	│		└── test-info.json
	└── step_2_test_results ($BITRISE_TEST_RESULT_DIR)
		├── step-info.json
		└── test_run
			├── results.xml
			├── screenshot_1.jpg
			├── screenshot_2.jpeg
			├── screenshot_3.png
			└── test-info.json
*/
func ParseTestResults(testsRootDir string, useLegacyXCResultExtractionMethod bool, logger logV2.Logger) (results Results, err error) {
	// read dirs in base tests dir
	// <root_tests_dir>

	testDirs, err := os.ReadDir(testsRootDir)
	if err != nil {
		return nil, err
	}

	// find test results in each dir, skip if invalid test dir
	for _, testDir := range testDirs {
		// <root_tests_dir>/<test_dir>
		testDirPath := filepath.Join(testsRootDir, testDir.Name())

		if !testDir.IsDir() {
			logger.Debugf("%s is not a directory", testDirPath)
			continue
		}

		// read unique test phase dirs
		testPhaseDirs, err := os.ReadDir(testDirPath)
		if err != nil {
			return nil, err
		}

		// find step-info in dir, continue if no step-info.json as this file is only required if step has exported artifacts also
		// <root_tests_dir>/<test_dir>/step-info.json

		stepInfoPth := filepath.Join(testDirPath, "step-info.json")
		if isExists, err := pathutil.IsPathExists(stepInfoPth); err != nil {
			logger.Warnf("Failed to check if step-info.json file exists in dir: %s: %s", testDirPath, err)
			continue
		} else if !isExists {
			continue
		}

		var stepInfo *models.TestResultStepInfo
		stepInfoFileContent, err := fileutil.ReadBytesFromFile(stepInfoPth)
		if err != nil {
			logger.Warnf("Failed to read step-info.json file in dir: %s, error: %s", testDirPath, err)
			continue
		}

		if err := json.Unmarshal(stepInfoFileContent, &stepInfo); err != nil {
			logger.Warnf("Failed to parse step-info.json file in dir: %s, error: %s", testDirPath, err)
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
				logger.Debugf("Running converter: %T", converter)

				converter.Setup(useLegacyXCResultExtractionMethod)

				// skip if it couldn't find a converter for the content type
				detected := converter.Detect(testFiles)

				logger.Debugf("known test result detected: %v", detected)

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

					testReport, err := converter.Convert()
					if err != nil {
						return nil, err
					}
					xmlData, err := xml.MarshalIndent(testReport, "", " ")
					if err != nil {
						return nil, err
					}
					xmlData = append([]byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"), xmlData...)

					attachments := findSupportedAttachments(testPhaseDirPath, logger)

					logger.Debugf("found attachments: %d", len(attachments))

					results = append(results, Result{
						Name:            testInfo.Name,
						XMLContent:      xmlData,
						AttachmentPaths: attachments,
						StepInfo:        *stepInfo,
					})
				}
			}
		}
	}
	return results, nil
}

// Upload ...
func (results Results) Upload(apiToken, endpointBaseURL, appSlug, buildSlug string, logger logV2.Logger) error {
	if results.calculateTotalSizeOfXMLContent() > maxTotalXMLSize {
		return fmt.Errorf("the total size of the test result XML files (%d MiB) exceeds the maximum allowed size of 100 MiB", results.calculateTotalSizeOfXMLContent()/1024/1024)
	}

	for _, result := range results {
		logger.Printf("Uploading: %s", result.Name)

		uploadReq := UploadRequest{
			FileInfo: FileInfo{
				FileName: "test_result.xml",
				FileSize: len(result.XMLContent),
			},
			Name: result.Name,
			Step: result.StepInfo,
		}
		for _, asset := range result.AttachmentPaths {
			fi, err := os.Stat(asset)
			if err != nil {
				return fmt.Errorf("failed to get file info for %s: %w", asset, err)
			}
			uploadReq.Assets = append(uploadReq.Assets, FileInfo{
				FileName: relativeFilePath(asset, result.Name),
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
		if err := httpCall(apiToken, http.MethodPost, uploadRequestURL, bytes.NewReader(uploadRequestBodyData), &uploadResponse, logger); err != nil {
			return fmt.Errorf("failed to initialise test result: %w", err)
		}

		if err := httpCall("", http.MethodPut, uploadResponse.URL, bytes.NewReader(result.XMLContent), nil, logger); err != nil {
			return fmt.Errorf("failed to upload test result xml: %w", err)
		}

		for _, upload := range uploadResponse.Assets {
			for _, file := range result.AttachmentPaths {
				if relativeFilePath(file, result.Name) == upload.FileName {
					fi, err := os.Open(file)
					if err != nil {
						return fmt.Errorf("failed to open test result attachment (%s): %w", file, err)
					}
					if err := httpCall("", http.MethodPut, upload.URL, fi, nil, logger); err != nil {
						return fmt.Errorf("failed to upload test result attachment (%s): %w", file, err)
					}
					break
				}
			}
		}

		var uploadPatchURL = fmt.Sprintf("%s/apps/%s/builds/%s/test_reports/%s", endpointBaseURL, appSlug, buildSlug, uploadResponse.ID)
		if err := httpCall(apiToken, http.MethodPatch, uploadPatchURL, strings.NewReader(`{"uploaded":true}`), nil, logger); err != nil {
			return fmt.Errorf("failed to finalise test result: %w", err)
		}
	}

	return nil
}

func (results Results) calculateTotalSizeOfXMLContent() int {
	totalSize := 0
	for _, result := range results {
		totalSize += len(result.XMLContent)
	}
	return totalSize
}

func relativeFilePath(absoluteFilePath, reportName string) string {
	pathComponent := string(filepath.Separator) + reportName + string(filepath.Separator)
	if strings.Contains(absoluteFilePath, pathComponent) {
		return strings.SplitN(absoluteFilePath, pathComponent, 2)[1]
	}
	return filepath.Base(absoluteFilePath)
}
