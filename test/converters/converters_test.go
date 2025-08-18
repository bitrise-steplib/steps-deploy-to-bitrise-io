// Package converters contains the interface that is required to be a package a test result converter.
// It must be possible to set files from outside(for example if someone wants to use
// a pre-filtered files list), need to return Junit4 xml test result, and needs to have a
// Detect method to see if the converter can run with the files included in the test result dictionary.
// (So a converter can run only if the dir has a TestSummaries.plist file for example)
package converters

import (
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters/xcresult3"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestXCresult3Converters(t *testing.T) {
	log.SetEnableDebugLog(true)
	want := testreport.TestReport{
		TestSuites: []testreport.TestSuite{
			{ // unit test
				Name:     "rtgtrghtrgTests",
				Tests:    2,
				Failures: 0,
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
	convertersPackageDir := filepath.Dir(b)
	testPackageDir := filepath.Dir(convertersPackageDir)
	projectRootDir := filepath.Dir(testPackageDir)

	for _, test := range []struct {
		name          string
		converter     Converter
		testFilePaths []string
		wantDetect    bool
		wantXML       testreport.TestReport
		wantXMLError  bool
	}{
		{
			name:          "xcresult3",
			converter:     &xcresult3.Converter{},
			testFilePaths: []string{filepath.Join(projectRootDir, "_tmp/xcresults/xcresult3_multi_level_UI_tests.xcresult")},
			wantDetect:    true,
			wantXMLError:  false,
			wantXML:       want,
		},
		{
			name:          "Long running test",
			converter:     &xcresult3.Converter{},
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
