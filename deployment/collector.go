package deployment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/v2/env"
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
		return nil
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

// ZipDirFunction ...
type ZipDirFunction func(sourceDirPth, destinationZipPth string, isContentOnly bool) error

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
	zipComparator   ZipComparator
	isDirFunction   IsDirFunction
	zipDirFunction  ZipDirFunction
	envRepository   env.Repository
	temporaryFolder string
}

// NewCollector ...
func NewCollector(
	zipComparator ZipComparator,
	isDirFunction IsDirFunction,
	zipDirFunction ZipDirFunction,
	envRepository env.Repository,
	temporaryFolder string,
) Collector {
	return Collector{
		zipComparator:   zipComparator,
		isDirFunction:   isDirFunction,
		zipDirFunction:  zipDirFunction,
		envRepository:   envRepository,
		temporaryFolder: temporaryFolder,
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

	deployableItems, err = c.zipDirectories(deployableItems)
	if err != nil {
		return []DeployableItem{}, err
	}

	deployableItems = c.mergeZipPairs(deployableItems)

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

		split := strings.Split(item, separator)
		if len(split) > 2 {
			return nil, fmt.Errorf("invalid item (%s): contains more than one '%s' character", item, separator)
		}

		key := split[len(split)-1]
		if key == "" {
			return nil, fmt.Errorf("invalid item (%s): environment variable key is empty", item)
		}

		path := strings.Join(split[:len(split)-1], separator)
		if path == "" && len(split) == 1 {
			path = c.envRepository.Get(key)
			if path == "" {
				return nil, fmt.Errorf("invalid item (%s): environment variable isn't set", item)
			}
		}

		if path == "" {
			return nil, fmt.Errorf("invalid item (%s): empty path", item)
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

func (c Collector) zipDirectories(items []DeployableItem) ([]DeployableItem, error) {
	for i, item := range items {
		if item.IntermediateFileMeta != nil && item.IntermediateFileMeta.IsDir {
			path, err := c.zipDir(item.Path)
			if err != nil {
				return nil, err
			}

			items[i].Path = path
		}
	}

	return items, nil
}

func (c Collector) zipDir(path string) (string, error) {
	name := filepath.Base(path)
	targetPth := filepath.Join(c.temporaryFolder, name+".zip")

	if err := c.zipDirFunction(path, targetPth, true); err != nil {
		return "", fmt.Errorf("failed to zip output dir, error: %s", err)
	}

	return targetPth, nil
}

func (c Collector) mergeZipPairs(deployableItems []DeployableItem) []DeployableItem {
	var mergedDeployableItems []DeployableItem
	pipelineDirs := map[string]DeployableItem{}
	zipBuildArtifacts := map[string]DeployableItem{}

	for _, item := range deployableItems {
		if item.IntermediateFileMeta != nil {
			if item.IntermediateFileMeta.IsDir {
				pipelineDirs[item.Path] = item
			}
			mergedDeployableItems = append(mergedDeployableItems, item)
			continue
		}

		if filepath.Ext(item.Path) == ".zip" {
			zipBuildArtifacts[item.Path] = item
		} else {
			mergedDeployableItems = append(mergedDeployableItems, item)
		}
	}

	// At this point mergedDeployableItems contains all Pipeline Files and no ZIP Build Artifacts.
	// Let's find Pipeline File pairs of ZIP Build Artifacts.
	for _, pipelineDir := range pipelineDirs {
		for pth, zipBuildArtifact := range zipBuildArtifacts {
			same, err := c.zipComparator.Equals(pipelineDir.Path, zipBuildArtifact.Path)
			if err != nil {
				log.Warnf("Couldn't compare Pipeline File (%s) and Build Artifact (%s): %s", pipelineDir.Path, zipBuildArtifact.Path, err)
				continue
			}

			if same {
				log.Warnf("Same directory specified both as Build Artifact (%s) and Pipeline File (%s), keeping Pipeline File...", zipBuildArtifact.Path, pipelineDir.Path)
				delete(zipBuildArtifacts, pth)
			}
		}
	}

	for _, item := range zipBuildArtifacts {
		mergedDeployableItems = append(mergedDeployableItems, item)
	}
	return mergedDeployableItems
}
