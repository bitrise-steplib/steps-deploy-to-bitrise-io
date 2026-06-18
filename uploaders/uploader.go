package uploaders

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"

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
	u.logCRC32C(artifact.Path)

	if u.useMultipartUpload {
		return u.uploadMultipart(buildURL, token, artifact, artifactType, contentType, item, buildArtifactMeta)
	}
	return u.uploadSingle(buildURL, token, artifact, artifactType, contentType, item, buildArtifactMeta)
}

// logCRC32C computes and logs the CRC32C (Castagnoli) checksum of the file being uploaded, as a
// record of the uploaded bytes' integrity. It is purely diagnostic: failures are logged, never
// returned.
func (u *Uploader) logCRC32C(path string) {
	crc, err := fileCRC32C(os.DirFS(filepath.Dir(path)), filepath.Base(path))
	if err != nil {
		u.logger.Warnf("Failed to compute CRC32C of %s: %s", filepath.Base(path), err)
		return
	}
	u.logger.Printf("CRC32C for %s: %s", filepath.Base(path), crc)
}

// fileCRC32C returns the named file's CRC32C (Castagnoli) checksum, base64-encoded big-endian — the
// same representation S3/R2 (x-amz-checksum-crc32c) and GCS (x-goog-hash crc32c) use.
func fileCRC32C(fsys fs.FS, name string) (sum string, err error) {
	f, err := fsys.Open(name)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, f.Close()) }()

	h := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	var buf [crc32.Size]byte
	binary.BigEndian.PutUint32(buf[:], h.Sum32())
	return base64.StdEncoding.EncodeToString(buf[:]), nil
}
