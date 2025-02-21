package main

import (
	"html/template"
	"slices"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_generateUrlOutputWithTemplate(t *testing.T) {
	defaultTemplate := "{{range $index, $element := .}}{{if $index}}|{{end}}{{$element.File}}=>{{$element.URL}}{{end}}"
	temp := template.New("test")
	temp, err := temp.Parse(defaultTemplate)
	if err != nil {
		t.Errorf("error during parsing: %s", err)
	}
	tests := []struct {
		name         string
		pages        []PublicInstallPage
		maxEnvLength int
		want         string
		wantWarn     bool
	}{
		{
			name:         "Empty list gives empty value",
			pages:        []PublicInstallPage{},
			maxEnvLength: 100,
			want:         "",
		},
		{
			name: "All content fits the variable",
			pages: []PublicInstallPage{
				{
					File: "Foo",
					URL:  "Bar",
				},
			},
			maxEnvLength: 100,
			want:         "Foo=>Bar",
		},
		{
			name: "One item doesn't fit",
			pages: []PublicInstallPage{
				{
					File: "Foo",
					URL:  "Bar",
				},
				{
					File: "Baz",
					URL:  "Qux",
				},
			},
			maxEnvLength: 10,
			want:         "Foo=>Bar",
			wantWarn:     true,
		},
		{
			name: "Multiple items doesn't fit",
			pages: []PublicInstallPage{
				{
					File: "Foo",
					URL:  "Bar",
				},
				{
					File: "Baz",
					URL:  "Qux",
				},
				{
					File: "Apple",
					URL:  "Pear",
				},
				{
					File: "Peach",
					URL:  "Grapes",
				},
			},
			maxEnvLength: 20,
			want:         "Foo=>Bar|Baz=>Qux",
			wantWarn:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotWarn, err := applyTemplateWithMaxSize(temp, tt.pages, tt.maxEnvLength)
			if err != nil {
				t.Errorf("applyTemplateWithMaxSize() error: %s", err)
			}
			if gotWarn != tt.wantWarn {
				t.Errorf("applyTemplateWithMaxSize() warning = %v, want %v", gotWarn, tt.wantWarn)
			}
			if got != tt.want {
				t.Errorf("applyTemplateWithMaxSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadConcurrency(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   int
	}{
		{
			name: "Zero value",
			config: Config{
				UploadConcurrency: "0",
			},
			want: 1,
		},
		{
			name: "Negative value",
			config: Config{
				UploadConcurrency: "-1",
			},
			want: 1,
		},
		{
			name: "In range value",
			config: Config{
				UploadConcurrency: "3",
			},
			want: 3,
		},
		{
			name: "Too large value",
			config: Config{
				UploadConcurrency: "100",
			},
			want: 20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, determineConcurrency(tt.config))
		})
	}
}

func Test_validateUserGroups(t *testing.T) {
	tests := []struct {
		name          string
		userGroupsStr string
		logger        func() log.Logger
		wantErr       error
	}{
		{
			name:          "Empty user groups",
			userGroupsStr: "",
			logger:        func() log.Logger { return mocks.NewLogger(t) },
		},
		{
			name:          "Valid user groups",
			userGroupsStr: strings.Join(validUserGroups, ","),
			logger:        func() log.Logger { return mocks.NewLogger(t) },
		},
		{
			name:          "Valid user groups with capital letter",
			userGroupsStr: "Testers",
			logger: func() log.Logger {
				logger := mocks.NewLogger(t)
				logger.On("Warnf", "User group %s is accepted by the backend, but it is not the recommended value. Please use one of the following values: %s", "Testers", strings.Join(validUserGroups, ", "))
				return logger
			},
		},
		{
			name:          "Accepted user groups",
			userGroupsStr: strings.Join(acceptedUserGroups, ","),
			logger: func() log.Logger {
				logger := mocks.NewLogger(t)
				// Expect a warning for each accepted but not valid user group
				for _, userGroup := range acceptedUserGroups {
					if !slices.Contains(validUserGroups, userGroup) {
						logger.On("Warnf", "User group %s is accepted by the backend, but it is not the recommended value. Please use one of the following values: %s", userGroup, strings.Join(validUserGroups, ", "))
					}
				}
				return logger
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr == nil {
				require.NoError(t, validateUserGroups(tt.userGroupsStr, tt.logger()))
			} else {
				require.EqualError(t, validateUserGroups(tt.userGroupsStr, tt.logger()), tt.wantErr.Error())
			}
		})
	}
}
