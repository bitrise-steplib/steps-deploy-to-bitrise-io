package uploaders

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/bitrise-io/go-utils/urlutil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

type UploadTask struct {
	ErrorMessage   string `json:"error_msg"`
	URL            string `json:"upload_url"`
	ID             int64  `json:"id"`
	IsIntermediate bool   `json:"is_intermediate_file"`
}

func (u UploadTask) Identifier() string {
	return fmt.Sprintf("%d", u.ID)
}

func (u *Uploader) uploadSingle(buildURL, token string, artifact ArtifactArgs, artifactType, contentType string, item *deployment.DeployableItem, buildArtifactMeta *AppDeploymentMetaData) ([]ArtifactURLs, error) {
	tasks, err := createArtifact(buildURL, token, artifact, artifactType, contentType, item.ArchiveAsArtifact, item.IntermediateFileMeta, u.httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create artifact (%s): %w", artifact.Path, err)
	}

	useIntermediateFileURLs := true
	if item.ArchiveAsArtifact && item.IntermediateFileMeta != nil && len(tasks) > 1 {
		// When the item is both a Build Artifact and an Intermediate File,
		// only surface the artifact URLs from the Build Artifact task.
		useIntermediateFileURLs = false
	}

	var artifactURLs []ArtifactURLs
	for _, task := range tasks {
		details, uploadErr := UploadArtifact(task.URL, artifact, contentType, u.httpClient)

		transferType := Artifact
		if task.IsIntermediate {
			transferType = Intermediate
		}
		u.tracker.logFileTransfer(transferType, details, uploadErr, item.ArchiveAsArtifact, item.IsIntermediateFile())

		if uploadErr != nil {
			return nil, fmt.Errorf("failed to upload artifact (%s): %w", artifact.Path, uploadErr)
		}

		urls, err := finishArtifact(buildURL, token, task.Identifier(), buildArtifactMeta, u.httpClient)
		if err != nil {
			return nil, fmt.Errorf("failed to finish artifact upload (%s): %w", artifact.Path, err)
		}

		if !task.IsIntermediate || useIntermediateFileURLs {
			artifactURLs = append(artifactURLs, urls)
		}
	}

	return artifactURLs, nil
}

func createArtifact(buildURL, token string, artifact ArtifactArgs, artifactType, contentType string, archiveAsArtifact bool, pipelineMeta *deployment.IntermediateFileMetaData, httpClient *HTTPClient) ([]UploadTask, error) {
	// create form data
	artifactName := filepath.Base(artifact.Path)

	httpClient.logger.Printf("file size: %s", units.BytesSize(float64(artifact.FileSize)))

	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("provided API token is empty")
	}

	data := url.Values{
		"api_token":           {token},
		"title":               {artifactName},
		"filename":            {artifactName},
		"artifact_type":       {artifactType},
		"file_size_bytes":     {fmt.Sprintf("%d", artifact.FileSize)},
		"content_type":        {contentType},
		"archive_as_artifact": {strconv.FormatBool(archiveAsArtifact)},
	}
	// ---

	if pipelineMeta != nil {
		pipelineInfoBytes, err := json.Marshal(pipelineMeta)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal deployment meta: %s", err)
		}

		data["intermediate_file_info"] = []string{string(pipelineInfoBytes)}
	}

	// perform request
	uri, err := urlutil.Join(buildURL, "artifacts.json")
	if err != nil {
		return nil, fmt.Errorf("failed to generate create artifact url, error: %s", err)
	}

	body, err := httpClient.doFormRequest(http.MethodPost, uri, data)
	if err != nil {
		type errorResponse struct {
			ErrorMessage string `json:"error_msg"`
		}
		var createResponse errorResponse
		if unmarshalErr := json.Unmarshal(body, &createResponse); unmarshalErr == nil && createResponse.ErrorMessage != "" {
			return nil, errors.New(createResponse.ErrorMessage)
		}
		return nil, err
	}

	var uploadTasks []UploadTask
	if err := json.Unmarshal(body, &uploadTasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response (%s), error: %s", string(body), err)
	}

	if len(uploadTasks) == 0 {
		return nil, fmt.Errorf("failed to create artifact on bitrise, error: no upload task received")
	}

	for _, task := range uploadTasks {
		if task.ErrorMessage != "" {
			return nil, fmt.Errorf("failed to create artifact on bitrise, error message: %s", task.ErrorMessage)
		}

		if task.URL == "" {
			return nil, fmt.Errorf("failed to create artifact on bitrise, error: missing upload url")
		}
		if task.ID == 0 {
			return nil, fmt.Errorf("failed to create artifact on bitrise, error: missing artifact id")
		}
	}

	return uploadTasks, nil
}

