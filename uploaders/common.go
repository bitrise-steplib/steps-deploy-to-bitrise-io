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
	"sync"
	"time"

	"github.com/docker/go-units"

	androidparser "github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/urlutil"
	iosparser "github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

const multipartConcurrentParts = 4

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
	PartURLs       []string `json:"part_urls"`
	ID             int64    `json:"id"`
	IsIntermediate bool     `json:"is_intermediate_file"`
}

func (u UploadTask) Identifier() string {
	return fmt.Sprintf("%d", u.ID)
}

// UploadedPart records the ETag returned by S3 after uploading a single part.
// Both fields are required by complete_multipart_upload on the server side.
type UploadedPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

func createArtifact(buildURL, token string, artifact ArtifactArgs, artifactType, contentType string, archiveAsArtifact bool, pipelineMeta *deployment.IntermediateFileMetaData) ([]UploadTask, error) {
	// create form data
	artifactName := filepath.Base(artifact.Path)

	log.Printf("file size: %s", units.BytesSize(float64(artifact.FileSize)))

	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("provided API token is empty")
	}

	isIntermediateFile := pipelineMeta != nil

	data := url.Values{
		"api_token":                    {token},
		"title":                        {artifactName},
		"filename":                     {artifactName},
		"artifact_type":                {artifactType},
		"file_size_bytes":              {fmt.Sprintf("%d", artifact.FileSize)},
		"content_type":                 {contentType},
		"archive_as_artifact":          {strconv.FormatBool(archiveAsArtifact)},
		"archive_as_intermediate_file": {strconv.FormatBool(isIntermediateFile)},
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
	uri, err := urlutil.Join(buildURL, "artifacts", "create_multipart_upload.json")
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
			if len(task.PartURLs) == 0 {
				return fmt.Errorf("failed to create artifact on bitrise, error: missing part urls")
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

func finishArtifact(buildURL, token, artifactID string, success bool, parts []UploadedPart, appDeploymentMeta *AppDeploymentMetaData) (ArtifactURLs, error) {
	// create form data
	data := url.Values{
		"api_token": {token},
		"success":   {strconv.FormatBool(success)},
	}
	for _, part := range parts {
		data.Add("parts[][part_number]", strconv.Itoa(part.PartNumber))
		data.Add("parts[][etag]", part.ETag)
	}
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
	uri, err := urlutil.Join(buildURL, "artifacts", artifactID, "finish_multipart_upload.json")
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

// uploadPart uploads a single chunk of a file to a presigned S3 part URL.
// It opens the file independently to allow concurrent calls without seeking conflicts.
// Returns the ETag header value which must be included in the finish call.
func uploadPart(partURL, filePath string, offset, size int64, partNumber int) (string, error) {
	netClient := &http.Client{Timeout: 10 * time.Minute}

	var etag string

	err := retry.Times(3).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("part %d: attempt %d failed, retrying", partNumber, attempt)
		}

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %s", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Warnf("failed to close file: %s", err)
			}
		}()

		// SectionReader provides a bounded, independently seekable view into the file,
		// safe for concurrent use since each goroutine opens its own file descriptor.
		section := io.NewSectionReader(file, offset, size)

		req, err := http.NewRequest(http.MethodPut, partURL, section)
		if err != nil {
			return fmt.Errorf("failed to create request: %s", err)
		}
		req.ContentLength = size

		resp, err := netClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to upload part %d: %s", partNumber, err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("Failed to close response body, error: %s", err)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %s", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("part %d: non-200 status %d: %s", partNumber, resp.StatusCode, body)
		}

		etag = resp.Header.Get("ETag")
		if etag == "" {
			return fmt.Errorf("part %d: response missing ETag header", partNumber)
		}

		return nil
	})

	return etag, err
}

// uploadAllParts splits the file into len(partURLs) chunks and uploads them concurrently,
// up to multipartConcurrentParts at a time. Returns the ETags required by the finish call.
func uploadAllParts(filePath string, fileSize int64, partURLs []string) ([]UploadedPart, error) {
	partCount := len(partURLs)
	chunkSize := (fileSize + int64(partCount) - 1) / int64(partCount)

	type partResult struct {
		partNumber int
		etag       string
		err        error
	}

	results := make(chan partResult, partCount)
	sem := make(chan struct{}, multipartConcurrentParts)

	var wg sync.WaitGroup
	for i, partURL := range partURLs {
		partNumber := i + 1
		url := partURL
		offset := int64(i) * chunkSize
		size := chunkSize
		if offset+size > fileSize {
			size = fileSize - offset
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			etag, err := uploadPart(url, filePath, offset, size, partNumber)
			results <- partResult{partNumber: partNumber, etag: etag, err: err}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	parts := make([]UploadedPart, 0, partCount)
	var errs []string
	for result := range results {
		if result.err != nil {
			errs = append(errs, fmt.Sprintf("part %d: %s", result.partNumber, result.err))
		} else {
			parts = append(parts, UploadedPart{PartNumber: result.partNumber, ETag: result.etag})
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("part upload failures: %s", strings.Join(errs, "; "))
	}

	return parts, nil
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
