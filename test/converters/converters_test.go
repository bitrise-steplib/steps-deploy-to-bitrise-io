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

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters/xcresult3"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
	"github.com/google/go-cmp/cmp"
)

func TestXCresult3Converters(t *testing.T) {
	log.SetEnableDebugLog(true)
	want := junit.XML{
		TestSuites: []junit.TestSuite{
			{ // unit test
				Name:     "rtgtrghtrgTests",
				Tests:    2,
				Failures: 0,
				Errors:   0,
				Time:     0.26388001441955566,
				TestCases: []junit.TestCase{
					{ // plain test case
						Name:      "testExample()",
						ClassName: "rtgtrghtrgTests",
						Time:      0.0006339550018310547,
					},
					{ // plain test case
						Name:      "testPerformanceExample()",
						ClassName: "rtgtrghtrgTests",
						Time:      0.2632460594177246,
					},
				},
			},
			{ // ui test
				Name:     "rtgtrghtrgUITests",
				Tests:    15,
				Failures: 3,
				Errors:   0,
				Time:     0.7596049308776855,
				TestCases: []junit.TestCase{
					// class rtgtrghtrg3UITests: XCTestCase inside rtgtrghtrgUITests class
					{
						Name:      "testExample",
						ClassName: "_TtCC17rtgtrghtrgUITests17rtgtrghtrgUITests18rtgtrghtrg3UITests",
						Time:      0.031695008277893066,
					},
					{
						Name:      "testFailMe",
						ClassName: "_TtCC17rtgtrghtrgUITests17rtgtrghtrgUITests18rtgtrghtrg3UITests",
						Time:      0.09049093723297119,
						Failure: &junit.Failure{
							Value: "file:///Users/tamaspapik/Develop/ios/rtgtrghtrg/rtgtrghtrgUITests/rtgtrghtrgUITests.swift:CharacterRangeLen=0&EndingLineNumber=67&StartingLineNumber=67 - XCTAssertTrue failed",
						},
					},
					{
						Name:      "testLaunchPerformance",
						ClassName: "_TtCC17rtgtrghtrgUITests17rtgtrghtrgUITests18rtgtrghtrg3UITests",
						Time:      0.036438941955566406,
					},

					// class rtgtrghtrg2UITests: XCTestCase
					{
						Name:      "testExample()",
						ClassName: "rtgtrghtrg2UITests",
						Time:      0.06093001365661621,
					},
					{
						Name:      "testFailMe()",
						ClassName: "rtgtrghtrg2UITests",
						Time:      0.08525991439819336,
						Failure: &junit.Failure{
							Value: "file:///Users/tamaspapik/Develop/ios/rtgtrghtrg/rtgtrghtrgUITests/rtgtrghtrgUITests.swift:CharacterRangeLen=0&EndingLineNumber=104&StartingLineNumber=104 - XCTAssertTrue failed",
						},
					},
					{
						Name:      "testLaunchPerformance()",
						ClassName: "rtgtrghtrg2UITests",
						Time:      0.041545987129211426,
					},

					// class rtgtrghtrg4UITests: rtgtrghtrgUITests (so rtgtrghtrg4UITests inherits rtgtrghtrgUITests -> test cases merged and the base class name is rtgtrghtrg4UITests)
					{
						Name:      "testExample()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.07126104831695557,
					},
					{
						Name:      "testExample2()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.043392062187194824,
					},
					{
						Name:      "testFailMe()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.04290807247161865,
					},
					{
						Name:      "testFailMe2()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.08395206928253174,
						Failure: &junit.Failure{
							Value: "file:///Users/tamaspapik/Develop/ios/rtgtrghtrg/rtgtrghtrgUITests/rtgtrghtrgUITests.swift:CharacterRangeLen=0&EndingLineNumber=129&StartingLineNumber=129 - XCTAssertTrue failed",
						},
					},
					{
						Name:      "testLaunchPerformance()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.048184990882873535,
					},
					{
						Name:      "testLaunchPerformance2()",
						ClassName: "rtgtrghtrg4UITests",
						Time:      0.030627012252807617,
					},

					// class rtgtrghtrgUITests: XCTestCase
					{
						Name:      "testExample()",
						ClassName: "rtgtrghtrgUITests",
						Time:      0.030849933624267578,
					},
					{
						Name:      "testFailMe()",
						ClassName: "rtgtrghtrgUITests",
						Time:      0.030683040618896484,
					},
					{
						Name:      "testLaunchPerformance()",
						ClassName: "rtgtrghtrgUITests",
						Time:      0.03138589859008789,
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
		wantXML       junit.XML
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
				cmp.Transformer("SortTestSuites", func(in []junit.TestSuite) []junit.TestSuite {
					s := append([]junit.TestSuite{}, in...)
					sort.Slice(s, func(i, j int) bool {
						return s[i].Time > s[j].Time
					})
					return s
				}),
				cmp.Transformer("SortTestCases", func(in []junit.TestCase) []junit.TestCase {
					s := append([]junit.TestCase{}, in...)
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
