package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/steps-deploy-to-bitrise-io/test/converters"
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
	Step   models.TestResultStepInfo `json:"step"`
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
	XMLContent []byte
	ImagePaths []string
	StepInfo   models.TestResultStepInfo
}

// Results ...
type Results []Result

func httpCall(apiToken, method, url string, input io.Reader, output interface{}) error {
	req, err := http.NewRequest(method, url+"/"+apiToken, input)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close body, error: %s\n", err)
		}
	}()

	if resp.StatusCode < 200 || 299 < resp.StatusCode {
		bodyData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unsuccessful response code: %d and failed to read body, error: %s", resp.StatusCode, err)
		}
		return fmt.Errorf("unsuccessful response code: %d\nbody:\n%s", resp.StatusCode, bodyData)
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

// ParseTestResults ...
func ParseTestResults(testsRootDir string) (results Results, err error) {
	// read dirs in base tests dir
	testDirs, err := ioutil.ReadDir(testsRootDir)
	if err != nil {
		return nil, err
	}

	// find test results in each dir, skip if invalid test dir
	for _, testDir := range testDirs {
		testDirPath := filepath.Join(testsRootDir, testDir.Name())

		// read one level of file set only <root_tests_dir>/<test_dir>/file_to_get
		testFiles, err := filepath.Glob(filepath.Join(testDirPath, "*"))
		if err != nil {
			return nil, err
		}

		// find step-info in dir
		var stepInfo *models.TestResultStepInfo
		for _, file := range testFiles {
			if strings.HasSuffix(file, "step-info.json") {
				stepInfoFileContent, err := fileutil.ReadBytesFromFile(file)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(stepInfoFileContent, &stepInfo); err != nil {
					return nil, err
				}
			}
		}

		// if no step-info.json then skip
		if stepInfo == nil {
			continue
		}

		// get the converter that can manage test type contained in the dir
		for _, converter := range converters.List() {
			// skip if couldn't find converter for content type
			if converter.Detect(testFiles) {
				xml, err := converter.XML()
				if err != nil {
					return nil, err
				}

				// so here I will have image paths, xml data, and step info per test dir in a bundle info
				results = append(results, Result{
					XMLContent: xml,
					ImagePaths: findImages(filepath.Join(testsRootDir, testDir.Name())),
					StepInfo:   *stepInfo,
				})
			}
		}
	}
	return results, nil
}

// Upload ...
func (results Results) Upload(apiToken, endpointBaseURL, appSlug, buildSlug string) error {
	for _, result := range results {
		uploadReq := UploadRequest{
			FileInfo: FileInfo{
				FileName: "test_result.xml",
				FileSize: len(result.XMLContent),
			},
			Step: result.StepInfo,
		}
		for _, asset := range result.ImagePaths {
			fi, err := os.Stat(asset)
			if err != nil {
				return err
			}
			uploadReq.Assets = append(uploadReq.Assets, FileInfo{
				FileName: filepath.Base(asset),
				FileSize: int(fi.Size()),
			})
		}

		uploadRequestBodyData, err := json.Marshal(uploadReq)
		if err != nil {
			return err
		}

		var uploadResponse UploadResponse
		if err := httpCall(apiToken, http.MethodPost, fmt.Sprintf("%s/apps/%s/builds/%s/test_reports", endpointBaseURL, appSlug, buildSlug), bytes.NewReader(uploadRequestBodyData), &uploadResponse); err != nil {
			return err
		}

		if err := httpCall(apiToken, http.MethodPut, uploadResponse.URL, bytes.NewReader(result.XMLContent), nil); err != nil {
			return err
		}

		for _, upload := range uploadResponse.Assets {
			for _, file := range result.ImagePaths {
				if filepath.Base(file) == upload.FileName {
					fi, err := os.Open(file)
					if err != nil {
						return err
					}
					if err := httpCall(apiToken, http.MethodPut, upload.URL, fi, nil); err != nil {
						return err
					}
					break
				}
			}
		}

		if err := httpCall(apiToken, http.MethodPatch, fmt.Sprintf("%s/apps/%s/builds/%s/test_reports/%s", endpointBaseURL, appSlug, buildSlug, uploadResponse.ID), nil, nil); err != nil {
			return err
		}
	}

	return nil
}
