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
		// Passing test cases - no failure, error, or skipped
		{
			name: "Passing test case - no details",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:      "passingTest",
					ClassName: "TestClass",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 0, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:      "passingTest",
					ClassName: "TestClass",
				},
			}}},
		},

		{
			name: "Passing test case - system error and output",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:      "passingTestWithOutput",
					ClassName: "TestClass",
					SystemOut: "Test passed\nExecution time: 100ms",
					SystemErr: "Info: deprecated method used",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 0, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:      "passingTestWithOutput",
					ClassName: "TestClass",
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "Test passed\nExecution time: 100ms"},
					SystemErr: &testreport.SystemErr{XMLName: xml.Name{Local: "system-err"}, Value: "Info: deprecated method used"},
				},
			}}},
		},

		// Failure test cases
		{
			name: "Failure - no details",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:    "failureNoDetailsTest",
					Failure: &Failure{},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:    "failureNoDetailsTest",
					Failure: &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: ""},
				},
			}}},
		},

		{
			name: "Failure - system error and output only",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:      "failureSystemOutputTest",
					Failure:   &Failure{},
					SystemOut: "Test execution started",
					SystemErr: "Error: NullPointerException",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:      "failureSystemOutputTest",
					Failure:   &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "System error:\nError: NullPointerException\n\nSystem output:\nTest execution started"},
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "Test execution started"},
					SystemErr: &testreport.SystemErr{XMLName: xml.Name{Local: "system-err"}, Value: "Error: NullPointerException"},
				},
			}}},
		},

		{
			name: "Failure - content only",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:    "simpleFailureTest",
					Failure: &Failure{Value: "assertion failed"},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:    "simpleFailureTest",
					Failure: &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "assertion failed"},
				},
			}}},
		},

		{
			name: "Failure - all attributes and system output",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:      "complexFailureTest",
					Failure:   &Failure{Type: "AssertionError", Message: "expected true", Value: "Stack trace"},
					SystemOut: "Test output: started",
					SystemErr: "Warning: deprecated API",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:      "complexFailureTest",
					Failure:   &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "AssertionError: expected true\n\nStack trace\n\nSystem error:\nWarning: deprecated API\n\nSystem output:\nTest output: started"},
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "Test output: started"},
					SystemErr: &testreport.SystemErr{XMLName: xml.Name{Local: "system-err"}, Value: "Warning: deprecated API"},
				},
			}}},
		},

		// Error test cases
		{
			name: "Error - no details",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:  "errorNoDetailsTest",
					Error: &Error{},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:    "errorNoDetailsTest",
					Failure: &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: ""},
				},
			}}},
		},

		{
			name: "Error - system error and output only",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:      "errorSystemOutputTest",
					Error:     &Error{},
					SystemOut: "Initialization complete",
					SystemErr: "Critical error occurred",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:      "errorSystemOutputTest",
					Failure:   &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "System error:\nCritical error occurred\n\nSystem output:\nInitialization complete"},
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "Initialization complete"},
					SystemErr: &testreport.SystemErr{XMLName: xml.Name{Local: "system-err"}, Value: "Critical error occurred"},
				},
			}}},
		},

		{
			name: "Error - content only",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:  "simpleErrorTest",
					Error: &Error{Value: "null pointer"},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:    "simpleErrorTest",
					Failure: &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "null pointer"},
				},
			}}},
		},

		{
			name: "Error - all attributes and system output",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:      "complexErrorTest",
					Failure:   &Failure{Type: "NullPointerException", Message: "object was null", Value: "Full stack trace"},
					SystemOut: "Test setup complete",
					SystemErr: "System error: OutOfMemory",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 1, Errors: 0, Skipped: 0, TestCases: []testreport.TestCase{
				{
					Name:      "complexErrorTest",
					Failure:   &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "NullPointerException: object was null\n\nFull stack trace\n\nSystem error:\nSystem error: OutOfMemory\n\nSystem output:\nTest setup complete"},
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "Test setup complete"},
					SystemErr: &testreport.SystemErr{XMLName: xml.Name{Local: "system-err"}, Value: "System error: OutOfMemory"},
				},
			}}},
		},

		// Skipped test cases
		{
			name: "Skipped - no details",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:    "skippedNoDetailsTest",
					Skipped: &Skipped{},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 0, Errors: 0, Skipped: 1, TestCases: []testreport.TestCase{
				{
					Name:    "skippedNoDetailsTest",
					Skipped: &testreport.Skipped{XMLName: xml.Name{Local: "skipped"}, Value: ""},
				},
			}}},
		},

		{
			name: "Skipped - system error and output only",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:      "skippedSystemOutputTest",
					Skipped:   &Skipped{},
					SystemOut: "Skipping test due to unmet prerequisites",
					SystemErr: "Warning: missing configuration",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 0, Errors: 0, Skipped: 1, TestCases: []testreport.TestCase{
				{
					Name:      "skippedSystemOutputTest",
					Skipped:   &testreport.Skipped{XMLName: xml.Name{Local: "skipped"}, Value: "System error:\nWarning: missing configuration\n\nSystem output:\nSkipping test due to unmet prerequisites"},
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "Skipping test due to unmet prerequisites"},
					SystemErr: &testreport.SystemErr{XMLName: xml.Name{Local: "system-err"}, Value: "Warning: missing configuration"},
				},
			}}},
		},

		{
			name: "Skipped - content only",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:    "simpleSkippedTest",
					Skipped: &Skipped{Message: "Test not implemented"},
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 0, Errors: 0, Skipped: 1, TestCases: []testreport.TestCase{
				{
					Name:    "simpleSkippedTest",
					Skipped: &testreport.Skipped{XMLName: xml.Name{Local: "skipped"}, Value: "Test not implemented"},
				},
			}}},
		},

		{
			name: "Skipped - all attributes and system output",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:      "complexSkippedTest",
					Skipped:   &Skipped{Message: "Environment not ready"},
					SystemOut: "Setup started but aborted",
					SystemErr: "Resource pool unavailable",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 1, Failures: 0, Errors: 0, Skipped: 1, TestCases: []testreport.TestCase{
				{
					Name:      "complexSkippedTest",
					Skipped:   &testreport.Skipped{XMLName: xml.Name{Local: "skipped"}, Value: "Environment not ready\n\nSystem error:\nResource pool unavailable\n\nSystem output:\nSetup started but aborted"},
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "Setup started but aborted"},
					SystemErr: &testreport.SystemErr{XMLName: xml.Name{Local: "system-err"}, Value: "Resource pool unavailable"},
				},
			}}},
		},

		// Multiple test cases in one suite
		{
			name: "Multiple test cases with mixed results",
			suites: []TestSuite{{TestCases: []TestCase{
				{
					Name:      "passingTest",
					ClassName: "TestClass",
				},
				{
					Name:    "failureTest",
					Failure: &Failure{Value: "expected true"},
				},
				{
					Name:  "errorTest",
					Error: &Error{Value: "null pointer"},
				},
				{
					Name:    "skippedTest",
					Skipped: &Skipped{},
				},
				{
					Name:      "anotherPassingTest",
					ClassName: "TestClass",
					SystemOut: "All good",
				},
				{
					Name:      "anotherFailureTest",
					Failure:   &Failure{Type: "AssertionError", Message: "value mismatch", Value: "Stack trace"},
					SystemErr: "Error details",
				},
				{
					Name:      "anotherErrorTest",
					Error:     &Error{Type: "IOException", Message: "file not found", Value: "Stack trace"},
					SystemOut: "Attempted to read file",
					SystemErr: "File read error",
				},
				{
					Name:      "anotherSkippedTest",
					Skipped:   &Skipped{Message: "Dependency missing"},
					SystemOut: "Skipping due to missing dependency",
				},
			}}},
			want: []testreport.TestSuite{{Tests: 8, Failures: 4, Errors: 0, Skipped: 2, TestCases: []testreport.TestCase{
				{
					Name:      "passingTest",
					ClassName: "TestClass",
				},
				{
					Name:    "failureTest",
					Failure: &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "expected true"},
				},
				{
					Name:    "errorTest",
					Failure: &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "null pointer"},
				},
				{
					Name:    "skippedTest",
					Skipped: &testreport.Skipped{XMLName: xml.Name{Local: "skipped"}, Value: ""},
				},
				{
					Name:      "anotherPassingTest",
					ClassName: "TestClass",
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "All good"},
				},
				{
					Name:      "anotherFailureTest",
					Failure:   &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "AssertionError: value mismatch\n\nStack trace\n\nSystem error:\nError details"},
					SystemErr: &testreport.SystemErr{XMLName: xml.Name{Local: "system-err"}, Value: "Error details"},
				},
				{
					Name:      "anotherErrorTest",
					Failure:   &testreport.Failure{XMLName: xml.Name{Local: "failure"}, Value: "IOException: file not found\n\nStack trace\n\nSystem error:\nFile read error\n\nSystem output:\nAttempted to read file"},
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "Attempted to read file"},
					SystemErr: &testreport.SystemErr{XMLName: xml.Name{Local: "system-err"}, Value: "File read error"},
				},
				{
					Name:      "anotherSkippedTest",
					Skipped:   &testreport.Skipped{XMLName: xml.Name{Local: "skipped"}, Value: "Dependency missing\n\nSystem output:\nSkipping due to missing dependency"},
					SystemOut: &testreport.SystemOut{XMLName: xml.Name{Local: "system-out"}, Value: "Skipping due to missing dependency"},
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
						Errors:   0,
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
									Value: "AssertionError: Assertion error message",
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
			name:    "Error message in Message attribute of Failure element",
			results: []resultReader{&fileReader{Filename: "./testdata/flaky_test.xml"}},
			want: testreport.TestReport{
				XMLName: xml.Name{Space: "", Local: ""},
				TestSuites: []testreport.TestSuite{
					{
						XMLName:  xml.Name{Space: "", Local: "testsuite"},
						Name:     "My Test Suite",
						Tests:    7,
						Failures: 5,
						Errors:   0,
						Skipped:  1,
						Time:     28.844,
						TestCases: []testreport.TestCase{
							{
								XMLName:           xml.Name{Space: "", Local: "testcase"},
								ConfigurationHash: "",
								Name:              "Testcase number 1",
								ClassName:         "example.exampleTest",
								Time:              0.764,
								Failure:           nil,
								Error:             nil,
								Skipped: &testreport.Skipped{
									XMLName: xml.Name{Space: "", Local: "skipped"},
									Value:   "System output:\n[INFO] 13:12:43:\n[INFO] 13:12:43:\tLog line 1 1\n[INFO] 13:12:43:\tLog line 2 1\n",
								},
								SystemOut: &testreport.SystemOut{
									XMLName: xml.Name{Space: "", Local: "system-out"},
									Value:   "[INFO] 13:12:43:\n[INFO] 13:12:43:\tLog line 1 1\n[INFO] 13:12:43:\tLog line 2 1\n",
								},
							},
							{
								XMLName:           xml.Name{Space: "", Local: "testcase"},
								ConfigurationHash: "",
								Name:              "Testcase number 2",
								ClassName:         "example.exampleTest",
								Time:              0.164,
								Failure:           nil,
								Skipped:           nil,
								Error:             nil,
								SystemOut: &testreport.SystemOut{
									XMLName: xml.Name{Space: "", Local: "system-out"},
									Value:   "[INFO] 13:12:43:\n[INFO] 13:12:43:\tLog line 1 2\n[INFO] 13:12:43:\tLog line 2 2\n",
								},
								SystemErr: &testreport.SystemErr{
									XMLName: xml.Name{Space: "", Local: "system-err"},
									Value:   "Some error message 2",
								},
							},
							{
								XMLName:           xml.Name{Space: "", Local: "testcase"},
								ConfigurationHash: "",
								Name:              "Testcase number 3",
								ClassName:         "example.exampleTest",
								Time:              0.445,
								Error:             nil,
								Skipped:           nil,
								Failure: &testreport.Failure{
									XMLName: xml.Name{Space: "", Local: "failure"},
									Value:   "Error\n\nSystem error:\nSome error message 3\n\nSystem output:\n[INFO] 13:12:43:\n[INFO] 13:12:43:\tLog line 1 3\n[INFO] 13:12:43:\tLog line 2 3\n",
								},
								SystemOut: &testreport.SystemOut{
									XMLName: xml.Name{Space: "", Local: "system-out"},
									Value:   "[INFO] 13:12:43:\n[INFO] 13:12:43:\tLog line 1 3\n[INFO] 13:12:43:\tLog line 2 3\n",
								},
								SystemErr: &testreport.SystemErr{
									XMLName: xml.Name{Space: "", Local: "system-err"},
									Value:   "Some error message 3",
								},
							},
							{
								XMLName:           xml.Name{Space: "", Local: "testcase"},
								ConfigurationHash: "",
								Name:              "Testcase number 3",
								ClassName:         "example.exampleTest",
								Time:              0,
								Skipped:           nil,
								Error:             nil,
								Failure: &testreport.Failure{
									XMLName: xml.Name{Space: "", Local: "failure"},
									Value:   "System error:\nFlaky failure system error",
								},
							},
							{
								XMLName:           xml.Name{Space: "", Local: "testcase"},
								ConfigurationHash: "",
								Name:              "Testcase number 3",
								ClassName:         "example.exampleTest",
								Time:              0,
								Skipped:           nil,
								Error:             nil,
								Failure: &testreport.Failure{
									XMLName: xml.Name{Space: "", Local: "failure"},
									Value:   "System error:\nFlaky error system error",
								},
							},
							{
								XMLName:           xml.Name{Space: "", Local: "testcase"},
								ConfigurationHash: "",
								Name:              "Testcase number 3",
								ClassName:         "example.exampleTest",
								Time:              0,
								Skipped:           nil,
								Error:             nil,
								Failure: &testreport.Failure{
									XMLName: xml.Name{Space: "", Local: "failure"},
									Value:   "System error:\nRerun failure system error",
								},
							},
							{
								XMLName:           xml.Name{Space: "", Local: "testcase"},
								ConfigurationHash: "",
								Name:              "Testcase number 3",
								ClassName:         "example.exampleTest",
								Time:              0,
								Skipped:           nil,
								Error:             nil,
								Failure: &testreport.Failure{
									XMLName: xml.Name{Space: "", Local: "failure"},
									Value:   "System error:\nRerun error system error",
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
				require.EqualValues(t, tt.want, got)
			}
		})
	}
}
