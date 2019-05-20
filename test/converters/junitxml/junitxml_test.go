package junitxml

import (
	"testing"
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
