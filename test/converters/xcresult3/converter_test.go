package xcresult3

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
)

// copyTestdataToDir ...
// To populate the _tmp dir with sample data
// run `bitrise run download_sample_artifacts` before running tests here,
// which will download https://github.com/bitrise-io/sample-artifacts
// into the _tmp dir.
func copyTestdataToDir(t require.TestingT, pathInTestdataDir, dirPathToCopyInto string) string {
	err := command.CopyDir(
		filepath.Join("../../../_tmp/", pathInTestdataDir),
		dirPathToCopyInto,
		true,
	)
	require.NoError(t, err)
	return dirPathToCopyInto
}

func TestConverter_XML(t *testing.T) {
	t.Run("xcresult3-flaky-with-rerun.xcresult", func(t *testing.T) {
		// copy test data to tmp dir
		tempTestdataDir, err := pathutil.NormalizedOSTempDirPath("test")
		if err != nil {
			t.Fatal("failed to create temp dir, error:", err)
		}
		defer func() {
			require.NoError(t, os.RemoveAll(tempTestdataDir))
		}()
		t.Log("tempTestdataDir: ", tempTestdataDir)
		tempXCResultPath := copyTestdataToDir(t, "./xcresults/xcresult3-flaky-with-rerun.xcresult", tempTestdataDir)

		c := Converter{
			xcresultPth: tempXCResultPath,
		}
		junitXML, err := c.XML()
		require.NoError(t, err)
		require.Equal(t, []junit.TestSuite{
			{
				Name: "BullsEyeTests", Tests: 5, Failures: 0, Skipped: 0, Errors: 0, Time: 0.9774,
				TestCases: []junit.TestCase{
					{
						Name: "testStartNewRoundUsesRandomValueFromApiRequest()", ClassName: "BullsEyeFakeTests",
						Time: 0.014,
					},
					{
						Name: "testGameStyleCanBeChanged()", ClassName: "BullsEyeMockTests",
						Time: 0.0093,
					},
					{
						Name: "testScoreIsComputedPerformance()", ClassName: "BullsEyeTests",
						Time: 0.74,
					},
					{
						Name: "testScoreIsComputedWhenGuessIsHigherThanTarget()", ClassName: "BullsEyeTests",
						Time: 0.0041,
					},
					{
						Name: "testScoreIsComputedWhenGuessIsLowerThanTarget()", ClassName: "BullsEyeTests",
						Time: 0.21,
					},
				},
			},
			{
				Name: "BullsEyeSlowTests", Tests: 2, Failures: 0, Skipped: 0, Errors: 0, Time: 0.53,
				TestCases: []junit.TestCase{
					{
						Name: "testApiCallCompletes()", ClassName: "BullsEyeSlowTests",
						Time: 0.28,
					},
					{
						Name: "testValidApiCallGetsHTTPStatusCode200()", ClassName: "BullsEyeSlowTests",
						Time: 0.25,
					},
				},
			},
			{
				Name: "BullsEyeUITests", Tests: 1, Failures: 0, Skipped: 0, Errors: 0, Time: 9,
				TestCases: []junit.TestCase{
					{
						Name: "testGameStyleSwitch()", ClassName: "BullsEyeUITests",
						Time: 9,
					},
				},
			},
			{ // testFlakyFeature() was flaky: failed once, and then passed the second run
				Name: "BullsEyeFlakyTests", Tests: 2, Failures: 0, Skipped: 1, Errors: 0, Time: 0.12,
				TestCases: []junit.TestCase{
					{
						Name: "testFlakyFeature()", ClassName: "BullsEyeFlakyTests", Time: 0.1,
					},
					{
						Name: "testFlakySkip()", ClassName: "BullsEyeSkippedTests", Time: 0.02,
						Skipped: &junit.Skipped{},
					},
				},
			},
		}, junitXML.TestSuites)
	})

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
			{
				Name:     "testProjectUITests",
				Tests:    3,
				Failures: 1,
				Skipped:  1,
				Errors:   0,
				Time:     0.435,
				TestCases: []junit.TestCase{
					{
						Name:      "testFailure()",
						ClassName: "testProjectUITests",
						Time:      0.26,
						Failure: &junit.Failure{
							Value: "testProjectUITests.swift:30: XCTAssertTrue failed",
						},
					},
					{
						Name:      "testSkip()",
						ClassName: "testProjectUITests",
						Time:      0.086,
						Skipped:   &junit.Skipped{},
					},
					{
						Name:      "testSuccess()",
						ClassName: "testProjectUITests",
						Time:      0.089,
					},
				},
			},
		}, junitXML.TestSuites)
	})
}

func BenchmarkConverter_XML(b *testing.B) {
	// copy test data to tmp dir
	tempTestdataDir, err := pathutil.NormalizedOSTempDirPath("test")
	if err != nil {
		b.Fatal("failed to create temp dir, error:", err)
	}
	defer func() {
		require.NoError(b, os.RemoveAll(tempTestdataDir))
	}()
	b.Log("tempTestdataDir: ", tempTestdataDir)
	tempXCResultPath := copyTestdataToDir(b, "./xcresults/xcresult3-flaky-with-rerun.xcresult", tempTestdataDir)

	c := Converter{
		xcresultPth: tempXCResultPath,
	}
	_, err = c.XML()
	require.NoError(b, err)
}
