package uploaders

import (
	"image"
	"image/png"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_uploadArtifact(t *testing.T) {
	const contentType = "image/png"

	file, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("setup: failed to create file, error: %s", err)
	}
	testFilePath, err := filepath.Abs(file.Name())
	if err != nil {
		t.Fatalf("setup: failed to get file path, error: %s", err)
	}

	img := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{rand.Intn(1000) + 1, rand.Intn(1000) + 1}})
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("setup: failed to write file, error: %s", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("setup: failed to get file info, error: %s", err)
	}
	wantFileSize := fileInfo.Size()

	if err := file.Close(); err != nil {
		t.Errorf("setup: failed to close file")
	}

	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		t.Logf("Content type: %s", r.Header.Get("Content-Type"))

		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("httptest: failed to read request, error: %s", err)
			return
		}

		if r.ContentLength != wantFileSize {
			t.Fatalf("httptest: Content-length got %d want %d", r.ContentLength, wantFileSize)
		}

		if r.Header.Get("X-Upload-Content-Length") != strconv.FormatInt(wantFileSize, 10) {
			t.Fatalf("httptest: X-Upload-Content-Length got %s want %d", r.Header.Get("X-Upload-Content-Length"), wantFileSize)
		}

		if r.Header.Get("Content-Type") != contentType {
			t.Fatalf("httptest: content type got: %s want: %s", r.Header.Get("Content-Type"), contentType)
		}

		if int64(len(bytes)) != wantFileSize {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name        string
		uploadURL   string
		artifactPth string
		contentType string
		wantErr     bool
	}{
		{
			name:        "Happy path",
			uploadURL:   storage.URL,
			artifactPth: testFilePath,
			contentType: contentType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileInfo, err := os.Stat(tt.artifactPth)
			require.NoError(t, err)
			artifact := ArtifactArgs{
				Path:     tt.artifactPth,
				FileSize: fileInfo.Size(),
			}
			if _, err := UploadArtifact(tt.uploadURL, artifact, tt.contentType); (err != nil) != tt.wantErr {
				t.Errorf("UploadArtifact() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
