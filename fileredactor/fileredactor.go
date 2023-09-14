package fileredactor

import (
	"fmt"
	"io"
	"os"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/redactwriter"
)

const bufferSize = 64 * 1024

// FileRedactor is an interface for a structure which, given a slice of file paths and another slice of secrets can
// process the specified files to redact secrets from them.
type FileRedactor interface {
	RedactFiles([]string, []string) error
}

type fileRedactor struct {
	fileManager fileutil.FileManager
}

// NewFileRedactor returns a structure that implements the FileRedactor interface
func NewFileRedactor(manager fileutil.FileManager) FileRedactor {
	return fileRedactor{
		fileManager: manager,
	}
}

func (f fileRedactor) RedactFiles(filePaths []string, secrets []string) error {
	logger := log.NewLogger()
	for _, path := range filePaths {
		if err := f.redactFile(path, secrets, logger); err != nil {
			return fmt.Errorf("failed to redact file (%s): %w", path, err)
		}
	}

	return nil
}

func (f fileRedactor) redactFile(path string, secrets []string, logger log.Logger) error {
	source, err := f.fileManager.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file for redaction (%s): %w", path, err)
	}
	defer func() {
		if err := source.Close(); err != nil {
			logger.Warnf("Failed to close file: %s", err)
		}
	}()

	newPath := path + ".redacted"
	destination, err := os.Create(newPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file for redaction: %w", err)
	}
	defer func() {
		if err := destination.Close(); err != nil {
			logger.Warnf("Failed to close file: %s", err)
		}
	}()

	redactWriter := redactwriter.New(secrets, destination, logger)
	if _, err := io.Copy(redactWriter, source); err != nil {
		return fmt.Errorf("failed to redact secrets: %w", err)
	}

	if err := redactWriter.Close(); err != nil {
		return fmt.Errorf("failed to close redact writer: %w", err)
	}

	//rename new file to old file name
	err = os.Rename(newPath, path)
	if err != nil {
		return fmt.Errorf("failed to overwrite old file (%s) with redacted file: %w", path, err)
	}

	return nil
}
