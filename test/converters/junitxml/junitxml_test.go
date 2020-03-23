package junitxml

import (
	"reflect"
	"testing"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/junit"
)

func Test_parseTestSuites(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "root element is testsuites", path: "./testdata/testsuites.xml", wantErr: false},
		{name: "root element is testsuite", path: "./testdata/testsuite.xml", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTestSuites(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTestSuites() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_regroupErrors(t *testing.T) {
	tests := []struct {
		name   string
		suites []junit.TestSuite
		want   []junit.TestSuite
	}{
		{
			name: "regroup error message",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Error: &junit.Error{
						Message: "error message",
					},
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "Error message:\nerror message",
					},
				},
			}}},
		},
		{
			name: "regroup error body",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Error: &junit.Error{
						Value: "error message",
					},
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "Error value:\nerror message",
					},
				},
			}}},
		},
		{
			name: "regroup system err",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					SystemErr: "error message",
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "System error:\nerror message",
					},
				},
			}}},
		},

		{
			name: "regroup error message - multiple test cases",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Error: &junit.Error{
						Message: "error message",
					},
				},
				{
					Error: &junit.Error{
						Message: "error message2",
					},
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "Error message:\nerror message",
					},
				},
				{
					Failure: &junit.Failure{
						Value: "Error message:\nerror message2",
					},
				},
			}}},
		},
		{
			name: "regroup error body - multiple test cases",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Error: &junit.Error{
						Value: "error message",
					},
				},
				{
					Error: &junit.Error{
						Value: "error message2",
					},
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "Error value:\nerror message",
					},
				},
				{
					Failure: &junit.Failure{
						Value: "Error value:\nerror message2",
					},
				},
			}}},
		},
		{
			name: "regroup system err - multiple test cases",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					SystemErr: "error message",
				},
				{
					SystemErr: "error message2",
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "System error:\nerror message",
					},
				},
				{
					Failure: &junit.Failure{
						Value: "System error:\nerror message2",
					},
				},
			}}},
		},
		{
			name: "should not touch failure",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "error message",
					},
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "error message",
					},
				},
			}}},
		},
		{
			name: "should append error body to failure",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "failure message",
					},
					Error: &junit.Error{
						Value: "error value",
					},
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "failure message\n\nError value:\nerror value",
					},
				},
			}}},
		},
		{
			name: "should append error message to failure",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "Failure message",
					},
					Error: &junit.Error{
						Message: "error value",
					},
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "Failure message\n\nError message:\nerror value",
					},
				},
			}}},
		},
		{
			name: "should append system error to failure",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "failure message",
					},
					SystemErr: "error value",
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "failure message\n\nSystem error:\nerror value",
					},
				},
			}}},
		},
		{
			name: "should append system error, error message, error body to failure",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "failure message",
					},
					SystemErr: "error value",
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "failure message\n\nSystem error:\nerror value",
					},
				},
			}}},
		},
		{
			name: "should append system error, error message, error body to failure",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Message: "failure message",
						Value:   "failure content",
					},
					SystemErr: "error value",
					Error: &junit.Error{
						Message: "message",
						Value:   "value",
					},
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "failure message\n\nfailure content\n\nError message:\nmessage\n\nError value:\nvalue\n\nSystem error:\nerror value",
					},
				},
			}}},
		},
		{
			name: "Should convert Message attribute of Failure element to the value of a Failure element",
			suites: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Message: "ErrorMsg",
					},
				},
			}}},
			want: []junit.TestSuite{{TestCases: []junit.TestCase{
				{
					Failure: &junit.Failure{
						Value: "ErrorMsg",
					},
				},
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := regroupErrors(tt.suites); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("regroupErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}
