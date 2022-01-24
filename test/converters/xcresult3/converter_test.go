package xcresult3

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
	"github.com/stretchr/testify/require"
)

// copyTestdataToDir ...
// To populate the _tmp dir with sample data
// run `bitrise run download_sample_artifacts` before running tests here,
// which will download https://github.com/bitrise-io/sample-artifacts
// into the _tmp dir.
func copyTestdataToDir(t *testing.T, pathInTestdataDir, dirPathToCopyInto string) string {
	err := command.CopyDir(
		filepath.Join("../../../_tmp/", pathInTestdataDir),
		dirPathToCopyInto,
		true,
	)
	require.NoError(t, err)
	return dirPathToCopyInto
}

func TestConverter_XML(t *testing.T) {
	t.Run("xcresults3 success-failed-skipped-tests.xcresult", func(t *testing.T) {
		// copy test data to tmp dir
		tempTestdataDir, err := pathutil.NormalizedOSTempDirPath("test")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}
		defer func() {
			require.NoError(t, os.RemoveAll(tempTestdataDir))
		}()
		t.Log("tempTestdataDir: ", tempTestdataDir)
		tempXCResultPath := copyTestdataToDir(t, "./xcresults/xcresult3-success-failed-skipped-tests.xcresult", tempTestdataDir)

		c := Converter{
			xcresultPth: tempXCResultPath,
		}
		junitXML, err := c.XML()
		require.NoError(t, err)
		require.Equal(t, []junit.TestSuite{
			junit.TestSuite{
				Name:     "testProjectUITests",
				Tests:    3,
				Failures: 1,
				Skipped:  1,
				Errors:   0,
				Time:     0.43262600898742676,
				TestCases: []junit.TestCase{
					junit.TestCase{
						Name:      "testFailure()",
						ClassName: "testProjectUITests",
						Time:      0.2580660581588745,
						Failure: &junit.Failure{
							Value: "file:///Users/alexeysomov/Library/Autosave%20Information/testProject/testProjectUITests/testProjectUITests.swift:CharacterRangeLen=0&EndingLineNumber=29&StartingLineNumber=29 - XCTAssertTrue failed",
						},
					},
					junit.TestCase{
						Name:      "testSkip()",
						ClassName: "testProjectUITests",
						Time:      0.08595001697540283,
						Skipped:   &junit.Skipped{},
					},
					junit.TestCase{
						Name:      "testSuccess()",
						ClassName: "testProjectUITests",
						Time:      0.08860993385314941,
					},
				},
			},
		}, junitXML.TestSuites)
	})
}
