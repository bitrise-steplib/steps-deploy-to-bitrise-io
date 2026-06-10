package uploaders

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	androidparser "github.com/bitrise-io/go-android/v2/metaparser"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	iosparser "github.com/bitrise-io/go-xcode/v2/metaparser"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

type Uploader struct {
	logger               log.Logger
	fileManager          fileutil.FileManager
	androidParser        *androidparser.Parser
	iosParser            *iosparser.Parser
	tracker              tracker
	useMultipartUpload   bool
	multipartConcurrency int
}

func New(
	logger log.Logger,
	fileManager fileutil.FileManager,
	androidParser *androidparser.Parser,
	iosParser *iosparser.Parser,
	useMultipartUpload bool,
	multipartConcurrency int,
) *Uploader {
	return &Uploader{
		logger:               logger,
		fileManager:          fileManager,
		androidParser:        androidParser,
		iosParser:            iosParser,
		tracker:              newTracker(env.NewRepository(), logger),
		useMultipartUpload:   useMultipartUpload,
		multipartConcurrency: multipartConcurrency,
	}
}

func (u *Uploader) Wait() {
	u.tracker.wait()
}

func (u *Uploader) upload(buildURL, token string, artifact ArtifactArgs, artifactType, contentType string, item *deployment.DeployableItem, buildArtifactMeta *AppDeploymentMetaData) ([]ArtifactURLs, error) {
	u.logFileHash(artifact, item)

	if u.useMultipartUpload {
		return u.uploadMultipart(buildURL, token, artifact, artifactType, contentType, item, buildArtifactMeta)
	}
	return u.uploadSingle(buildURL, token, artifact, artifactType, contentType, item, buildArtifactMeta)
}

// logFileHash computes and logs the SHA256 of the file being uploaded. It also
// verifies that the number of bytes readable from the file matches the size
// reported by os.Stat (artifact.FileSize) — the value sent to the backend as
// file_size_bytes and used as the upload Content-Length. A mismatch means the
// file changed between sizing and upload, which would corrupt the artifact.
func (u *Uploader) logFileHash(artifact ArtifactArgs, item *deployment.DeployableItem) {
	hash, size, err := u.sha256OfFile(artifact.Path)
	if err != nil {
		u.logger.Warnf("Failed to compute SHA256 of %s: %s", artifact.Path, err)
		return
	}
	u.logger.Printf("SHA256: %s", hash)

	if size != artifact.FileSize {
		u.logger.Warnf("File size mismatch for %s: reported %d bytes, but %d bytes are uploadable. The file changed during deployment.",
			artifact.Path, artifact.FileSize, size)
	}
}

// sha256OfFile streams the file once, returning both its SHA256 and the number
// of bytes read.
func (u *Uploader) sha256OfFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			u.logger.Warnf("Failed to close file: %s", err)
		}
	}()

	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, fmt.Errorf("read file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), size, nil
}
