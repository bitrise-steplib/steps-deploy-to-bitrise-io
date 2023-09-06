package fileredactor

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

// FilePathProcessor is an interface for an entity which accepts file paths separated by the
// newline (`\n`) character and returns a slice of absolute paths.
type FilePathProcessor interface {
	ProcessFilePaths(string) ([]string, error)
}

type filePathProcessor struct {
	envRepository env.Repository
	pathModifier  pathutil.PathModifier
	pathChecker   pathutil.PathChecker
}

// NewFilePathProcessor returns a structure which implements the FilePathProcessor interface.
// The implementation includes handling filepaths defined as environment variables, relative file paths,
// and absolute file paths.
// The implementation also includes making sure the filepath exists and is not a directory.
func NewFilePathProcessor(repository env.Repository, modifier pathutil.PathModifier, checker pathutil.PathChecker) FilePathProcessor {
	return filePathProcessor{
		envRepository: repository,
		pathModifier:  modifier,
		pathChecker:   checker,
	}
}

func (f filePathProcessor) ProcessFilePaths(filePaths string) ([]string, error) {
	filePaths = strings.TrimSpace(filePaths)
	if filePaths == "" {
		return nil, nil
	}

	var processedFilePaths []string

	list := strings.Split(filePaths, "\n")
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		path := item
		if strings.HasPrefix(item, "$") {
			path = f.envRepository.Get(item[1:])
			if path == "" {
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
