package deployment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	separator = ":"
)

// IntermediateFileMetaData ...
type IntermediateFileMetaData struct {
	EnvKey string `json:"env_key"`
	IsDir  bool   `json:"is_dir"`
}

// DeployableItem ...
type DeployableItem struct {
	Path                 string
	IntermediateFileMeta *IntermediateFileMetaData
}

// ConvertPaths ...
func ConvertPaths(paths []string) []DeployableItem {
	if len(paths) == 0 {
		return []DeployableItem{}
	}

	var items []DeployableItem
	for _, path := range paths {
		items = append(items, DeployableItem{
			Path:                 path,
			IntermediateFileMeta: nil,
		})
	}

	return items
}

// ArchiveDirFunction ...
type ArchiveDirFunction func(sourceDirPath, outputPath string, isContentOnly bool) error

// IsDirFunction ...
type IsDirFunction func(path string) (bool, error)

// DefaultIsDirFunction ...
func DefaultIsDirFunction(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), nil
}

// Collector ...
type Collector struct {
	isDirFunction      IsDirFunction
	archiveDirFunction ArchiveDirFunction
	temporaryFolder    string
}

// NewCollector ...
func NewCollector(
	isDirFunction IsDirFunction,
	archiveDirFunction ArchiveDirFunction,
	temporaryFolder string,
) Collector {
	return Collector{
		isDirFunction:      isDirFunction,
		archiveDirFunction: archiveDirFunction,
		temporaryFolder:    temporaryFolder,
	}
}

// AddIntermediateFiles ...
func (c Collector) AddIntermediateFiles(deployableItems []DeployableItem, intermediateFileList string) ([]DeployableItem, error) {
	intermediateFiles, err := c.processIntermediateFiles(intermediateFileList)
	if err != nil {
		return []DeployableItem{}, err
	}

	deployableItems, err = c.mergeItems(deployableItems, intermediateFiles)
	if err != nil {
		return []DeployableItem{}, err
	}

	deployableItems, err = c.archiveDirectories(deployableItems)
	if err != nil {
		return []DeployableItem{}, err
	}

	return deployableItems, nil
}

func (c Collector) processIntermediateFiles(s string) (map[string]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	intermediateFiles := map[string]string{}

	list := strings.Split(s, "\n")
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		if strings.Count(item, separator) != 1 {
			return nil, fmt.Errorf("invalid item (%s): doesn't contain exactly one '%s' character", item, separator)
		}

		idx := strings.LastIndex(item, separator)
		path := item[:idx]
		if path == "" {
			return nil, fmt.Errorf("invalid item (%s): doesn't specify file path", item)
		}

		key := item[idx+1:]
		if key == "" {
			return nil, fmt.Errorf("invalid item (%s): doesn't specify key", item)
		}

		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}

		intermediateFiles[path] = key
	}

	return intermediateFiles, nil
}

func (c Collector) mergeItems(items []DeployableItem, files map[string]string) ([]DeployableItem, error) {
	for path, envKey := range files {
		isDirectory, err := c.isDirFunction(path)
		if err != nil {
			return nil, err
		}

		index := c.indexOfItemWithPath(items, path)

		if index == -1 {
			item := DeployableItem{
				Path: path,
				IntermediateFileMeta: &IntermediateFileMetaData{
					EnvKey: envKey,
					IsDir:  isDirectory,
				},
			}
			items = append(items, item)
		} else {
			items[index].IntermediateFileMeta = &IntermediateFileMetaData{
				EnvKey: envKey,
				IsDir:  isDirectory,
			}
		}
	}

	return items, nil
}

func (c Collector) indexOfItemWithPath(items []DeployableItem, path string) int {
	if items == nil {
		return -1
	}

	for i, item := range items {
		if item.Path == path {
			return i
		}
	}

	return -1
}

func (c Collector) archiveDirectories(items []DeployableItem) ([]DeployableItem, error) {
	for i, item := range items {
		if item.IntermediateFileMeta != nil && item.IntermediateFileMeta.IsDir {
			path, err := c.archiveDir(item.Path)
			if err != nil {
				return nil, err
			}

			items[i].Path = path
		}
	}

	return items, nil
}

func (c Collector) archiveDir(path string) (string, error) {
	name := filepath.Base(path)
	targetPath := filepath.Join(c.temporaryFolder, name+".tar")

	if err := c.archiveDirFunction(path, targetPath, true); err != nil {
		return "", fmt.Errorf("failed to archive output dir, error: %s", err)
	}

	return targetPath, nil
}
