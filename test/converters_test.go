package test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/testreport"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-xcode/v2/testresult/xcresult3"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

const sampleArtifactsGitURL = "https://github.com/bitrise-io/sample-artifacts.git"

func TestXCresult3Converters(t *testing.T) {
	// xcresulttool renders attachment timestamps in the process timezone; the expected reports are UTC.
	t.Setenv("TZ", "UTC")
	log.SetEnableDebugLog(true)
	want := testreport.TestReport{
		TestSuites: []testreport.TestSuite{
			{ // unit test
				Name:     "rtgtrghtrgTests",
				Tests:    2,
				Failures: 0,
				Errors:   0,
				Time:     0.26063,
				TestCases: []testreport.TestCase{
					{ // plain test case
						Name:      "testExample()",
						ClassName: "rtgtrghtrgTests",
						Time:      0.00063,
					},
					{ // plain test case
						Name:      "testPerformanceExample()",
						ClassName: "rtgtrghtrgTests",
						Time:      0.26,
					},
				},
			},
			{ // ui test
				Name:     "rtgtrghtrgUITests",
				Tests:    15,
				Failures: 3,
				Errors:   0,
				Time:     0.759,
				TestCases: []testreport.TestCase{
					// class rtgtrghtrg3UITests: XCTestCase inside rtgtrghtrgUITests class
					{
						Name:      "testExample",
						ClassName: "_TtCC17rtgtrghtrgUITests17rtgtrghtrgUITests18rtgtrghtrg3UITests",
						Time:      0.032,
					},
					{
						Name:      "testFailMe",
						ClassName: "_TtCC17rtgtrghtrgUITests17rtgtrghtrgUITests18rtgtrghtrg3UITests",
						Time:      0.09,
						Failure: &testreport.Failure{
							Value: "XCTAssertTrue failed",
						},
						Properties: &testreport.Properties{
							Property: []testreport.Property{
								{
									Name:  "attachment_0",
									Value: "Screenshot 2019-11-25 at 12.28.29 PM_1574684909530999898.jpeg",
								},
								{
									Name:  "attachment_1",
									Value: "Screenshot 2019-11-25 at 12.28.29 PM_1574684909592000007.jpeg",
								},
							},
						},
					},
					{
						Name:      "testLaunchPerformance",
						ClassName: "_TtCC17rtgtrghtrgUITests17rtgtrghtrgUITests18rtgtrghtrg3UITests",
						Time:      0.036,
					},
					// class rtgtrghtrg2UITests: XCTestCase
					{
						Name:      "testExample()",
						ClassName: "rtgtrghtrg2UITests",
						Time:      0.061,
					},
					{
						Name:      "testFailMe()",
						ClassName: "rtgtrghtrg2UITests",
						Time:      0.085,
						Failure: &testreport.Failure{
							Value: "XCTAssertTrue failed",
						},
						Properties: &testreport.Properties{
							Property: []testreport.Property{
								{
									Name:  "attachment_0",
									Value: "Screenshot 2019-11-25 at 12.28.29 PM_1574684909736999988.jpeg",
								},
								{
									Name:  "attachment_1",
									Value: "Screenshot 2019-11-25 at 12.28.29 PM_1574684909776999950.jpeg",
								},
							},
						},
					},
					{
						Name:      "testLaunchPerformance()",
						ClassName: "rtgtrghtrg2UITests",
						Time:      0.042,
					},
					// class rtgtrghtrg4UITests: rtgtrghtrgUITests (so rtgtrghtrg4UITests inherits rtgtrghtrgUITests -> test cases merged and the base class name is rtgtrghtrg4UITests)
					{
						Name:      "testExample()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.071,
					},
					{
						Name:      "testExample2()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.043,
					},
					{
						Name:      "testFailMe()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.043,
					},
					{
						Name:      "testFailMe2()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.084,
						Failure: &testreport.Failure{
							Value: "XCTAssertTrue failed",
						},
						Properties: &testreport.Properties{
							Property: []testreport.Property{
								{
									Name:  "attachment_0",
									Value: "Screenshot 2019-11-25 at 12.28.30 PM_1574684910020999908.jpeg",
								},
								{
									Name:  "attachment_1",
									Value: "Screenshot 2019-11-25 at 12.28.30 PM_1574684910062000036.jpeg",
								},
							},
						},
					},
					{
						Name:      "testLaunchPerformance()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.048,
					},
					{
						Name:      "testLaunchPerformance2()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.031,
					},
					// class rtgtrghtrgUITests: XCTestCase
					{
						Name:      "testExample()",
						ClassName: "rtgtrghtrgUITests",
						Time:      0.031,
					},
					{
						Name:      "testFailMe()",
						ClassName: "rtgtrghtrgUITests",
						Time:      0.031,
					},
					{
						Name:      "testLaunchPerformance()",
						ClassName: "rtgtrghtrgUITests",
						Time:      0.031,
					},
				},
			},
		},
	}

	_, b, _, _ := runtime.Caller(0)
	testPackageDir := filepath.Dir(b)

	multiLevelUITestsXCResult := resolveSampleArtifact(t, "xcresults/xcresult3_multi_level_UI_tests.xcresult")

	for _, test := range []struct {
		name          string
		converter     testreport.Converter
		testFilePaths []string
		wantDetect    bool
		wantXML       testreport.TestReport
		wantXMLError  bool
	}{
		{
			name:          "xcresult3",
			converter:     xcresult3.NewConverter(false),
			testFilePaths: []string{multiLevelUITestsXCResult},
			wantDetect:    true,
			wantXMLError:  false,
			wantXML:       want,
		},
		{
			name:          "Long running test",
			converter:     xcresult3.NewConverter(false),
			testFilePaths: []string{filepath.Join(testPackageDir, "testdata/test_result_with_18m_long_test_case.xcresult")},
			wantDetect:    true,
			wantXMLError:  false,
			wantXML: testreport.TestReport{
				TestSuites: []testreport.TestSuite{
					{
						Name:     "BullsEyeSlowTests",
						Tests:    1,
						Failures: 0,
						Time:     1080,
						TestCases: []testreport.TestCase{
							{
								Name:      "testSleepingFor16mins()",
								ClassName: "BullsEyeSlowTests",
								Time:      1080,
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotDetected := test.converter.Detect(test.testFilePaths)
			require.Equal(t, test.wantDetect, gotDetected)

			got, err := test.converter.Convert()
			if test.wantXMLError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			opts := []cmp.Option{
				cmp.Transformer("SortTestSuites", func(in []testreport.TestSuite) []testreport.TestSuite {
					s := append([]testreport.TestSuite{}, in...)
					sort.Slice(s, func(i, j int) bool {
						return s[i].Time > s[j].Time
					})
					return s
				}),
				cmp.Transformer("SortTestCases", func(in []testreport.TestCase) []testreport.TestCase {
					s := append([]testreport.TestCase{}, in...)
					sort.Slice(s, func(i, j int) bool {
						return s[i].Time > s[j].Time
					})
					return s
				}),
			}

			if diff := cmp.Diff(test.wantXML, got, opts...); diff != "" {
				t.Fatalf("Test report mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// resolveSampleArtifact returns a path to a fixture from the sample-artifacts repo. It reuses the
// _tmp checkout the _download_sample_artifacts CI workflow creates when the fixture is present, and
// otherwise clones into that same _tmp dir, so local runs share one checkout with CI and re-cloning
// is avoided across runs.
func resolveSampleArtifact(t *testing.T, relPath string) string {
	t.Helper()

	_, thisFile, _, _ := runtime.Caller(0)
	projectRootDir := filepath.Dir(filepath.Dir(thisFile))
	tmpDir := filepath.Join(projectRootDir, "_tmp")
	artifactPath := filepath.Join(tmpDir, relPath)
	if dirExists(artifactPath) {
		return artifactPath
	}

	require.NoError(t, os.RemoveAll(tmpDir))
	cmd := command.NewFactory(env.NewRepository()).Create("git", []string{"clone", "--depth", "1", sampleArtifactsGitURL, tmpDir}, nil)
	require.NoError(t, cmd.Run())

	return artifactPath
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
