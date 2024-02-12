package zipreader

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/ryanuber/go-glob"
)

type ZipReader struct {
	zipReader *zip.ReadCloser
}

func OpenZip(archivePath string) (*ZipReader, error) {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive %s: %w", archivePath, err)
	}

	return &ZipReader{zipReader: zipReader}, nil
}

func (reader ZipReader) Close() error {
	return reader.zipReader.Close()
}

func (reader ZipReader) ReadFile(targetPathGlob string) ([]byte, error) {
	var files []*zip.File
	var fileNames []string
	for _, file := range reader.zipReader.File {
		name := file.Name
		fmt.Println(name)
		if glob.Glob(targetPathGlob, name) {
			files = append(files, file)
			fileNames = append(fileNames, name)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no file found with pattern: %s", targetPathGlob)
	} else if len(files) > 1 {
		return nil, fmt.Errorf("multiple files (%s) found with pattern: %s", strings.Join(fileNames, ", "), targetPathGlob)
	}

	file := files[0]
	r, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open archive file %s: %w", file.Name, err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Warnf("failed to close archive file %s: %s", file.Name, err)
		}
	}()

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive file %s: %w", file.Name, err)
	}

	return b, nil
}

func (reader ZipReader) IsFileOrDirExistsInZipArchive(targetPathGlob string) bool {
	for _, file := range reader.zipReader.File {
		if glob.Glob(targetPathGlob, file.Name) {
			return true
		}
	}
	return false
}
