package uploaders

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

const snapshotFileSizeLimitInBytes = 1024 * 1024 * 1024

// DeployFile ...
func (u *Uploader) DeployFile(item deployment.DeployableItem, buildURL, token string) (ArtifactURLs, error) {
	pth := item.Path
	fileSize, err := u.fileManager.FileSizeInBytes(item.Path)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("get file size: %w", err)
	}

	// TODO: This is a workaround to avoid uploading a file that is being modified during the upload process,
	//  which can cause an issue like: request body larger than specified content length at file upload.
	deploySnapshot := false
	if fileSize <= snapshotFileSizeLimitInBytes {
		snapshotPth, err := createSnapshot(pth)
		if err != nil {
			u.logger.Warnf("failed to create snapshot of %s: %s", pth, err)
		} else {
			defer func() {
				if err := os.Remove(snapshotPth); err != nil {
					u.logger.Warnf("Failed to remove snapshot file: %s", err)
				}
			}()
			pth = snapshotPth
			deploySnapshot = true
		}
	}

	if deploySnapshot {
		u.logger.Printf("Deploying snapshot of original file: %s", pth)
	} else {
		u.logger.Printf("Deploying file: %s", pth)
	}

	artifact := ArtifactArgs{
		Path:     pth,
		FileSize: fileSize,
	}

	uploadURL, artifactID, err := createArtifact(buildURL, token, artifact, "file", "")
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("create file artifact: %s %w", artifact.Path, err)
	}

	if err := UploadArtifact(uploadURL, artifact, ""); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload file artifact, error: %s", err)
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, nil, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish file artifact, error: %s", err)
	}
	return artifactURLs, nil
}

// createSnapshot copies a file to a temporary directory with the same file name.
func createSnapshot(originalPath string) (string, error) {
	originalFile, err := os.Open(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open original file: %w", err)
	}
	defer func() {
		if err := originalFile.Close(); err != nil {
			log.Warnf("Failed to close original file: %s", err)
		}
	}()

	tmpDir, err := pathutil.NormalizedOSTempDirPath("snapshot")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	tmpFilePath := filepath.Join(tmpDir, filepath.Base(originalPath))
	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			log.Warnf("Failed to close temp file: %s", err)
		}
	}()

	if _, err := io.Copy(tmpFile, originalFile); err != nil {
		return "", fmt.Errorf("failed to copy contents: %w", err)
	}

	return tmpFilePath, nil
}
