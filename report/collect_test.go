package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecialContentTypes(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected string
	}{
		{
			name:     "Plain text file",
			fileName: "abc.txt",
			expected: "text/plain; charset=utf-8",
		},
		{
			name:     "Javascript file",
			fileName: "abc.js",
			expected: "text/javascript; charset=utf-8",
		},
		{
			name:     "CSS file",
			fileName: "abc.css",
			expected: "text/css; charset=utf-8",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), test.fileName)
			err := os.WriteFile(path, []byte("hello world"), 0644)
			require.NoError(t, err)

			contentType := detectContentType(path)
			assert.Equal(t, test.expected, contentType)
		})
	}
}
