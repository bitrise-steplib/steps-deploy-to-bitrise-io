package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/steps-deploy-to-bitrise-io/test/converters"
	"github.com/google/uuid"
)

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

// ResultAssets ...
type ResultAssets struct {
	XMLContent []byte                    `json:"xml"`
	ImagePaths []string                  `json:"-"`
	StepInfo   models.TestResultStepInfo `json:"step_info"`
}

// Bundle ...
type Bundle []ResultAssets

// ParseTestResults ...
func ParseTestResults(testsRootDir string) (bundle Bundle, err error) {
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

		// get the handler that can manage test type contained in the dir
		for _, handler := range converters.List() {
			handler.SetFiles(testFiles)
			// skip if couldn't find handler for content type
			if handler.Detect() {
				xml, err := handler.XML()
				if err != nil {
					return nil, err
				}

				// so here I will have image paths, xml data, and step info per test dir in a bundle info
				bundle = append(bundle, ResultAssets{
					XMLContent: xml,
					ImagePaths: findImages(filepath.Join(testsRootDir, testDir.Name())),
					StepInfo:   *stepInfo,
				})
			}
		}
	}
	return bundle, nil
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode < 200 || 299 < resp.StatusCode {
		bodyData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unsuccessful response code: %d and failed to read body, error: %s", resp.StatusCode, err)
		}
		return fmt.Errorf("unsuccessful response code: %d\nbody:\n%s", resp.StatusCode, bodyData)
	}
	return nil
}

// Upload ...
func (bundle Bundle) Upload(client *http.Client, endpointURL string) error {
	// generate UUID->filename and UUID->filepath maps and fill them with the assets need to be uploaded
	assetUploadRequestMap, assetIDPathMap := map[string]string{}, map[string]string{}
	for _, testResultAsset := range bundle {
		for _, path := range testResultAsset.ImagePaths {
			assetUUID := uuid.New().String()
			assetUploadRequestMap[assetUUID] = filepath.Base(path)
			assetIDPathMap[assetUUID] = path
		}
	}

	// Get an uploadURL map from the server UUID->URL to upload
	var assetUploadRequestMapData bytes.Buffer
	if err := json.NewEncoder(&assetUploadRequestMapData).Encode(assetUploadRequestMap); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, endpointURL, bytes.NewReader(assetUploadRequestMapData.Bytes()))
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close body, error: %s\n", err)
		}
	}()

	if err := checkResponse(resp); err != nil {
		return err
	}

	var assetUploadURLMap map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&assetUploadURLMap); err != nil {
		return err
	}

	// upload files matching to heir UUIDs
	for uuid, uploadURL := range assetUploadURLMap {
		assetPath, ok := assetIDPathMap[uuid]
		if !ok {
			return fmt.Errorf("unknown UUID(%s) for asset", uuid)
		}

		assetFile, err := os.Open(assetPath)
		if err != nil {
			return err
		}

		req, err := http.NewRequest(http.MethodPut, uploadURL, assetFile)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("Failed to close body, error: %s\n", err)
			}
		}()

		if err := checkResponse(resp); err != nil {
			return err
		}

	}

	// post a confirmation the assets are uploaded and send the xml and step info with it as well
	var bundleData bytes.Buffer
	if err := json.NewEncoder(&bundleData).Encode(bundle); err != nil {
		return err
	}

	fmt.Println("ITT", string(bundleData.String()))

	req, err = http.NewRequest(http.MethodPost, endpointURL, bytes.NewReader(bundleData.Bytes()))
	if err != nil {
		return err
	}

	resp, err = client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close body, error: %s\n", err)
		}
	}()

	return checkResponse(resp)
}
