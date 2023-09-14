package fileredactor

import (
	"os"
	"path"
	"testing"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RedactFiles(t *testing.T) {
	secrets := []string{
		"SUPER_SECRET_WORD",
		"ANOTHER_SECRET_WORD",
	}
	filePath := path.Join(t.TempDir(), "step-output.txt")
	content, err := os.ReadFile("testdata/before_redaction.txt")
	require.NoError(t, err)

	fileManager := fileutil.NewFileManager()
	err = fileManager.WriteBytes(filePath, content)
	require.NoError(t, err)

	fileRedactor := NewFileRedactor(fileutil.NewFileManager())
	err = fileRedactor.RedactFiles([]string{filePath}, secrets)
	require.NoError(t, err)

	got, err := os.ReadFile(filePath)
	require.NoError(t, err)

	want, err := os.ReadFile("testdata/after_redaction.txt")
	require.NoError(t, err)

	assert.Equal(t, want, got)
}
