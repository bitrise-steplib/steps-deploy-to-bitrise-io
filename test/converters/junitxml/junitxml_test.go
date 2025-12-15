package junitxml

import (
	"encoding/xml"
	"testing"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
	"github.com/stretchr/testify/require"
)

func Test_parseTestReport(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "root element is testsuites", path: "./testdata/testsuites.xml", wantErr: false},
		{name: "root element is another testsuites", path: "./testdata/testsuites.junit", wantErr: false},
		{name: "root element is testsuite", path: "./testdata/testsuite.xml", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTestReport(&fileReader{Filename: tt.path})
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTestReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_convertTestReport(t *testing.T) {
	tests := []struct {
		name   string
		suites []TestSuite
		want   []testreport.TestSuite
	}{
		{
			name: "convert error message",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Error: &Error{
						Message: "error message",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Error: &testreport.Error{
						XMLName: xml.Name{Local: "error"},
						Value:   "error message",
					},
				},
			}}},
		},
		{
			name: "convert error body",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Error: &Error{
						Value: "error message",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Error: &testreport.Error{
						XMLName: xml.Name{Local: "error"},
						Value:   "error message",
					},
				},
			}}},
		},
		{
			name: "preserve system err",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					SystemErr: "error message",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					SystemErr: "error message",
				},
			}}},
		},

		{
			name: "convert error message - multiple test cases",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Error: &Error{
						Message: "error message",
					},
				},
				{
					Error: &Error{
						Message: "error message2",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 2, Failures: 2, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Error: &testreport.Error{
						XMLName: xml.Name{Local: "error"},
						Value:   "error message",
					},
				},
				{
					Error: &testreport.Error{
						XMLName: xml.Name{Local: "error"},
						Value:   "error message2",
					},
				},
			}}},
		},
		{
			name: "convert error body - multiple test cases",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Error: &Error{
						Value: "error message",
					},
				},
				{
					Error: &Error{
						Value: "error message2",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 2, Failures: 2, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Error: &testreport.Error{
						XMLName: xml.Name{Local: "error"},
						Value:   "error message",
					},
				},
				{
					Error: &testreport.Error{
						XMLName: xml.Name{Local: "error"},
						Value:   "error message2",
					},
				},
			}}},
		},
		{
			name: "preserve system err - multiple test cases",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					SystemErr: "error message",
				},
				{
					SystemErr: "error message2",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 2, Failures: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					SystemErr: "error message",
				},
				{
					SystemErr: "error message2",
				},
			}}},
		},
		{
			name: "should not touch failure",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Failure: &Failure{
						Value: "error message",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "error message",
					},
				},
			}}},
		},
		{
			name: "should keep error separate from failure",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Failure: &Failure{
						Value: "failure message",
					},
					Error: &Error{
						Value: "error value",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "failure message",
					},
					Error: &testreport.Error{
						XMLName: xml.Name{Local: "error"},
						Value:   "error value",
					},
				},
			}}},
		},
		{
			name: "should keep error message separate from failure",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Failure: &Failure{
						Value: "Failure message",
					},
					Error: &Error{
						Message: "error value",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "Failure message",
					},
					Error: &testreport.Error{
						XMLName: xml.Name{Local: "error"},
						Value:   "error value",
					},
				},
			}}},
		},
		{
			name: "should preserve system error separately from failure",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Failure: &Failure{
						Value: "failure message",
					},
					SystemErr: "error value",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "failure message",
					},
					SystemErr: "error value",
				},
			}}},
		},
		{
			name: "should preserve system error separately from failure (duplicate test)",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Failure: &Failure{
						Value: "failure message",
					},
					SystemErr: "error value",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "failure message",
					},
					SystemErr: "error value",
				},
			}}},
		},
		{
			name: "should keep all fields separate",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Failure: &Failure{
						Message: "failure message",
						Value:   "failure content",
					},
					SystemErr: "error value",
					Error: &Error{
						Message: "message",
						Value:   "value",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "failure message\n\nfailure content",
					},
					Error: &testreport.Error{
						XMLName: xml.Name{Local: "error"},
						Value:   "message\n\nvalue",
					},
					SystemErr: "error value",
				},
			}}},
		},
		{
			name: "Should convert Message attribute of Failure element to the value of a Failure element",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Failure: &Failure{
						Message: "ErrorMsg",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "ErrorMsg",
					},
				},
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTestReport(TestReport{TestSuites: tt.suites})
			require.Equal(t, testreport.TestReport{TestSuites: tt.want}, got)
		})
	}
}

