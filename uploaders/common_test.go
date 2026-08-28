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
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
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
			if _, err := UploadArtifact(tt.uploadURL, artifact, tt.contentType, log.NewLogger()); (err != nil) != tt.wantErr {
				t.Errorf("UploadArtifact() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_uploadPart(t *testing.T) {
	content := make([]byte, 100)
	for i := range content {
		content[i] = byte(i)
	}
	filePath := writeTempFile(t, content)

	var gotBody []byte
	var gotContentLength int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentLength = r.ContentLength
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("ETag", `"test-etag"`)
	}))
	defer server.Close()

	etag, err := uploadPart(server.URL, filePath, 25, 50, 1, log.NewLogger())

	require.NoError(t, err)
	require.Equal(t, `"test-etag"`, etag)
	require.Equal(t, int64(50), gotContentLength)
	require.Equal(t, content[25:75], gotBody)
}

func Test_uploadAllParts(t *testing.T) {
	makeContent := func(size int) []byte {
		b := make([]byte, size)
		for i := range b {
			b[i] = byte(i % 256)
		}
		return b
	}

	t.Run("even split", func(t *testing.T) {
		content := makeContent(100)
		filePath := writeTempFile(t, content)

		var mu sync.Mutex
		var bodies [][]byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			bodies = append(bodies, body)
			mu.Unlock()
			w.Header().Set("ETag", `"etag"`)
		}))
		defer server.Close()

		parts, err := uploadAllParts(filePath, 100, 50, []string{server.URL, server.URL}, 2, log.NewLogger())

		require.NoError(t, err)
		require.Len(t, parts, 2)
		sort.Slice(bodies, func(i, j int) bool { return bodies[i][0] < bodies[j][0] })
		require.Equal(t, content[:50], bodies[0])
		require.Equal(t, content[50:], bodies[1])
	})

	t.Run("last part trimmed to remaining bytes", func(t *testing.T) {
		content := makeContent(101)
		filePath := writeTempFile(t, content)

		var mu sync.Mutex
		var sizes []int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			sizes = append(sizes, r.ContentLength)
			mu.Unlock()
			w.Header().Set("ETag", `"etag"`)
		}))
		defer server.Close()

		parts, err := uploadAllParts(filePath, 101, 51, []string{server.URL, server.URL}, 2, log.NewLogger())

		require.NoError(t, err)
		require.Len(t, parts, 2)
		sort.Slice(sizes, func(i, j int) bool { return sizes[i] > sizes[j] })
		require.Equal(t, []int64{51, 50}, sizes)
	})

	t.Run("rejects zero part size", func(t *testing.T) {
		filePath := writeTempFile(t, []byte("x"))

		_, err := uploadAllParts(filePath, 1, 0, []string{"http://unused"}, 1, log.NewLogger())

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid part size")
	})

	t.Run("rejects mismatch between part_urls count and the count implied by part size", func(t *testing.T) {
		filePath := writeTempFile(t, make([]byte, 100))

		// 100 bytes / 50-byte parts → 2 parts expected, but we pass 3 URLs
		_, err := uploadAllParts(filePath, 100, 50, []string{"http://a", "http://b", "http://c"}, 1, log.NewLogger())

		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match expected")
	})
}

func writeTempFile(t *testing.T, content []byte) string {
	t.Helper()
	f, err := os.CreateTemp("", "uploadtest-*")
	require.NoError(t, err)

	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.Write(content)

	require.NoError(t, err)
	require.NoError(t, f.Close())

	return f.Name()
}
