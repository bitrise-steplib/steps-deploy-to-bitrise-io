package main

import (
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
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
