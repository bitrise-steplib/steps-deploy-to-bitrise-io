package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	logV2 "github.com/bitrise-io/go-utils/v2/log"
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
		if err := fileutil.WriteStringToFile(filepath.Join(dir, file), content); err != nil {
			return err
		}
	}
	return nil
}

func Test_Upload(t *testing.T) {
	tempDir, err := pathutil.NormalizedOSTempDirPath("test")
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
			XMLContent: testXMLContent,
			StepInfo:   testStepInfo,
			ImagePaths: testAssetPaths,
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

			receivedData, err := ioutil.ReadAll(r.Body)
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
					fileData, err := fileutil.ReadStringFromFile(assetPath)
					if err != nil {
						t.Fatal(err) //nolint:govet // We should fix it one day, but it requires a bigger refactor
					}

					if fileData != string(receivedData) {
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
	sampleTestSummariesPlist, err := fileutil.ReadStringFromFile(filepath.Join("testdata", "ios_testsummaries_plist.golden"))
	if err != nil {
		t.Fatal("unable to read golden file, error:", err)
	}
	sampleIOSXmlOutput, err := fileutil.ReadStringFromFile(filepath.Join("testdata", "ios_xml_output.golden"))
	if err != nil {
		t.Fatal("unable to read golden file, error:", err)
	}

	// creating test results
	{
		testsDir, err := pathutil.NormalizedOSTempDirPath("test")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		_, err = os.MkdirTemp(testsDir, "test-result")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		bundle, err := ParseTestResults(testsDir, logV2.NewLogger())
		if err != nil {
			t.Fatal("failed to get bundle, error:", err)
		}

		if len(bundle) != 0 {
			t.Fatal("should be 0 test asset pack")
		}
	}

	// creating android test results
	{
		testsDir, err := pathutil.NormalizedOSTempDirPath("test")
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
		if err := createDummyFilesInDirWithContent(phaseDir, "test content", []string{"image.png", "image3.jpeg", "dirty.gif", "dirty.html"}); err != nil {
			t.Fatal("failed to create dummy files in dir, error:", err)
		}
		if err := createDummyFilesInDirWithContent(phaseDir, sampleIOSXmlOutput, []string{"result.xml"}); err != nil {
			t.Fatal("failed to create dummy files in dir, error:", err)
		}

		bundle, err := ParseTestResults(testsDir, logV2.NewLogger())
		if err != nil {
			t.Fatal("failed to get bundle, error:", err)
		}
		if len(bundle) != 1 {
			t.Fatalf("should be 1 test asset pack: %#v", bundle)
		}

		assert.Equal(t, sampleIOSXmlOutput, string(bundle[0].XMLContent))
	}

	// creating ios test results
	{
		testsDir, err := pathutil.NormalizedOSTempDirPath("test")
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

		bundle, err := ParseTestResults(testsDir, logV2.NewLogger())
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
	tmpDir := t.TempDir()
	gitDir := path.Join(tmpDir, "git")

	// The xcresult3 format has many small encoded binary files, so it is better to use a real xcresult file.
	// We are storing these in the sample-artifacts git repo.
	cmd := command.NewFactory(env.NewRepository()).Create("git", []string{"clone", "--depth", "1", "https://github.com/bitrise-io/sample-artifacts.git", gitDir}, nil)
	err := cmd.Run()
	require.NoError(t, err)

	testDir := path.Join(tmpDir, "tests")
	testResultDir := path.Join(testDir, "test-result")
	err = os.MkdirAll(testDir, os.ModePerm)
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

	oldDir := path.Join(gitDir, "xcresults", "xcresult3-device-configuration-tests.xcresult")
	newDir := path.Join(phaseDir, "xcresult3-device-configuration-tests.xcresult")
	copyCmd := command.NewFactory(env.NewRepository()).Create("cp", []string{"-a", oldDir, newDir}, nil)
	err = copyCmd.Run()
	require.NoError(t, err)

	bundle, err := ParseTestResults(testDir, logV2.NewLogger())
	require.NoError(t, err)

	want, err := fileutil.ReadStringFromFile(filepath.Join("testdata", "ios_device_config_xml_output.golden"))
	require.NoError(t, err)

	assert.Equal(t, 1, len(bundle))
	assert.Equal(t, want, string(bundle[0].XMLContent))
}
