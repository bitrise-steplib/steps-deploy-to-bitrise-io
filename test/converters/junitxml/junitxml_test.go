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
		{"regroup error message", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Error: &junit.Error{Message: "error message"}},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "Error message:\nerror message"},
				}},
			},
		},

		{"regroup error body", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Error: &junit.Error{Value: "error message"}},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "Error value:\nerror message"},
				}},
			},
		},

		{"regroup system err", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{SystemErr: "error message"},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "System error:\nerror message"},
				}},
			},
		},

		{"regroup error message - multiple test cases", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Error: &junit.Error{Message: "error message"}},
				{Error: &junit.Error{Message: "error message2"}},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "Error message:\nerror message"},
					{Failure: "Error message:\nerror message2"},
				}},
			},
		},

		{"regroup error body - multiple test cases", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Error: &junit.Error{Value: "error message"}},
				{Error: &junit.Error{Value: "error message2"}},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "Error value:\nerror message"},
					{Failure: "Error value:\nerror message2"},
				}},
			},
		},

		{"regroup system err - multiple test cases", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{SystemErr: "error message"},
				{SystemErr: "error message2"},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "System error:\nerror message"},
					{Failure: "System error:\nerror message2"},
				}},
			},
		},

		{"should not touch failure", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Failure: "error message"},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "error message"},
				}},
			},
		},

		{"should append error body to failure", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Failure: "failure message", Error: &junit.Error{Value: "error value"}},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "failure message\n\nError value:\nerror value"},
				}},
			},
		},

		{"should append error message to failure", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Failure: "failure message", Error: &junit.Error{Message: "error value"}},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "failure message\n\nError message:\nerror value"},
				}},
			},
		},

		{"should append system error to failure", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Failure: "failure message", SystemErr: "error value"},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "failure message\n\nSystem error:\nerror value"},
				}},
			},
		},
		{"should append system error, error message, error body to failure", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Failure: "failure message", SystemErr: "error value"},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "failure message\n\nSystem error:\nerror value"},
				}},
			},
		},
		{"should append system error, error message, error body to failure", []junit.TestSuite{
			{TestCases: []junit.TestCase{
				{Failure: "failure message", SystemErr: "error value", Error: &junit.Error{Message: "message", Value: "value"}},
			}},
		},
			[]junit.TestSuite{
				{TestCases: []junit.TestCase{
					{Failure: "failure message\n\nError message:\nmessage\n\nError value:\nvalue\n\nSystem error:\nerror value"},
				}},
			},
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
