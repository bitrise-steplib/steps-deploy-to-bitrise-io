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
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters"
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
	req, err := http.NewRequest(method, url, input)
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
	// <root_tests_dir>
	fmt.Println("read root dir:", testsRootDir)
	testDirs, err := ioutil.ReadDir(testsRootDir)
	if err != nil {
		return nil, err
	}
	fmt.Println("count:", len(testDirs))
	fmt.Println("paths:", testDirs)
	fmt.Println()

	// find test results in each dir, skip if invalid test dir
	for _, testDir := range testDirs {
		// <root_tests_dir>/<test_dir>
		testDirPath := filepath.Join(testsRootDir, testDir.Name())
		// read unique test phase dirs
		fmt.Println("read test result dir:", testDirPath)
		testPhaseDirs, err := ioutil.ReadDir(testDirPath)
		if err != nil {
			return nil, err
		}
		fmt.Println("count:", len(testDirPath))
		fmt.Println("paths:", testDirPath)
		fmt.Println()

		// find step-info in dir, continue if no step-info.json as this file is only required if step has exported artifacts also
		// <root_tests_dir>/<test_dir>/step-info.json
		fmt.Println("readfile:", filepath.Join(testDirPath, "step-info.json"))
		var stepInfo *models.TestResultStepInfo
		stepInfoFileContent, err := fileutil.ReadBytesFromFile(filepath.Join(testDirPath, "step-info.json"))
		if err != nil {
			continue
		}
		fmt.Println("unmarshal:", string(stepInfoFileContent))
		if err := json.Unmarshal(stepInfoFileContent, &stepInfo); err != nil {
			continue
		}
		fmt.Println("done:", *stepInfo)

		for _, testPhaseDir := range testPhaseDirs {
			// <root_tests_dir>/<test_dir>/<unique_dir>
			testPhaseDirPath := filepath.Join(testDirPath, testPhaseDir.Name())
			fmt.Println("readdir:", testPhaseDirPath)
			// read one level of file set only <root_tests_dir>/<test_dir>/<unique_dir>/files_to_get
			testFiles, err := filepath.Glob(filepath.Join(testPhaseDirPath, "*"))
			if err != nil {
				return nil, err
			}
			fmt.Println("done:", testFiles)

			// get the converter that can manage test type contained in the dir
			for _, converter := range converters.List() {
				fmt.Println("running converter")
				// skip if couldn't find converter for content type
				if converter.Detect(testFiles) {
					fmt.Println("detected")
					// test-info.json file is required
					fmt.Println("reading file:", filepath.Join(testPhaseDirPath, "test-info.json"))
					testInfoFileContent, err := fileutil.ReadBytesFromFile(filepath.Join(testPhaseDirPath, "test-info.json"))
					if err != nil {
						return nil, err
					}
					fmt.Println("done")
					var testInfo struct {
						Name string `json:"test-name"`
					}
					fmt.Println("unmarshal:", string(testInfoFileContent))
					if err := json.Unmarshal(testInfoFileContent, &testInfo); err != nil {
						return nil, err
					}
					fmt.Println("done:", testInfo)

					fmt.Println("getting xml")
					junitXML, err := converter.XML()
					if err != nil {
						return nil, err
					}
					fmt.Println("done, marshalling:", junitXML)

					xmlData, err := xml.MarshalIndent(junitXML, "", " ")
					if err != nil {
						return nil, err
					}
					xmlData = append([]byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"), xmlData...)

					// so here I will have image paths, xml data, and step info per test dir in a bundle info
					results = append(results, Result{
						Name:       testInfo.Name,
						XMLContent: xmlData,
						ImagePaths: findImages(filepath.Join(testsRootDir, testPhaseDir.Name())),
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

		if err := httpCall("", http.MethodPut, uploadResponse.URL, bytes.NewReader(result.XMLContent), nil); err != nil {
			return err
		}

		for _, upload := range uploadResponse.Assets {
			for _, file := range result.ImagePaths {
				if filepath.Base(file) == upload.FileName {
					fi, err := os.Open(file)
					if err != nil {
						return err
					}
					if err := httpCall("", http.MethodPut, upload.URL, fi, nil); err != nil {
						return err
					}
					break
				}
			}
		}

		if err := httpCall(apiToken, http.MethodPatch, fmt.Sprintf("%s/apps/%s/builds/%s/test_reports/%s", endpointBaseURL, appSlug, buildSlug, uploadResponse.ID), strings.NewReader(`{"uploaded":true}`), nil); err != nil {
			return err
		}
	}

	return nil
}
