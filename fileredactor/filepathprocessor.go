package fileredactor

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

type FilePathProcessor interface {
	ProcessFilePaths(string) ([]string, error)
}

type filePathProcessor struct {
	envRepository env.Repository
	pathChecker   pathutil.PathChecker
	pathModifier  pathutil.PathModifier
}

func NewFilePathProcessor(repository env.Repository, checker pathutil.PathChecker, modifier pathutil.PathModifier) FilePathProcessor {
	return filePathProcessor{
		envRepository: repository,
		pathChecker:   checker,
		pathModifier:  modifier,
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

		path, err := f.pathModifier.AbsPath(path)
		if err != nil {
			return nil, err
		}

		isDir, err := f.pathChecker.IsDirExists(path)
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
