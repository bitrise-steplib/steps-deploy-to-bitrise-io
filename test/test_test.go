package test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	logV2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createDummyFilesInDirWithContent(dir, content string, fileNames []string) error {
	for _, file := range fileNames {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, file)), 0777); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

func Test_Upload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal("failed to create temp dir, error:", err)
	}

	testResponseID := "mock-test-id"
	testXMLContent := []byte("test xml content")
	testStepInfo := models.TestResultStepInfo{ID: "test-ID", Title: "test-Title", Version: "test-Version", Number: 19}
	testAssetPaths := []string{filepath.Join(tempDir, "image1.png"), filepath.Join(tempDir, "image2.png"), filepath.Join(tempDir, "image3.png")}

	if err := createDummyFilesInDirWithContent(tempDir, "dummy data", []string{"image1.png", "image2.png", "image3.png"}); err != nil {
		t.Fatal(err)
	}

	results := Results{
		Result{
			XMLContent:      testXMLContent,
			StepInfo:        testStepInfo,
			AttachmentPaths: testAssetPaths,
		},
	}

	go func() { //nolint:staticcheck // We should fix it one day, but it requires a bigger refactor
		router := mux.NewRouter()

		router.HandleFunc("/test/apps/{app_slug}/builds/{build_slug}/test_reports/{accessToken}", func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			if _, ok := vars["app_slug"]; !ok {
				t.Fatal("app_slug must be specified") //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}
			if _, ok := vars["build_slug"]; !ok {
				t.Fatal("build_slug must be specified") //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}

			var uploadReq UploadRequest
			if err := json.NewDecoder(r.Body).Decode(&uploadReq); err != nil {
				t.Fatal("failed to execute get request, error:", err) //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}

			response := UploadResponse{
				ID:        testResponseID,
				UploadURL: UploadURL{FileName: uploadReq.FileName, URL: "http://localhost:8893/teststorage/" + uploadReq.FileName},
			}

			for _, asset := range uploadReq.Assets {
				response.Assets = append(response.Assets, UploadURL{
					FileName: asset.FileName,
					URL:      "http://localhost:8893/teststorage/" + asset.FileName,
				})
			}

			b, err := json.Marshal(response)
			if err != nil {
				t.Fatal(err) //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}
			if _, err := w.Write(b); err != nil {
				t.Fatal("Failed to write to the writer, error:", err) //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}
		}).Methods("POST")

		router.HandleFunc("/teststorage/{file_name}", func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			fName, ok := vars["file_name"]
			if !ok {
				t.Fatal("file_name must be specified") //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}

			receivedData, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err) //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}

			if fName == "test_result.xml" {
				if string(receivedData) == string(testXMLContent) {
					w.WriteHeader(http.StatusOK)
					return
				}
			}

			for _, assetPath := range testAssetPaths {
				if filepath.Base(assetPath) == fName {
					fileData, err := os.ReadFile(assetPath)
					if err != nil {
						t.Fatal(err) //nolint:govet // We should fix it one day, but it requires a bigger refactor
					}

					if string(fileData) != string(receivedData) {
						t.Fatal("files are not the same!") //nolint:govet // We should fix it one day, but it requires a bigger refactor
					}

					w.WriteHeader(http.StatusOK)
					return
				}
			}

			w.WriteHeader(http.StatusNotAcceptable)
		}).Methods("PUT")

		router.HandleFunc("/test/apps/{app_slug}/builds/{build_slug}/test_reports/{id}/{accessToken}", func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			if _, ok := vars["app_slug"]; !ok {
				t.Fatal("app_slug must be specified") //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}
			if _, ok := vars["build_slug"]; !ok {
				t.Fatal("build_slug must be specified") //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}
			id, ok := vars["id"]
			if !ok {
				t.Fatal("id must be specified") //nolint:govet // We should fix it one day, but it requires a bigger refactor
			}

			if id != testResponseID {
				w.WriteHeader(http.StatusNotAcceptable)
			}

		}).Methods("PATCH")

		t.Fatal(http.ListenAndServe(":8893", router)) //nolint:staticcheck,govet // We should fix it one day, but it requires a bigger refactor
	}()

	time.Sleep(time.Second)

	if err := results.Upload("access-token", "http://localhost:8893/test", "test-app-slug", "test-build-slug", logV2.NewLogger()); err != nil {
		t.Fatalf("%v", errors.WithStack(err))
		return
	}
}

func Test_ParseXctestResults(t *testing.T) {
	sampleTestSummariesPlist := readFileString(t, filepath.Join("testdata", "ios_testsummaries_plist.golden"))
	sampleIOSXmlOutput := readFileString(t, filepath.Join("testdata", "ios_xml_output.golden"))

	// creating test results
	{
		testsDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		_, err = os.MkdirTemp(testsDir, "test-result")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		bundle, err := ParseTestResults(testsDir, false, pathutil.NewPathChecker(), pathutil.NewPathModifier(), logV2.NewLogger())
		if err != nil {
			t.Fatal("failed to get bundle, error:", err)
		}

		if len(bundle) != 0 {
			t.Fatal("should be 0 test asset pack")
		}
	}

	// creating android test results
	{
		testsDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		testDir, err := os.MkdirTemp(testsDir, "test-result")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		phaseDir, err := os.MkdirTemp(testDir, "phase")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		if err := createDummyFilesInDirWithContent(testDir, `{"title": "test title"}`, []string{"step-info.json"}); err != nil {
			t.Fatal("failed to create dummy files in dir, error:", err)
		}
		if err := createDummyFilesInDirWithContent(phaseDir, `{"name": "test name"}`, []string{"test-info.json"}); err != nil {
			t.Fatal("failed to create dummy files in dir, error:", err)
		}
		if err := createDummyFilesInDirWithContent(phaseDir, "test content", []string{"image.png", "image3.jpeg", "dirty.gif", "dirty.html", "logs.txt", "zzz.log"}); err != nil {
			t.Fatal("failed to create dummy files in dir, error:", err)
		}
		if err := createDummyFilesInDirWithContent(phaseDir, sampleIOSXmlOutput, []string{"result.xml"}); err != nil {
			t.Fatal("failed to create dummy files in dir, error:", err)
		}

		bundle, err := ParseTestResults(testsDir, false, pathutil.NewPathChecker(), pathutil.NewPathModifier(), logV2.NewLogger())
		if err != nil {
			t.Fatal("failed to get bundle, error:", err)
		}
		if len(bundle) != 1 {
			t.Fatalf("should be 1 test asset pack: %#v", bundle)
		}

		assert.Equal(t, sampleIOSXmlOutput, string(bundle[0].XMLContent))
		// Check if the attachments are correctly by the end of paths
		assert.Equal(t, 4, len(bundle[0].AttachmentPaths))
		assert.True(t, strings.HasSuffix(bundle[0].AttachmentPaths[0], "image.png"))
		assert.True(t, strings.HasSuffix(bundle[0].AttachmentPaths[1], "image3.jpeg"))
		assert.True(t, strings.HasSuffix(bundle[0].AttachmentPaths[2], "logs.txt"))
		assert.True(t, strings.HasSuffix(bundle[0].AttachmentPaths[3], "zzz.log"))
	}

	// creating ios test results
	{
		testsDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		testDir, err := os.MkdirTemp(testsDir, "test-result")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		phaseDir, err := os.MkdirTemp(testDir, "phase")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		if err := createDummyFilesInDirWithContent(testDir, `{"title": "test title"}`, []string{"step-info.json"}); err != nil {
			t.Fatal("failed to create dummy files in dir, error:", err)
		}
		if err := createDummyFilesInDirWithContent(phaseDir, `{"name": "test name"}`, []string{"test-info.json"}); err != nil {
			t.Fatal("failed to create dummy files in dir, error:", err)
		}
		if err := createDummyFilesInDirWithContent(phaseDir, sampleTestSummariesPlist, []string{"mytests.xcresult/TestSummaries.plist"}); err != nil {
			t.Fatal("failed to create dummy files in dir, error:", err)
		}

		bundle, err := ParseTestResults(testsDir, false, pathutil.NewPathChecker(), pathutil.NewPathModifier(), logV2.NewLogger())
		if err != nil {
			t.Fatal("failed to get bundle, error:", err)
		}

		if len(bundle) != 1 {
			t.Fatal("should be 1 test asset pack")
		}

		assert.Equal(t, sampleIOSXmlOutput, string(bundle[0].XMLContent))
	}
}

func Test_ParseXctest3Results(t *testing.T) {
	// xcresulttool renders attachment timestamps in the process timezone; the golden is UTC.
	t.Setenv("TZ", "UTC")
	tmpDir := t.TempDir()

	testDir := path.Join(tmpDir, "tests")
	testResultDir := path.Join(testDir, "test-result")
	err := os.MkdirAll(testDir, os.ModePerm)
	require.NoError(t, err)

	phaseDir := path.Join(testResultDir, "phase")
	err = os.MkdirAll(testDir, os.ModePerm)
	require.NoError(t, err)

	if err := createDummyFilesInDirWithContent(testResultDir, `{"title": "test title"}`, []string{"step-info.json"}); err != nil {
		t.Fatal("failed to create dummy files in dir, error:", err)
	}
	if err := createDummyFilesInDirWithContent(phaseDir, `{"name": "test name"}`, []string{"test-info.json"}); err != nil {
		t.Fatal("failed to create dummy files in dir, error:", err)
	}

	oldDir := resolveSampleArtifact(t, "xcresults/xcresult3-device-configuration-tests.xcresult")
	newDir := path.Join(phaseDir, "xcresult3-device-configuration-tests.xcresult")
	copyCmd := command.NewFactory(env.NewRepository()).Create("cp", []string{"-a", oldDir, newDir}, nil)
	err = copyCmd.Run()
	require.NoError(t, err)

	bundle, err := ParseTestResults(testDir, false, pathutil.NewPathChecker(), pathutil.NewPathModifier(), logV2.NewLogger())
	require.NoError(t, err)

	want := readFileString(t, filepath.Join("testdata", "ios_device_config_xml_output.golden"))

	assert.Equal(t, 1, len(bundle))
	assert.Equal(t, want, string(bundle[0].XMLContent))
}

func Test_findSupportedAttachments(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_attachments")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	files := []string{
		"screenshot.JPG",
		"screenshot2.png",
		"log.txt",
		"video.mp4",
		"recording.webm",
		"subfolder/nested.jpg",
		"subfolder/clip.ogg",
		"subfolder/deep/image.png",
		"subfolder/deep/movie.mp4",
	}
	err = createDummyFilesInDirWithContent(tempDir, "test", files)
	require.NoError(t, err)

	result := findSupportedAttachments(tempDir, logV2.NewLogger())

	assert.Len(t, result, 9) // all supported files including videos
}
