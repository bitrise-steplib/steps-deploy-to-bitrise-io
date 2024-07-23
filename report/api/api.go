package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/uploaders"
)

// ClientAPI ...
type ClientAPI interface {
	CreateReport(params CreateReportParameters) (CreateReportResponse, error)
	UploadAsset(url, path, contentType string) error
	FinishReport(identifier string, allAssetsUploaded bool) error
}

// HTTPClient ...
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// TestReportClient ...
type TestReportClient struct {
	logger     log.Logger
	httpClient HTTPClient
	buildURL   string
	authToken  string
}

// NewBitriseClient ...
func NewBitriseClient(buildURL, authToken string, logger log.Logger) *TestReportClient {
	httpClient := retry.NewHTTPClient().StandardClient()

	return &TestReportClient{
		logger:     logger,
		httpClient: httpClient,
		buildURL:   buildURL,
		authToken:  authToken,
	}
}

// CreateReport ...
func (t *TestReportClient) CreateReport(params CreateReportParameters) (CreateReportResponse, error) {
	url := fmt.Sprintf("%s/html_reports.json", t.buildURL)

	body, err := json.Marshal(params)
	if err != nil {
		return CreateReportResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return CreateReportResponse{}, err
	}

	resp, err := t.perform(req)
	if err != nil {
		return CreateReportResponse{}, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CreateReportResponse{}, err
	}

	var response CreateReportResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return CreateReportResponse{}, err
	}

	return response, nil
}

// UploadAsset ...
func (t *TestReportClient) UploadAsset(url, path, contentType string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	artifact := uploaders.ArtifactArgs {
		Path: path,
		FileSize: fileInfo.Size(),
	}
	return uploaders.UploadArtifact(url, artifact, contentType)
}

// FinishReport ...
func (t *TestReportClient) FinishReport(identifier string, allAssetsUploaded bool) error {
	url := fmt.Sprintf("%s/html_reports/%s.json", t.buildURL, identifier)

	type parameters struct {
		Uploaded bool `json:"is_uploaded"`
	}
	params := parameters{Uploaded: allAssetsUploaded}

	body, err := json.Marshal(params)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	_, err = t.perform(req)
	if err != nil {
		return err
	}

	return nil
}

func (t *TestReportClient) perform(request *http.Request) (*http.Response, error) {
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	// Header.Set canonizes the keys, so we need to set the token this way.
	request.Header["BUILD_API_TOKEN"] = []string{t.authToken}

	dump, err := httputil.DumpRequest(request, false)
	if err != nil {
		t.logger.Warnf("Request dump failed: %w", err)
	} else {
		t.logger.Debugf("Request dump: %s", string(dump))
	}

	resp, err := t.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.logger.Warnf("Failed to close response body: %w", err)
		}
	}()

	dump, err = httputil.DumpResponse(resp, true)
	if err != nil {
		t.logger.Warnf("Response dump failed: %s", err)
	} else {
		t.logger.Debugf("Response dump: %s", string(dump))
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		message, err := parseErrorMessage(resp)
		if err != nil {
			t.logger.Warnf("Failed to parse error message from the response: %s", err)
		}

		return nil, fmt.Errorf("request to %s failed: status code should be 2xx (%d): %s", resp.Request.URL, resp.StatusCode, message)
	}

	return resp, nil
}

func parseErrorMessage(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type errorResponse struct {
		Message string `json:"error_msg"`
	}

	var response errorResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	return response.Message, nil
}
