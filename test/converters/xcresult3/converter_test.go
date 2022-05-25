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
				Name: "BullsEyeTests", Tests: 5, Failures: 0, Skipped: 0, Errors: 0, Time: 0.9777920246124268,
				TestCases: []junit.TestCase{
					{
						Name: "testStartNewRoundUsesRandomValueFromApiRequest()", ClassName: "BullsEyeFakeTests",
						Time: 0.014459013938903809,
					},
					{
						Name: "testGameStyleCanBeChanged()", ClassName: "BullsEyeMockTests",
						Time: 0.00929105281829834,
					},
					{
						Name: "testScoreIsComputedPerformance()", ClassName: "BullsEyeTests",
						Time: 0.7373920679092407,
					},
					{
						Name: "testScoreIsComputedWhenGuessIsHigherThanTarget()", ClassName: "BullsEyeTests",
						Time: 0.004113912582397461,
					},
					{
						Name: "testScoreIsComputedWhenGuessIsLowerThanTarget()", ClassName: "BullsEyeTests",
						Time: 0.21253597736358643,
					},
				},
			},
			{
				Name: "BullsEyeSlowTests", Tests: 2, Failures: 0, Skipped: 0, Errors: 0, Time: 0.5334550142288208,
				TestCases: []junit.TestCase{
					{
						Name: "testApiCallCompletes()", ClassName: "BullsEyeSlowTests",
						Time: 0.2844870090484619,
					},
					{
						Name: "testValidApiCallGetsHTTPStatusCode200()", ClassName: "BullsEyeSlowTests",
						Time: 0.2489680051803589,
					},
				},
			},
			{
				Name: "BullsEyeUITests", Tests: 1, Failures: 0, Skipped: 0, Errors: 0, Time: 9.606701016426086,
				TestCases: []junit.TestCase{
					{
						Name: "testGameStyleSwitch()", ClassName: "BullsEyeUITests",
						Time: 9.606701016426086,
					},
				},
			},
			{ // testFlakyFeature() was flaky: failed once, and then passed the second run
				Name: "BullsEyeFlakyTests", Tests: 3, Failures: 1, Skipped: 1, Errors: 0, Time: 0.22202682495117188,
				TestCases: []junit.TestCase{
					{
						Name: "testFlakyFeature()", ClassName: "BullsEyeFlakyTests", Time: 0.1958599090576172,
						Failure: &junit.Failure{
							Value: "/Users/vagrant/git/BullsEyeFlakyTests/BullsEyeFlakyTests.swift:43 - XCTAssertEqual failed: (\"1\") is not equal to (\"0\") - Number is not even",
						},
					},
					{
						Name: "testFlakyFeature()", ClassName: "BullsEyeFlakyTests", Time: 0.00603795051574707,
					},
					{
						Name: "testFlakySkip()", ClassName: "BullsEyeSkippedTests", Time: 0.020128965377807617,
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
				Time:     0.43262600898742676,
				TestCases: []junit.TestCase{
					{
						Name:      "testFailure()",
						ClassName: "testProjectUITests",
						Time:      0.2580660581588745,
						Failure: &junit.Failure{
							Value: "/Users/alexeysomov/Library/Autosave Information/testProject/testProjectUITests/testProjectUITests.swift:30 - XCTAssertTrue failed",
						},
					},
					{
						Name:      "testSkip()",
						ClassName: "testProjectUITests",
						Time:      0.08595001697540283,
						Skipped:   &junit.Skipped{},
					},
					{
						Name:      "testSuccess()",
						ClassName: "testProjectUITests",
						Time:      0.08860993385314941,
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
