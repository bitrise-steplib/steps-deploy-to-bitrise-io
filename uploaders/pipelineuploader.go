package uploaders

import (
	"fmt"
	"github.com/bitrise-io/go-utils/log"
	"os"
	"path/filepath"
	"strings"
)

const (
	separator = ":"
)

type PipelineUploader interface {
	UploadFiles(fileList, buildURL, buildAPIToken string) error
}

type PipelineFileUploadFunction func(path, buildURL, token string, metaData *PipelineIntermediateFileMetaData) (ArtifactURLs, error)
type IsDirFunction func(path string) (bool, error)
type ZipDirFunction func(sourceDirPth, destinationZipPth string, isContentOnly bool) error

func DefaultIsDirFunction(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), nil
}

type pipelineUploader struct {
	uploadFunction  PipelineFileUploadFunction
	isDirFunction   IsDirFunction
	zipDirFunction  ZipDirFunction
	temporaryFolder string
}

func NewPipelineUploader(
	uploadFunction PipelineFileUploadFunction,
	isDirFunction IsDirFunction,
	zipDirFunction ZipDirFunction,
	temporaryFolder string,
) PipelineUploader {
	return pipelineUploader{
		uploadFunction:  uploadFunction,
		isDirFunction:   isDirFunction,
		zipDirFunction:  zipDirFunction,
		temporaryFolder: temporaryFolder,
	}
}

func (p pipelineUploader) UploadFiles(fileList, buildURL, buildAPIToken string) error {
	intermediateFiles, err := p.parsePipelineIntermediateFiles(fileList)
	if err != nil {
		return err
	}

	for path, key := range intermediateFiles {
		fmt.Println()
		log.Donef("Pushing pipeline intermediate file: %s", path)

		filePath, metaData, err := p.createFilePathAndMetaData(path, key)
		if err != nil {
			return err
		}

		_, err = p.uploadFunction(filePath, buildURL, buildAPIToken, &metaData)
		if err != nil {
			return fmt.Errorf("failed to push pipeline intermediate file (%s): %s", path, err)
		}
	}

	return nil
}

func (p pipelineUploader) parsePipelineIntermediateFiles(s string) (map[string]string, error) {
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

		intermediateFiles[path] = key
	}

	return intermediateFiles, nil
}

func (p pipelineUploader) createFilePathAndMetaData(path, key string) (string, PipelineIntermediateFileMetaData, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", PipelineIntermediateFileMetaData{}, fmt.Errorf("failed to push pipeline intermediate file (%s): %s", path, err)
	}

	meta := PipelineIntermediateFileMetaData{
		EnvKey: key,
	}

	isDir, err := p.isDirFunction(absolutePath)
	if err != nil {
		return "", PipelineIntermediateFileMetaData{}, err
	}

	if isDir {
		absolutePath, err = p.zipDir(absolutePath)
		if err != nil {
			return "", PipelineIntermediateFileMetaData{}, err
		}

		meta.IsDir = true
	}

	return absolutePath, meta, nil
}

func (p pipelineUploader) zipDir(path string) (string, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	targetPth := filepath.Join(p.temporaryFolder, name+".zip")

	if err := p.zipDirFunction(path, targetPth, true); err != nil {
		return "", fmt.Errorf("failed to zip output dir, error: %s", err)
	}

	return targetPth, nil
}
