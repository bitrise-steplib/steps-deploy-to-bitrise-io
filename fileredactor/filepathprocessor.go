package fileredactor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/env"
)

type FilePathProcessor interface {
	ProcessFilePaths(string) ([]string, error)
}

type filePathProcessor struct {
	envRepository env.Repository
}

func NewFilePathProcessor(repository env.Repository) FilePathProcessor {
	return filePathProcessor{
		envRepository: repository,
	}
}

func (f filePathProcessor) ProcessFilePaths(filePaths string) ([]string, error) {
	filePaths = strings.TrimSpace(filePaths)
	if filePaths == "" {
		return nil, nil
	}

	processedFilePaths := []string{}

	list := strings.Split(filePaths, "\n")
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		path := item
		if strings.HasPrefix(item, "$") {
			path = f.envRepository.Get(item[1:])
			if path != "" {
				return nil, fmt.Errorf("invalid item (%s): environment variable isn't set", item)
			}
		}

		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}

		isDir, err := f.isDirectory(path)
		if err != nil {
			return nil, fmt.Errorf("failed to check if path (%s) is a directory: %w", path, err)
		}
		if isDir {
			return nil, fmt.Errorf("path (%s) is a directory and cannot be redacted, please make sure to only provide filepaths as inputs", path)
		}

		processedFilePaths = append(processedFilePaths, path)
	}

	return processedFilePaths, nil
}

func (f filePathProcessor) isDirectory(s string) (bool, error) {
	fileInfo, err := os.Stat(s)
	if err != nil {
		return false, err
	}

	if fileInfo.IsDir() {
		return true, nil
	}

	return false, nil
}
