package uploaders

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-units"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/urlutil"
	logV2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

type MultipartUploadTask struct {
	PartURLs       []string `json:"part_urls"`
	ID             int64    `json:"id"`
	PartSize       int64    `json:"part_size"`
	IsIntermediate bool     `json:"is_intermediate_file"`
}

func (u MultipartUploadTask) Identifier() string {
	return fmt.Sprintf("%d", u.ID)
}

// UploadedPart records the ETag returned by S3 after uploading a single part.
// Both fields are required by complete_multipart_upload on the server side.
type UploadedPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

func (u *Uploader) uploadMultipart(buildURL, token string, artifact ArtifactArgs, artifactType, contentType string, item *deployment.DeployableItem, buildArtifactMeta *AppDeploymentMetaData) ([]ArtifactURLs, error) {
	tasks, err := createMultipartArtifact(buildURL, token, artifact, artifactType, contentType, item.ArchiveAsArtifact, item.IntermediateFileMeta, u.logger)
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
		start := time.Now()
		parts, uploadErr := uploadAllParts(artifact.Path, artifact.FileSize, task.PartSize, task.PartURLs, u.multipartConcurrency, u.logger)

		transferType := Artifact
		if task.IsIntermediate {
			transferType = Intermediate
		}
		details := TransferDetails{
			Size:     artifact.FileSize,
			Duration: time.Since(start),
			Hostname: extractHost(task.PartURLs[0]),
			Multipart: &MultipartTransferDetails{
				PartCount:   len(task.PartURLs),
				PartSize:    task.PartSize,
				Concurrency: u.multipartConcurrency,
			},
		}
		u.tracker.logFileTransfer(transferType, details, uploadErr, item.ArchiveAsArtifact, item.IsIntermediateFile())

		if uploadErr != nil {
			if _, abortErr := finishMultipartArtifact(buildURL, token, task.Identifier(), false, nil, nil, u.logger); abortErr != nil {
				u.logger.Warnf("Failed to abort multipart upload for artifact %d: %s", task.ID, abortErr)
			}
			return nil, fmt.Errorf("failed to upload artifact parts (%s): %w", artifact.Path, uploadErr)
		}

		urls, err := finishMultipartArtifact(buildURL, token, task.Identifier(), true, parts, buildArtifactMeta, u.logger)
		if err != nil {
			return nil, fmt.Errorf("failed to finish artifact upload (%s): %w", artifact.Path, err)
		}

		if !task.IsIntermediate || useIntermediateFileURLs {
			artifactURLs = append(artifactURLs, urls)
		}
	}

	return artifactURLs, nil
}

