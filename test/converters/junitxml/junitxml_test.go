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
			name: "regroup error message",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Error: &Error{
						Message: "error message",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "Error message:\nerror message",
					},
				},
			}}},
		},
		{
			name: "regroup error body",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Error: &Error{
						Value: "error message",
					},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "Error value:\nerror message",
					},
				},
			}}},
		},
		{
			name: "regroup system err",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					SystemErr: "error message",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "System error:\nerror message",
					},
				},
			}}},
		},

		{
			name: "regroup error message - multiple test cases",
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
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "Error message:\nerror message",
					},
				},
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "Error message:\nerror message2",
					},
				},
			}}},
		},
		{
			name: "regroup error body - multiple test cases",
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
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "Error value:\nerror message",
					},
				},
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "Error value:\nerror message2",
					},
				},
			}}},
		},
		{
			name: "regroup system err - multiple test cases",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					SystemErr: "error message",
				},
				{
					SystemErr: "error message2",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 2, Failures: 2, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "System error:\nerror message",
					},
				},
				{
					Failure: &testreport.Failure{
						XMLName: xml.Name{Local: "failure"},
						Value:   "System error:\nerror message2",
					},
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
			name: "should append error body to failure",
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
						Value:   "failure message\n\nError value:\nerror value",
					},
				},
			}}},
		},
		{
			name: "should append error message to failure",
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
						Value:   "Failure message\n\nError message:\nerror value",
					},
				},
			}}},
		},
		{
			name: "should append system error to failure",
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
						Value:   "failure message\n\nSystem error:\nerror value",
					},
				},
			}}},
		},
		{
			name: "should append system error, error message, error body to failure",
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
						Value:   "failure message\n\nSystem error:\nerror value",
					},
				},
			}}},
		},
		{
			name: "should append system error, error message, error body to failure",
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
						Value:   "failure message\n\nfailure content\n\nError message:\nmessage\n\nError value:\nvalue\n\nSystem error:\nerror value",
					},
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
							testreport.TestCase{
								XMLName:   xml.Name{Local: "testcase"},
								Name:      "testPaymentSuccessShowsTooltip()",
								ClassName: "PaymentContextTests",
								Time:      0.19384193420410156,
								Failure:   nil,
							},
							testreport.TestCase{
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
							testreport.TestCase{
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
			name:    "Error message in Message attribute of Failure element",
			results: []resultReader{&fileReader{Filename: "./testdata/flaky_test.xml"}},
			want: testreport.TestReport{
				XMLName: xml.Name{Space: "", Local: ""},
				TestSuites: []testreport.TestSuite{
					{
						XMLName: xml.Name{Space: "", Local: "testsuite"},
						Name:    "My Test Suite",
						Tests:   7, Failures: 6, Skipped: 1,
						Time: 28.844,
						TestCases: []testreport.TestCase{
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 1", ClassName: "example.exampleTest", Time: 0.764,
								Failure: nil,
								Skipped: &testreport.Skipped{XMLName: xml.Name{Space: "", Local: "skipped"}}},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 2", ClassName: "example.exampleTest", Time: 0.164,
								Failure: &testreport.Failure{XMLName: xml.Name{Space: "", Local: "failure"}, Value: "System error:\nSome error message 2"},
								Skipped: nil},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0.445,
								Failure: &testreport.Failure{XMLName: xml.Name{Space: "", Local: "failure"}, Value: "Failure message\n\nError value:\nError\n\nSystem error:\nSome error message 3"},
								Skipped: nil},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0,
								Failure: &testreport.Failure{XMLName: xml.Name{Space: "", Local: "failure"}, Value: "System error:\nFlaky failure system error"},
								Skipped: nil},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0,
								Failure: &testreport.Failure{XMLName: xml.Name{Space: "", Local: "failure"}, Value: "System error:\nFlaky error system error"},
								Skipped: nil},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0,
								Failure: &testreport.Failure{XMLName: xml.Name{Space: "", Local: "failure"}, Value: "System error:\nRerun failure system error"},
								Skipped: nil},
							{XMLName: xml.Name{Space: "", Local: "testcase"}, ConfigurationHash: "", Name: "Testcase number 3", ClassName: "example.exampleTest", Time: 0,
								Failure: &testreport.Failure{XMLName: xml.Name{Space: "", Local: "failure"}, Value: "System error:\nRerun error system error"},
								Skipped: nil},
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
