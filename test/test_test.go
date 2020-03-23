package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
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

	go func() {
		router := mux.NewRouter()

		router.HandleFunc("/test/apps/{app_slug}/builds/{build_slug}/test_reports/{accessToken}", func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			if _, ok := vars["app_slug"]; !ok {
				t.Fatal("app_slug must be specified")
			}
			if _, ok := vars["build_slug"]; !ok {
				t.Fatal("build_slug must be specified")
			}

			var uploadReq UploadRequest
			if err := json.NewDecoder(r.Body).Decode(&uploadReq); err != nil {
				t.Fatal("failed to execute get request, error:", err)
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
				t.Fatal(err)
			}
			if _, err := w.Write(b); err != nil {
				t.Fatal("Failed to write to the writer, error:", err)
			}
		}).Methods("POST")

		router.HandleFunc("/teststorage/{file_name}", func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			fName, ok := vars["file_name"]
			if !ok {
				t.Fatal("file_name must be specified")
			}

			receivedData, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
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
						t.Fatal(err)
					}

					if fileData != string(receivedData) {
						t.Fatal("files are not the same!")
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
				t.Fatal("app_slug must be specified")
			}
			if _, ok := vars["build_slug"]; !ok {
				t.Fatal("build_slug must be specified")
			}
			id, ok := vars["id"]
			if !ok {
				t.Fatal("id must be specified")
			}

			if id != testResponseID {
				w.WriteHeader(http.StatusNotAcceptable)
			}

		}).Methods("PATCH")

		t.Fatal(http.ListenAndServe(":8893", router))
	}()

	time.Sleep(time.Second)

	if err := results.Upload("access-token", "http://localhost:8893/test", "test-app-slug", "test-build-slug"); err != nil {
		t.Fatalf("%v", errors.WithStack(err))
		return
	}
}

func Test_ParseTestResults(t *testing.T) {
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

		_, err = ioutil.TempDir(testsDir, "test-result")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		bundle, err := ParseTestResults(testsDir)
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

		testDir, err := ioutil.TempDir(testsDir, "test-result")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		phaseDir, err := ioutil.TempDir(testDir, "phase")
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

		bundle, err := ParseTestResults(testsDir)
		if err != nil {
			t.Fatal("failed to get bundle, error:", err)
		}
		if len(bundle) != 1 {
			t.Fatalf("should be 1 test asset pack: %#v", bundle)
		}

		if len(bundle[0].XMLContent) != len(sampleIOSXmlOutput) {
			t.Fatal(fmt.Sprintf("wrong xml content: %s", bundle[0].XMLContent))
		}
	}

	// creating ios test results
	{
		testsDir, err := pathutil.NormalizedOSTempDirPath("test")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		testDir, err := ioutil.TempDir(testsDir, "test-result")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}

		phaseDir, err := ioutil.TempDir(testDir, "phase")
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

		bundle, err := ParseTestResults(testsDir)
		if err != nil {
			t.Fatal("failed to get bundle, error:", err)
		}

		if len(bundle) != 1 {
			t.Fatal("should be 1 test asset pack")
		}

		if len(string(bundle[0].XMLContent)) != len(sampleIOSXmlOutput) {
			t.Fatal("wrong xml content\n" + string(bundle[0].XMLContent) + "\n\n" + sampleIOSXmlOutput)
		}
	}
}
