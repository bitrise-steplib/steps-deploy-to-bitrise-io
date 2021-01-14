package uploaders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/urlutil"
)

// ArtifactURLs ...
type ArtifactURLs struct {
	PublicInstallPageURL string
	PermanentDownloadURL string
}

func createArtifact(buildURL, token, artifactPth, artifactType string) (string, string, error) {
	log.Printf("creating artifact")

	// create form data
	artifactName := filepath.Base(artifactPth)
	fileSize, err := fileSizeInBytes(artifactPth)
	if err != nil {
		return "", "", fmt.Errorf("failed to get file size, error: %s", err)
	}

	megaBytes := fileSize / 1024.0 / 1024.0
	roundedMegaBytes := int(roundPlus(megaBytes, 2))

	if roundedMegaBytes < 1 {
		log.Printf("  file size: %dB", int(fileSize))
	} else {
		log.Printf("  file size: %dMB", roundedMegaBytes)
	}

	data := url.Values{
		"api_token":       {token},
		"title":           {artifactName},
		"filename":        {artifactName},
		"artifact_type":   {artifactType},
		"file_size_bytes": {fmt.Sprintf("%d", int(fileSize))},
	}
	// ---

	// perform request
	uri, err := urlutil.Join(buildURL, "artifacts.json")
	if err != nil {
		return "", "", fmt.Errorf("failed to generate create artifact url, error: %s", err)
	}

	var response *http.Response

	// process response
	type createArtifactResponse struct {
		ErrorMessage string `json:"error_msg"`
		UploadURL    string `json:"upload_url"`
		ID           int    `json:"id"`
	}

	var artifactResponse createArtifactResponse

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

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read create artifact response, error: %s", err)
		}
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to create artifact on bitrise, status code: %d, response: %s", response.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, &artifactResponse); err != nil {
			return fmt.Errorf("failed to unmarshal response (%s), error: %s", string(body), err)
		}

		if artifactResponse.ErrorMessage != "" {
			return fmt.Errorf("failed to create artifact on bitrise, error message: %s", artifactResponse.ErrorMessage)
		}
		if artifactResponse.UploadURL == "" {
			return fmt.Errorf("failed to create artifact on bitrise, error: no upload url received")
		}
		if artifactResponse.ID == 0 {
			return fmt.Errorf("failed to create artifact on bitrise, error: no artifact id received")
		}

		return nil
	}); err != nil {
		return "", "", err
	}

	return artifactResponse.UploadURL, fmt.Sprintf("%d", artifactResponse.ID), nil
}

func uploadArtifact(uploadURL, artifactPth, contentType string) error {
	log.Printf("uploading artifact")

	netClient := &http.Client{
		Timeout: 10 * time.Minute,
	}

	return retry.Times(3).Wait(5).Try(func(attempt uint) error {
		file, err := os.Open(artifactPth)
		if err != nil {
			return fmt.Errorf("failed to open artifact, error: %s", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Warnf("failed to close file, error: %s", err)
			}
		}()

		// Set Content Length manually (https://stackoverflow.com/a/39764726), as it is part of signature in signed URL
		fileInfo, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file info for %s, error: %s", artifactPth, err)
		}

		// Initializes request body to nil to send a Content-Length of 0: https://github.com/golang/go/issues/20257#issuecomment-299509391
		var reqBody io.Reader
		if fileInfo.Size() > 0 {
			reqBody = ioutil.NopCloser(file)
		}

		request, err := http.NewRequest(http.MethodPut, uploadURL, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request, error: %s", err)
		}

		if contentType != "" {
			request.Header.Add("Content-Type", contentType)
		}

		request.ContentLength = fileInfo.Size()

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

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non success status code: %d, headers: %s, body: %s", resp.StatusCode, resp.Header, body)
		}

		return nil
	})
}

func finishArtifact(buildURL, token, artifactID, artifactInfo, notifyUserGroups, notifyEmails, isEnablePublicPage string) (ArtifactURLs, error) {
	log.Printf("finishing artifact")

	// create form data
	data := url.Values{"api_token": {token}}
	if artifactInfo != "" {
		data["artifact_info"] = []string{artifactInfo}
	}
	if notifyUserGroups != "" {
		data["notify_user_groups"] = []string{notifyUserGroups}
	}
	if notifyEmails != "" {
		data["notify_emails"] = []string{notifyEmails}
	}
	if isEnablePublicPage == "true" {
		data["is_enable_public_page"] = []string{"yes"}
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
		body, err := ioutil.ReadAll(response.Body)
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

	if isEnablePublicPage == "true" {
		if artifactResponse.PublicInstallPageURL == "" {
			return ArtifactURLs{}, fmt.Errorf("public install page was enabled, but no public install page generated")
		}

		return ArtifactURLs{
			PublicInstallPageURL: artifactResponse.PublicInstallPageURL,
			PermanentDownloadURL: artifactResponse.PermanentDownloadURL,
		}, nil
	}

	return ArtifactURLs{
		PermanentDownloadURL: artifactResponse.PermanentDownloadURL,
	}, nil
}
