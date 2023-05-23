package fileredactor

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/bitrise-io/go-utils/v2/fileutil"

	"github.com/stretchr/testify/assert"
)

func Test_RedactFiles(t *testing.T) {
	originalFile := "testdata/file_to_redact_original.txt"
	fileToRedact := "testdata/file_to_redact.txt"
	wantFile := "testdata/want.txt"
	source, _ := os.Open(originalFile)
	destination, _ := os.Create(fileToRedact)
	w, err := io.Copy(destination, source)
	fmt.Print(w)
	source.Close()
	destination.Close()
	fileRedactor := NewFileRedactor(fileutil.NewFileManager())
	err = fileRedactor.RedactFiles([]string{fileToRedact}, []string{"SUPER_SECRET_WORD"})

	assert.NoError(t, err)

	actual, _ := os.ReadFile(fileToRedact)
	expected, _ := os.ReadFile(wantFile)

	assert.Equalf(t, actual, expected, "")
}