func TestConverter_Convert(t *testing.T) {
	tests := []struct {
		name    string
		results []resultReader
		want    testreport.TestReport
		wantErr bool
	}{
		{
			name: "Error message in Message attribute of Failure element",
			results: []resultReader{&stringReader{
				Contents: `<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="2" failures="1">
	<testsuite name="MyApp-Unit-Tests" tests="2" failures="0" time="0.398617148399353">
		<testcase classname="PaymentContextTests" name="testPaymentSuccessShowsTooltip()" time="0.19384193420410156">
		</testcase>
		<testcase classname="PaymentContextTests" name="testCannotCheckoutIfPaymentIsActive()" time="0.17543494701385498">
			<failure message="XCTAssertTrue failed">
			</failure>
		</testcase>
	</testsuite>
</testsuites>`,
			}},
			want: testreport.TestReport{
				TestSuites: []testreport.TestSuite{
					{
						XMLName:  xml.Name{Local: "testsuite"},
						Name:     "MyApp-Unit-Tests",
						Tests:    2,
						Failures: 1,
						Time:     0.398617148399353,
						TestCases: []testreport.TestCase{
							{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testPaymentSuccessShowsTooltip()",
								ClassName: "PaymentContextTests",
								Time:      0.19384193420410156,
								Failure:   nil,
							},
							{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testCannotCheckoutIfPaymentIsActive()",
								ClassName: "PaymentContextTests",
								Time:      0.17543494701385498,
								Failure: &testreport.Failure{
									XMLName: xml.Name{Local: "failure"},
									Value:   "XCTAssertTrue failed",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Test properties are converted",
			results: []resultReader{&stringReader{
				Contents: `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
 <testsuite name="BitriseBasicUITest" tests="1" failures="0" errors="0" time="10">
  <testcase name="testCollectionView()" classname="BitriseBasicUITest" time="10">
   <properties>
    <property name="attachment_1" value="some_attachment.png" />
   </properties>
  </testcase>
 </testsuite>
</testsuites>`,
			}},
			want: testreport.TestReport{
				TestSuites: []testreport.TestSuite{
					{
						XMLName:  xml.Name{Local: "testsuite"},
						Name:     "BitriseBasicUITest",
						Tests:    1,
						Failures: 0,
						Time:     10,
						TestCases: []testreport.TestCase{
							{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testCollectionView()",
								ClassName: "BitriseBasicUITest",
								Time:      10,
								Failure:   nil,
								Properties: &testreport.Properties{
									XMLName: xml.Name{Local: "properties"},
									Property: []testreport.Property{
										{
											XMLName: xml.Name{Local: "property"},
											Name:    "attachment_1",
											Value:   "some_attachment.png",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Tests in nested test suites are converted",
			results: []resultReader{&stringReader{
				Contents: `<?xml version="1.0" encoding="UTF-8"?>
<!--
This is a basic JUnit-style XML example to highlight the basis structure.

Example by Testmo. Copyright 2023 Testmo GmbH. All rights reserved.
Testmo test management software - https://www.testmo.com/
-->
<testsuites time="15.682687">
    <testsuite name="Tests.Registration" time="6.605871">
        <testcase name="testCase1" classname="Tests.Registration" time="2.113871" />
        <testcase name="testCase2" classname="Tests.Registration" time="1.051" />
        <testcase name="testCase3" classname="Tests.Registration" time="3.441" />
    </testsuite>
    <testsuite name="Tests.Authentication" time="9.076816">
        <testsuite name="Tests.Authentication.Login" time="4.356">
            <testcase name="testCase4" classname="Tests.Authentication.Login" time="2.244" />
            <testcase name="testCase5" classname="Tests.Authentication.Login" time="0.781" />
            <testcase name="testCase6" classname="Tests.Authentication.Login" time="1.331" />
        </testsuite>
        <testcase name="testCase7" classname="Tests.Authentication" time="2.508" />
        <testcase name="testCase8" classname="Tests.Authentication" time="1.230816" />
        <testcase name="testCase9" classname="Tests.Authentication" time="0.982">
            <failure message="Assertion error message" type="AssertionError">
                <!-- Call stack printed here -->
            </failure>
        </testcase>
    </testsuite>
</testsuites>`,
			}},
			want: testreport.TestReport{
				TestSuites: []testreport.TestSuite{
					{
						XMLName:  xml.Name{Local: "testsuite"},
						Name:     "Tests.Registration",
						Tests:    3,
						Failures: 0,
						Time:     6.605871,
						TestCases: []testreport.TestCase{
							{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testCase1",
								ClassName: "Tests.Registration",
								Time:      2.113871,
							},
							{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testCase2",
								ClassName: "Tests.Registration",
								Time:      1.051,
							},
							{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testCase3",
								ClassName: "Tests.Registration",
								Time:      3.441,
							},
						},
					},
					{
						XMLName:  xml.Name{Local: "testsuite"},
						Name:     "Tests.Authentication",
						Tests:    3,
						Failures: 1,
						Time:     9.076816,
						TestCases: []testreport.TestCase{
							{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testCase7",
								ClassName: "Tests.Authentication",
								Time:      2.508,
							},
							{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testCase8",
								ClassName: "Tests.Authentication",
								Time:      1.230816,
							},
							{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testCase9",
								ClassName: "Tests.Authentication",
								Time:      0.982,
								Failure: &testreport.Failure{
									XMLName: xml.Name{
										Local: "failure",
									},
									Value: "Assertion error message",
								},
							},
						},
						TestSuites: []testreport.TestSuite{
							{
								XMLName:  xml.Name{Local: "testsuite"},
								Name:     "Tests.Authentication.Login",
								Tests:    3,
								Failures: 0,
								Time:     4.356,
								TestCases: []testreport.TestCase{
									{
										XMLName:   xml.Name{Local: "testcase"},
										Name:      "testCase4",
										ClassName: "Tests.Authentication.Login",
										Time:      2.244,
									},
									{
										XMLName:   xml.Name{Local: "testcase"},
										Name:      "testCase5",
										ClassName: "Tests.Authentication.Login",
										Time:      0.781,
									},
									{
										XMLName:   xml.Name{Local: "testcase"},
										Name:      "testCase6",
										ClassName: "Tests.Authentication.Login",
										Time:      1.331,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Converter{
				results: tt.results,
			}
			got, err := h.Convert()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestConverter_Convert_Grouped_report(t *testing.T) {
	tests := []struct {
		name    string
		results []resultReader
		want    testreport.TestReport
		wantErr bool
	}{
		{
			name:    "Flaky test with separate error and system error fields",
			results: []resultReader{&fileReader{Filename: "./testdata/flaky_test.xml"}},
			want: testreport.TestReport{
				XMLName: xml.Name{Space: "", Local: ""},
				TestSuites: []testreport.TestSuite{
					{
						XMLName: xml.Name{Space: "", Local: "testsuite"},
						Name:    "My Test Suite",
						Tests:   7, Failures: 1, Skipped: 1,
						Time: 28.844,
						TestCases: []testreport.TestCase{
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 1", ClassName: "example.exampleTest", Time: 0.764,
								Failure:   nil,
								Error:     nil,
								Skipped:   &testreport.Skipped{XMLName: xml.Name{Space: "", Local: "skipped"}},
								SystemOut: "[INFO] 13:12:43:\n[INFO] 13:12:43:\tLog line 1 1\n[INFO] 13:12:43:\tLog line 2 1\n"},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 2", ClassName: "example.exampleTest", Time: 0.164,
								Failure:   nil,
								Error:     nil,
								Skipped:   nil,
								SystemErr: "Some error message 2",
								SystemOut: "[INFO] 13:12:43:\n[INFO] 13:12:43:\tLog line 1 2\n[INFO] 13:12:43:\tLog line 2 2\n"},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0.445,
								Failure:   &testreport.Failure{XMLName: xml.Name{Space: "", Local: "failure"}, Value: "Failure message"},
								Error:     &testreport.Error{XMLName: xml.Name{Space: "", Local: "error"}, Value: "Error"},
								Skipped:   nil,
								SystemErr: "Some error message 3",
								SystemOut: "[INFO] 13:12:43:\n[INFO] 13:12:43:\tLog line 1 3\n[INFO] 13:12:43:\tLog line 2 3\n"},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0,
								Failure:   nil,
								Error:     nil,
								Skipped:   nil,
								SystemErr: "Flaky failure system error"},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0,
								Failure:   nil,
								Error:     nil,
								Skipped:   nil,
								SystemErr: "Flaky error system error"},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0,
								Failure:   nil,
								Error:     nil,
								Skipped:   nil,
								SystemErr: "Rerun failure system error"},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0,
								Failure:   nil,
								Error:     nil,
								Skipped:   nil,
								SystemErr: "Rerun error system error"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Converter{
				results: tt.results,
			}
			got, err := h.Convert()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.EqualValues(t, tt.want, got)
			}
		})
	}
}
