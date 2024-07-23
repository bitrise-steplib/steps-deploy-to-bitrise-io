package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/report/api/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	authToken = "auth-token"
	buildURL  = "build-url"
)

func TestCreateReport(t *testing.T) {
	tests := []struct {
		name               string
		params             CreateReportParameters
		responseStatusCode int
		responseBody       string
		wantError          bool
		expectedError      error
		expectedOutput     CreateReportResponse
	}{
		{
			name: "Successful request",
			params: CreateReportParameters{
				Title: "title",
				Assets: []CreateReportAsset{
					{
						RelativePath: "/path/to/somewhere",
						FileSize:     10,
						ContentType:  "text",
					},
				},
			},
			responseStatusCode: 200,
			responseBody: `{
"id": "some-id",
"assets": [
	{
		"relative_path": "/path/to/somewhere",
		"upload_url": "http://test.test"
	}]
}`,
			wantError: false,
			expectedOutput: CreateReportResponse{
				Identifier: "some-id",
				AssetURLs: []CreateReportURL{
					{
						RelativePath: "/path/to/somewhere",
						URL:          "http://test.test",
					},
				},
			},
		},
		{
			name: "Handle failure",
			params: CreateReportParameters{
				Title: "another-title",
				Assets: []CreateReportAsset{
					{
						RelativePath: "/path/to/somewhere",
						FileSize:     3,
						ContentType:  "png",
					},
				},
			},
			responseStatusCode: 301,
			responseBody:       "{\"error_msg\": \"There was an error\"}",
			wantError:          true,
			expectedError:      fmt.Errorf("request to %s/html_reports.json failed: status code should be 2xx (301): There was an error", buildURL),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiClient, mockHTTPClient := createSutAndMock(t)

			var request http.Request
			setupMockNetworking(t, mockHTTPClient, &request, tt.responseBody, tt.responseStatusCode)

			response, err := apiClient.CreateReport(tt.params)
			assert.Equal(t, fmt.Sprintf("%s/html_reports.json", buildURL), request.URL.String())
			assert.Equal(t, []string{authToken}, request.Header["BUILD_API_TOKEN"]) //nolint:staticcheck // See TestReportClient.perform()

			if tt.wantError {
				assert.Equal(t, tt.expectedError, err)
			} else {
				var received CreateReportParameters
				err = json.NewDecoder(request.Body).Decode(&received)
				assert.NoError(t, err)
				assert.Equal(t, tt.params, received)

				assert.Equal(t, tt.expectedOutput, response)
			}

			mockHTTPClient.AssertExpectations(t)
		})
	}
}

func TestFinishReport(t *testing.T) {
	tests := []struct {
		name               string
		identifier         string
		allAssetsUploaded  bool
		responseStatusCode int
		responseBody       string
		wantError          bool
		expectedError      error
	}{
		{
			name:               "Successful request",
			identifier:         "report-id",
			allAssetsUploaded:  true,
			responseStatusCode: 200,
			responseBody:       "",
			wantError:          false,
		},
		{
			name:               "Handle failure",
			identifier:         "report-id",
			allAssetsUploaded:  false,
			responseStatusCode: 301,
			responseBody:       "{\"error_msg\": \"There was an error\"}",
			wantError:          true,
			expectedError:      fmt.Errorf("request to %s/html_reports/report-id.json failed: status code should be 2xx (301): There was an error", buildURL),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiClient, mockHTTPClient := createSutAndMock(t)

			var request http.Request
			setupMockNetworking(t, mockHTTPClient, &request, tt.responseBody, tt.responseStatusCode)

			err := apiClient.FinishReport(tt.identifier, tt.allAssetsUploaded)
			assert.Equal(t, fmt.Sprintf("%s/html_reports/%s.json", buildURL, tt.identifier), request.URL.String())
			assert.Equal(t, []string{authToken}, request.Header["BUILD_API_TOKEN"]) //nolint:staticcheck // See TestReportClient.perform()

			if tt.wantError {
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}

			mockHTTPClient.AssertExpectations(t)
		})
	}
}

func createSutAndMock(t *testing.T) (TestReportClient, *mocks.HttpClient) {
	mockHTTPClient := mocks.NewHttpClient(t)
	client := TestReportClient{
		logger:     log.NewLogger(),
		httpClient: mockHTTPClient,
		authToken:  authToken,
		buildURL:   buildURL,
	}

	return client, mockHTTPClient
}

func setupMockNetworking(t *testing.T, mockHTTPClient *mocks.HttpClient, request *http.Request, body string, statusCode int) {
	response := &http.Response{Body: io.NopCloser(bytes.NewReader([]byte(body)))}
	response.StatusCode = statusCode

	mockHTTPClient.On("Do", mock.Anything).Return(response, nil).Run(func(args mock.Arguments) {
		if request == nil {
			return
		}

		value, ok := args.Get(0).(*http.Request)
		if !ok {
			require.Fail(t, "Failed to cast to http.Request")
		}

		*request = *value
		response.Request = value
	})
}
