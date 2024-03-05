package report

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	loggerV2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/report/api"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/report/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Smallest valid base64 encoded image data which returns a correct content type.
const (
	pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z/C/HgAGgwJ/lK3Q6wAAAABJRU5ErkJggg=="
	jpgBase64 = "/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////wgALCAABAAEBAREA/8QAFBABAAAAAAAAAAAAAAAAAAAAAP/aAAgBAQABPxA="
)

func TestFindsAndUploadsReports(t *testing.T) {
	reportDir, reports := createReports(t)

	mockClient := mocks.NewClientAPI(t)
	setupMockingForReport(mockClient, reports[0])
	setupMockingForReport(mockClient, reports[2])
	setupMockingForReport(mockClient, reports[3])

	uploader := HTMLReportUploader{
		client:      mockClient,
		logger:      loggerV2.NewLogger(),
		reportDir:   reportDir,
		concurrency: 1,
	}

	uploadErrors := uploader.DeployReports()
	require.Equal(t, 0, len(uploadErrors))

	mockClient.AssertExpectations(t)
}

func TestInvalidReportFiltering(t *testing.T) {
	reportDir, reports := createReports(t)
	uploader := HTMLReportUploader{
		client:      nil,
		logger:      loggerV2.NewLogger(),
		reportDir:   reportDir,
		concurrency: 1,
	}

	// Create an invalid report
	reports[0].Assets[2].Path = filepath.Join(filepath.Dir(reports[0].Assets[2].Path), "my_report_index.html")
	reports[0].Assets[2].TestDirRelativePath = filepath.Join(filepath.Dir(reports[0].Assets[2].TestDirRelativePath), "my_report_index.html")

	validatedReports, validatedErrors := uploader.validate(reports)

	expectedErrors := []error{
		fmt.Errorf("missing index.html file for Test A"),
		fmt.Errorf("missing index.html file for Test B"),
	}
	assert.Equal(t, expectedErrors, validatedErrors)
	assert.Equal(t, len(validatedReports), 2)
}

// createReports creates dummy data for multiple test reports.
func createReports(t *testing.T) (string, []Report) {
	tempDir := t.TempDir()
	reportData := []map[string]interface{}{
		{
			"name": "Test A",
			"assets": []map[string]string{
				{
					"name": "a.png",
					"size": "70",
					"type": "image/png",
				},
				{
					"name": "b.jpg",
					"size": "134",
					"type": "image/jpeg",
				},
				{
					"name": "index.html",
					"size": "1",
					"type": "text/html; charset=utf-8",
				},
			},
		},
		{
			"name":   "Test B",
			"assets": []map[string]string{},
		},
		{
			"name": "Test C",
			"assets": []map[string]string{
				{
					"name": "index.html",
					"size": "1",
					"type": "text/html; charset=utf-8",
				},
			},
		},
		{
			"name": "Test D",
			"info": map[string]string{
				"category": "random",
			},
			"assets": []map[string]string{
				{
					"name": "index.html",
					"size": "1",
					"type": "text/html; charset=utf-8",
				},
			},
		},
	}

	var reports []Report
	for _, data := range reportData {
		reportName, ok := data["name"].(string)
		if !ok {
			require.Fail(t, "Failed to cast the report name to string type")
		}

		reportPath := filepath.Join(tempDir, reportName)
		require.NoError(t, os.Mkdir(reportPath, 0755))

		assetDirPath := filepath.Join(reportPath, "something.xcresult")
		require.NoError(t, os.Mkdir(assetDirPath, 0755))

		assetData, ok := data["assets"].([]map[string]string)
		if !ok {
			require.Fail(t, "Failed to cast the asset data to map type")
		}

		var assets []Asset
		for _, asset := range assetData {
			assetName := asset["name"]

			var bytes []byte
			if strings.HasSuffix(assetName, ".png") {
				decoded, err := base64.StdEncoding.DecodeString(pngBase64)
				require.NoError(t, err)

				bytes = decoded
			} else if strings.HasSuffix(assetName, ".jpg") {
				decoded, err := base64.StdEncoding.DecodeString(jpgBase64)
				require.NoError(t, err)

				bytes = decoded
			} else {
				bytes = []byte("a")
			}

			assetPath := filepath.Join(assetDirPath, assetName)
			require.NoError(t, os.WriteFile(assetPath, bytes, 0755))

			relativePath, err := filepath.Rel(reportPath, assetPath)
			require.NoError(t, err)

			size, err := strconv.Atoi(asset["size"])
			require.NoError(t, err)

			assets = append(assets, Asset{
				Path:                assetPath,
				TestDirRelativePath: relativePath,
				FileSize:            int64(size),
				ContentType:         asset["type"],
			})
		}

		var reportInfo Info
		if reportInfoRaw, ok := data["info"].(map[string]string); ok {
			reportInfoData, err := json.Marshal(reportInfoRaw)
			require.NoError(t, err)

			reportInfoFile := filepath.Join(reportPath, htmlReportInfoFile)
			require.NoError(t, os.WriteFile(reportInfoFile, reportInfoData, 0755))

			require.NoError(t, json.Unmarshal(reportInfoData, &reportInfo))
		}

		reports = append(reports, Report{
			Name:   reportName,
			Info:   reportInfo,
			Assets: assets,
		})
	}

	return tempDir, reports
}

// setupMockingForReport sets up the mock to expect the given report.
func setupMockingForReport(client *mocks.ClientAPI, report Report) {
	var requestAssets []api.CreateReportAsset
	var responseURLs []api.CreateReportURL
	for i, asset := range report.Assets {
		requestAssets = append(requestAssets, api.CreateReportAsset{
			RelativePath: asset.TestDirRelativePath,
			FileSize:     asset.FileSize,
			ContentType:  asset.ContentType,
		})
		responseURLs = append(responseURLs, api.CreateReportURL{
			RelativePath: asset.TestDirRelativePath,
			URL:          fmt.Sprintf("%s-%d", asset.TestDirRelativePath, i),
		})
	}
	requestParams := api.CreateReportParameters{
		Title:    report.Name,
		Category: report.Info.Category,
		Assets:   requestAssets,
	}

	response := api.CreateReportResponse{
		Identifier: fmt.Sprintf("%s-identifier", report.Name),
		AssetURLs:  responseURLs,
	}

	client.On("CreateReport", requestParams).Return(response, nil)

	for i, responseURL := range responseURLs {
		asset := report.Assets[i]
		client.On("UploadAsset", responseURL.URL, asset.Path, asset.ContentType).Return(nil)
	}

	client.On("FinishReport", response.Identifier, true).Return(nil)
}
