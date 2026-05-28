package uploaders

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bitrise-io/go-utils/v2/env"
	logV2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
	"github.com/stretchr/testify/require"
)

func newTestUploader(t *testing.T, useMultipart bool) *Uploader {
	t.Helper()
	logger := logV2.NewLogger()
	return &Uploader{
		logger:               logger,
		tracker:              newTracker(env.NewRepository(), logger),
		useMultipartUpload:   useMultipart,
		multipartConcurrency: 1,
	}
}

func Test_upload_singlePath(t *testing.T) {
	content := []byte("test content")
	filePath := writeTempFile(t, content)

	var singleCalled, multipartCalled bool

	storageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer storageServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/artifacts.json":
			singleCalled = true
			json.NewEncoder(w).Encode([]UploadTask{{URL: storageServer.URL, ID: 1}}) //nolint:errcheck
		case "/artifacts/create_multipart_upload.json":
			multipartCalled = true
			w.WriteHeader(http.StatusInternalServerError)
		case "/artifacts/1/finish_upload.json":
			json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
				"permanent_download_url": "https://example.com/download",
				"details_page_url":       "https://example.com/details",
			})
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer apiServer.Close()

	item := &deployment.DeployableItem{Path: filePath}
	artifact := ArtifactArgs{Path: filePath, FileSize: int64(len(content))}

	urls, err := newTestUploader(t, false).upload(apiServer.URL, "token", artifact, "file", "", item, nil)

	require.NoError(t, err)
	require.Len(t, urls, 1)
	require.Equal(t, "https://example.com/download", urls[0].PermanentDownloadURL)
	require.Equal(t, "https://example.com/details", urls[0].DetailsPageURL)
	require.True(t, singleCalled, "expected /artifacts.json to be called")
	require.False(t, multipartCalled, "expected /artifacts/create_multipart_upload.json NOT to be called")
}

func Test_upload_multipartPath(t *testing.T) {
	content := []byte("test content")
	filePath := writeTempFile(t, content)

	var singleCalled, multipartCalled bool

	storageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"test-etag"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer storageServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/artifacts.json":
			singleCalled = true
			w.WriteHeader(http.StatusInternalServerError)
		case "/artifacts/create_multipart_upload.json":
			multipartCalled = true
			json.NewEncoder(w).Encode([]MultipartUploadTask{{ //nolint:errcheck
				PartURLs: []string{storageServer.URL},
				ID:       1,
				PartSize: int64(len(content)),
			}})
		case "/artifacts/1/finish_multipart_upload.json":
			json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
				"permanent_download_url": "https://example.com/download",
				"details_page_url":       "https://example.com/details",
			})
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer apiServer.Close()

	item := &deployment.DeployableItem{Path: filePath}
	artifact := ArtifactArgs{Path: filePath, FileSize: int64(len(content))}

	urls, err := newTestUploader(t, true).upload(apiServer.URL, "token", artifact, "file", "", item, nil)

	require.NoError(t, err)
	require.Len(t, urls, 1)
	require.Equal(t, "https://example.com/download", urls[0].PermanentDownloadURL)
	require.Equal(t, "https://example.com/details", urls[0].DetailsPageURL)
	require.False(t, singleCalled, "expected /artifacts.json NOT to be called")
	require.True(t, multipartCalled, "expected /artifacts/create_multipart_upload.json to be called")
}
