package uploaders

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/urlutil"
)

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

	response, err := http.PostForm(uri, data)
	if err != nil {
		return "", "", fmt.Errorf("failed to perform create artifact request, error: %s", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Errorf("Failed to close reponse body, error: %s", err)
		}
	}()
	// --

	// process response
	type createArtifactResponse struct {
		ErrorMessage string `json:"error_msg"`
		UploadURL    string `json:"upload_url"`
		ID           int    `json:"id"`
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read create artifact response, error: %s", err)
	}
	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to create artifact on bitrise, status code: %d, response: %s", response.StatusCode, string(body))
	}

	var artifactResponse createArtifactResponse
	if err := json.Unmarshal(body, &artifactResponse); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal response (%s), error: %s", string(body), err)
	}

	if artifactResponse.ErrorMessage != "" {
		return "", "", fmt.Errorf("failed to create artifact on bitrise, error message: %s", artifactResponse.ErrorMessage)
	}
	if artifactResponse.UploadURL == "" {
		return "", "", fmt.Errorf("failed to create artifact on bitrise, error: no upload url got")
	}
	if artifactResponse.ID == 0 {
		return "", "", fmt.Errorf("failed to create artifact on bitrise, error: no artifact id got")
	}
	// ---

	return artifactResponse.UploadURL, fmt.Sprintf("%d", artifactResponse.ID), nil
}

func uploadArtifact(uploadURL, artifactPth, contentType string) error {
	log.Printf("uploading artifact")

	data, err := os.Open(artifactPth)
	if err != nil {
		return fmt.Errorf("failed to open artifact, error: %s", err)
	}
	defer func() {
		if err := data.Close(); err != nil {
			log.Errorf("Failed to close file, error: %s", err)
		}
	}()

	args := []string{"curl", "--fail", "--tlsv1", "--globoff"}
	if contentType != "" {
		args = append(args, "-H", fmt.Sprintf("Content-Type: %s", contentType))
	}
	args = append(args, "-T", artifactPth, "-X", "PUT", uploadURL)

	cmd, err := command.NewFromSlice(args)
	if err != nil {
		return err
	}

	cmd.SetStdout(os.Stdout).SetStderr(os.Stderr)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func finishArtifact(buildURL, token, artifactID, artifactInfo, notifyUserGroups, notifyEmails, isEnablePublicPage string) (string, error) {
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
		return "", fmt.Errorf("failed to generate finish artifact url, error: %s", err)
	}

	response, err := http.PostForm(uri, data)
	if err != nil {
		return "", fmt.Errorf("failed to perform finish artifact request, error: %s", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Errorf("Failed to close reponse body, error: %s", err)
		}
	}()
	// --

	// process response
	type finishArtifactResponse struct {
		PublicInstallPageURL string `json:"public_install_page_url"`
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read finish artifact response, error: %s", err)
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create artifact on bitrise, status code: %d, response: %s", response.StatusCode, string(body))
	}

	var artifactResponse finishArtifactResponse
	if err := json.Unmarshal(body, &artifactResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal response (%s), error: %s", string(body), err)
	}
	// ---

	if isEnablePublicPage == "true" {
		if artifactResponse.PublicInstallPageURL == "" {
			return "", fmt.Errorf("public install page was enabled, but no public install page generated")
		}

		return artifactResponse.PublicInstallPageURL, nil
	}
	return "", nil
}
