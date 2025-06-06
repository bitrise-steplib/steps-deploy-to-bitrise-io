package uploaders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"

	androidparser "github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/urlutil"
	iosparser "github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

type ArtifactURLs struct {
	PublicInstallPageURL string
	PermanentDownloadURL string
	DetailsPageURL       string
}

type AppDeploymentMetaData struct {
	AndroidArtifactInfo    *androidparser.ArtifactMetadata
	IOSArtifactInfo        *iosparser.ArtifactMetadata
	NotifyUserGroups       string
	AlwaysNotifyUserGroups string
	NotifyEmails           string
	IsEnablePublicPage     bool
}

type ArtifactArgs struct {
	Path     string
	FileSize int64 // bytes
}

type TransferDetails struct {
	Size     int64
	Duration time.Duration
	Hostname string
}

type UploadTask struct {
	ErrorMessage   string `json:"error_msg"`
	URL            string `json:"upload_url"`
	ID             int64  `json:"id"`
	IsIntermediate bool   `json:"is_intermediate_file"`
}

func (u UploadTask) Identifier() string {
	return fmt.Sprintf("%d", u.ID)
}

func createArtifact(buildURL, token string, artifact ArtifactArgs, artifactType, contentType string, archiveAsArtifact bool, pipelineMeta *deployment.IntermediateFileMetaData) ([]UploadTask, error) {
	// create form data
	artifactName := filepath.Base(artifact.Path)

	log.Printf("file size: %s", units.BytesSize(float64(artifact.FileSize)))

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

	var response *http.Response
	var uploadTasks []UploadTask

	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("%d attempt failed", attempt)
		}
		response, err = http.PostForm(uri, data)
		if err != nil {
			return fmt.Errorf("failed to perform create artifact request, error: %s", err)
		}

		defer func() {
			if err := response.Body.Close(); err != nil {
				log.Errorf("Failed to close reponse body, error: %s", err)
			}
		}()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read create artifact response, error: %s", err)
		}
		if response.StatusCode != http.StatusOK {
			type errorResponse struct {
				ErrorMessage string `json:"error_msg"`
			}
			var createResponse errorResponse
			if unmarshalErr := json.Unmarshal(body, &createResponse); unmarshalErr != nil {
				return errors.New(string(body))
			}

			return errors.New(createResponse.ErrorMessage)
		}

		if err := json.Unmarshal(body, &uploadTasks); err != nil {
			return fmt.Errorf("failed to unmarshal response (%s), error: %s", string(body), err)
		}

		if len(uploadTasks) == 0 {
			return fmt.Errorf("failed to create artifact on bitrise, error: no upload task received")
		}

		for _, task := range uploadTasks {
			if task.ErrorMessage != "" {
				return fmt.Errorf("failed to create artifact on bitrise, error message: %s", task.ErrorMessage)
			}

			if task.URL == "" {
				return fmt.Errorf("failed to create artifact on bitrise, error: missing upload url")
			}
			if task.ID == 0 {
				return fmt.Errorf("failed to create artifact on bitrise, error: missing artifact id")
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return uploadTasks, nil
}

func UploadArtifact(uploadURL string, artifact ArtifactArgs, contentType string) (TransferDetails, error) {
	netClient := &http.Client{
		Timeout: 10 * time.Minute,
	}

	start := time.Now()

	err := retry.Times(3).Wait(5).Try(func(attempt uint) error {
		file, err := os.Open(artifact.Path)
		if err != nil {
			return fmt.Errorf("failed to open artifact, error: %s", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Warnf("failed to close file, error: %s", err)
			}
		}()

		// Initializes request body to nil to send a Content-Length of 0: https://github.com/golang/go/issues/20257#issuecomment-299509391
		var reqBody io.Reader
		if artifact.FileSize > 0 {
			reqBody = io.NopCloser(file)
		}

		request, err := http.NewRequest(http.MethodPut, uploadURL, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request, error: %s", err)
		}

		if contentType != "" {
			request.Header.Add("Content-Type", contentType)
		}

		request.Header.Add("X-Upload-Content-Length", strconv.FormatInt(artifact.FileSize, 10)) // header used by Google Cloud Storage signed URLs
		request.ContentLength = artifact.FileSize

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		request = request.WithContext(ctx)

		resp, err := netClient.Do(request)
		if err != nil {
			return fmt.Errorf("failed to upload artifact, error: %s", err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("Failed to close response body, error: %s", err)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non success status code: %d, headers: %s, body: %s", resp.StatusCode, resp.Header, body)
		}

		return nil
	})

	details := TransferDetails{
		Size:     artifact.FileSize,
		Duration: time.Since(start),
		Hostname: extractHost(uploadURL),
	}

	return details, err
}

func finishArtifact(buildURL, token, artifactID string, appDeploymentMeta *AppDeploymentMetaData) (ArtifactURLs, error) {
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

	var response *http.Response

	type finishArtifactResponse struct {
		PublicInstallPageURL string   `json:"public_install_page_url"`
		PermanentDownloadURL string   `json:"permanent_download_url"`
		DetailsPageURL       string   `json:"details_page_url"`
		InvalidEmails        []string `json:"invalid_emails"`
	}

	var artifactResponse finishArtifactResponse
	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("%d attempt failed", attempt)
		}
		response, err = http.PostForm(uri, data)
		if err != nil {
			return fmt.Errorf("failed to perform finish artifact request, error: %s", err)
		}
		defer func() {
			if err := response.Body.Close(); err != nil {
				log.Errorf("Failed to close reponse body, error: %s", err)
			}
		}()

		// process response
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read finish artifact response, error: %s", err)
		}
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to create artifact on bitrise, status code: %d, response: %s", response.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, &artifactResponse); err != nil {
			return fmt.Errorf("failed to unmarshal response (%s), error: %s", string(body), err)
		}

		return nil
	}); err != nil {
		return ArtifactURLs{}, err
	}

	if len(artifactResponse.InvalidEmails) > 0 {
		log.Warnf("Invalid e-mail addresses: %s", strings.Join(artifactResponse.InvalidEmails, ", "))
	}

	return ArtifactURLs{
		PermanentDownloadURL: artifactResponse.PermanentDownloadURL,
		DetailsPageURL:       artifactResponse.DetailsPageURL,
		PublicInstallPageURL: artifactResponse.PublicInstallPageURL,
	}, nil
}

func printableAppInfo(appInfo interface{}) string {
	bytes, err := json.Marshal(appInfo)
	if err != nil {
		return fmt.Sprintf("failed to marshal app info: %+v, error: %s", appInfo, err)
	}

	return string(bytes)
}

func extractHost(downloadURL string) string {
	u, err := url.Parse(downloadURL)
	if err != nil {
		return "unknown"
	}

	return strings.TrimPrefix(u.Hostname(), "www.")
}