func createMultipartArtifact(buildURL, token string, artifact ArtifactArgs, artifactType, contentType string, archiveAsArtifact bool, pipelineMeta *deployment.IntermediateFileMetaData, logger logV2.Logger) ([]MultipartUploadTask, error) {
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

	if pipelineMeta != nil {
		pipelineInfoBytes, err := json.Marshal(pipelineMeta)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal deployment meta: %s", err)
		}

		data["intermediate_file_info"] = []string{string(pipelineInfoBytes)}
	}

	uri, err := urlutil.Join(buildURL, "artifacts", "create_multipart_upload.json")
	if err != nil {
		return nil, fmt.Errorf("failed to generate create artifact url, error: %s", err)
	}

	var response *http.Response
	var uploadTasks []MultipartUploadTask

	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("%d attempt failed", attempt)
		}

		req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(data.Encode()))
		if err != nil {
			return fmt.Errorf("failed to create request: %s", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		if dump, dumpErr := httputil.DumpRequestOut(req, true); dumpErr == nil {
			logger.Debugf("create_multipart_upload request:\n%s", dump)
		}

		response, err = http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to perform create artifact request, error: %s", err)
		}

		defer func() {
			if err := response.Body.Close(); err != nil {
				log.Errorf("Failed to close reponse body, error: %s", err)
			}
		}()

		if dump, dumpErr := httputil.DumpResponse(response, true); dumpErr == nil {
			logger.Debugf("create_multipart_upload response:\n%s", dump)
		}

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

func finishMultipartArtifact(buildURL, token, artifactID string, success bool, parts []UploadedPart, appDeploymentMeta *AppDeploymentMetaData, logger logV2.Logger) (ArtifactURLs, error) {
	data := url.Values{
		"api_token": {token},
		"success":   {strconv.FormatBool(success)},
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

	uri, err := urlutil.Join(buildURL, "artifacts", artifactID, "finish_multipart_upload.json")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to generate finish artifact url, error: %s", err)
	}

	// url.Values.Encode() sorts keys alphabetically, which groups all parts[][etag]
	// entries before all parts[][part_number] entries. The nested parameter parser
	// creates a new array element each time it sees a key the tail already has, so
	// 3 etags + 3 part_numbers becomes 5 objects instead of 3. To work around this,
	// we need to encode the parts manually after encoding the other parameters.
	partNumKey := url.QueryEscape("parts[][part_number]")
	partEtagKey := url.QueryEscape("parts[][etag]")
	var partsEncoded strings.Builder
	for _, part := range parts {
		partsEncoded.WriteByte('&')
		partsEncoded.WriteString(partNumKey)
		partsEncoded.WriteByte('=')
		partsEncoded.WriteString(strconv.Itoa(part.PartNumber))
		partsEncoded.WriteByte('&')
		partsEncoded.WriteString(partEtagKey)
		partsEncoded.WriteByte('=')
		partsEncoded.WriteString(url.QueryEscape(part.ETag))
	}
	encodedBody := data.Encode() + partsEncoded.String()

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

		req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(encodedBody))
		if err != nil {
			return fmt.Errorf("failed to create request: %s", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		if dump, dumpErr := httputil.DumpRequestOut(req, true); dumpErr == nil {
			logger.Debugf("finish_multipart_upload request:\n%s", dump)
		}

		response, err = http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to perform finish artifact request, error: %s", err)
		}
		defer func() {
			if err := response.Body.Close(); err != nil {
				log.Errorf("Failed to close reponse body, error: %s", err)
			}
		}()

		if dump, dumpErr := httputil.DumpResponse(response, true); dumpErr == nil {
			logger.Debugf("finish_multipart_upload response:\n%s", dump)
		}

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
func uploadPart(partURL, filePath string, offset, size int64, partNumber int, logger logV2.Logger) (string, error) {
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

		logger.Debugf("part %d upload: PUT %s (offset=%d size=%d)", partNumber, partURL, offset, size)

		resp, err := netClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to upload part %d: %s", partNumber, err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("Failed to close response body, error: %s", err)
			}
		}()

		if dump, dumpErr := httputil.DumpResponse(resp, true); dumpErr == nil {
			logger.Debugf("part %d response:\n%s", partNumber, dump)
		}

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

// uploadAllParts uploads the file in partSize-byte chunks, one per partURL, up to
// concurrency parts at a time. The last part is trimmed to the file's
// remaining bytes. Returns the ETags required by the finish call.
func uploadAllParts(filePath string, fileSize int64, partSize int64, partURLs []string, concurrency int, logger logV2.Logger) ([]UploadedPart, error) {
	if partSize <= 0 {
		return nil, fmt.Errorf("invalid part size %d", partSize)
	}
	expectedPartCount := (fileSize + partSize - 1) / partSize
	if int64(len(partURLs)) != expectedPartCount {
		return nil, fmt.Errorf("part_urls count (%d) does not match expected (%d) for file size %d and part size %d", len(partURLs), expectedPartCount, fileSize, partSize)
	}

	type partResult struct {
		partNumber int
		etag       string
		err        error
	}

	results := make(chan partResult, len(partURLs))
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup
	for i, partURL := range partURLs {
		partNumber := i + 1
		url := partURL
		offset := int64(i) * partSize
		size := partSize
		if offset+size > fileSize {
			size = fileSize - offset
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			etag, err := uploadPart(url, filePath, offset, size, partNumber, logger)
			results <- partResult{partNumber: partNumber, etag: etag, err: err}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	parts := make([]UploadedPart, 0, len(partURLs))
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