func UploadArtifact(uploadURL string, artifact ArtifactArgs, contentType string, httpClient *HTTPClient) (TransferDetails, error) {
	start := time.Now()

	// Initializes request body to nil to send a Content-Length of 0: https://github.com/golang/go/issues/20257#issuecomment-299509391
	var rawBody any
	if artifact.FileSize > 0 {
		file, err := os.Open(artifact.Path)
		if err != nil {
			return TransferDetails{}, fmt.Errorf("failed to open artifact, error: %s", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				httpClient.logger.Warnf("failed to close file, error: %s", err)
			}
		}()
		rawBody = file
	}

	req, err := retryablehttp.NewRequest(http.MethodPut, uploadURL, rawBody)
	if err != nil {
		return TransferDetails{}, fmt.Errorf("failed to create request, error: %s", err)
	}

	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}
	req.Header.Add("X-Upload-Content-Length", strconv.FormatInt(artifact.FileSize, 10)) // header used by Google Cloud Storage signed URLs
	req.ContentLength = artifact.FileSize

	_, err = httpClient.doRequest(req)

	details := TransferDetails{
		Size:     artifact.FileSize,
		Duration: time.Since(start),
		Hostname: extractHost(uploadURL),
	}

	return details, err
}

func finishArtifact(buildURL, token, artifactID string, appDeploymentMeta *AppDeploymentMetaData, httpClient *HTTPClient) (ArtifactURLs, error) {
	// create form data
	data := url.Values{"api_token": {token}}
	if appDeploymentMeta != nil {
		var artifactInfoBytes []byte
		var err error
		if appDeploymentMeta.IOSArtifactInfo != nil {
			artifactInfoBytes, err = json.Marshal(appDeploymentMeta.IOSArtifactInfo)
		} else if appDeploymentMeta.AndroidArtifactInfo != nil {
			artifactInfoBytes, err = json.Marshal(appDeploymentMeta.AndroidArtifactInfo)
		} else {
			err = fmt.Errorf("artifact metadata is missing")
		}
		if err != nil {
			return ArtifactURLs{}, fmt.Errorf("failed to marshal app deployment meta: %s", err)
		}
		artifactInfo := string(artifactInfoBytes)

		if artifactInfo != "" {
			data["artifact_info"] = []string{artifactInfo}
		}
		if appDeploymentMeta.NotifyUserGroups != "" {
			data["notify_user_groups"] = []string{appDeploymentMeta.NotifyUserGroups}
		}
		if appDeploymentMeta.AlwaysNotifyUserGroups != "" {
			data["always_notify_user_groups"] = []string{appDeploymentMeta.AlwaysNotifyUserGroups}
		}
		if appDeploymentMeta.NotifyEmails != "" {
			data["notify_emails"] = []string{appDeploymentMeta.NotifyEmails}
		}
		if appDeploymentMeta.IsEnablePublicPage {
			data["is_enable_public_page"] = []string{"yes"}
		}
	}

	// ---

	// perform request
	uri, err := urlutil.Join(buildURL, "artifacts", artifactID, "finish_upload.json")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to generate finish artifact url, error: %s", err)
	}

	type finishArtifactResponse struct {
		PublicInstallPageURL string   `json:"public_install_page_url"`
		PermanentDownloadURL string   `json:"permanent_download_url"`
		DetailsPageURL       string   `json:"details_page_url"`
		InvalidEmails        []string `json:"invalid_emails"`
	}

	body, err := httpClient.doFormRequest(http.MethodPost, uri, data)
	if err != nil {
		return ArtifactURLs{}, err
	}

	var artifactResponse finishArtifactResponse
	if err := json.Unmarshal(body, &artifactResponse); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to unmarshal response (%s), error: %s", string(body), err)
	}

	if len(artifactResponse.InvalidEmails) > 0 {
		httpClient.logger.Warnf("Invalid e-mail addresses: %s", strings.Join(artifactResponse.InvalidEmails, ", "))
	}

	return ArtifactURLs{
		PermanentDownloadURL: artifactResponse.PermanentDownloadURL,
		DetailsPageURL:       artifactResponse.DetailsPageURL,
		PublicInstallPageURL: artifactResponse.PublicInstallPageURL,
	}, nil
}
