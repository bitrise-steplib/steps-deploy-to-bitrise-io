// Package converters contains the interface that is required to be a package a test result converter.
// It must be possible to set files from outside(for example if someone wants to use
// a pre-filtered files list), need to return Junit4 xml test result, and needs to have a
// Detect method to see if the converter can run with the files included in the test result dictionary.
// (So a converter can run only if the dir has a TestSummaries.plist file for example)
package converters

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters/xcresult3"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
	"github.com/google/go-cmp/cmp"

	"github.com/bitrise-io/go-utils/log"
)

func TestXCresult3Converters(t *testing.T) {
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

	for _, test := range []struct {
		name          string
		converter     Intf
		testFilePaths []string
		wantDetect    bool
		wantXML       testreport.TestReport
		wantXMLError  bool
	}{
		{
			name:          "xcresult3",
			converter:     &xcresult3.Converter{},
			testFilePaths: []string{filepath.Join(os.Getenv("BITRISE_SOURCE_DIR"), "_tmp/xcresults/xcresult3_multi_level_UI_tests.xcresult")},
			wantDetect:    true,
			wantXMLError:  false,
			wantXML:       want,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.converter.Detect(test.testFilePaths); got != test.wantDetect {
				t.Fatalf("detect want: %v, got: %v", test.wantDetect, got)
			}

			got, err := test.converter.XML()
			if test.wantXMLError && err == nil {
				t.Fatalf("xml error want: %v, got: %v", test.wantXMLError, got)
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

			if !cmp.Equal(got, test.wantXML, opts...) {
				t.Fatalf("xml want: %+v, got: %+v", test.wantXML, got)
			}
		})
	}
}
