package uploaders

import (
	"crypto/md5" //nolint:gosec // S3/R2 ETag algorithm and content hash, not used for security
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// checksumStatus describes the outcome of validating an uploaded file against the ETag the storage
// backend reported.
type checksumStatus string

const (
	checksumSingleOK          checksumStatus = "single:ok"
	checksumSingleMismatch    checksumStatus = "single:mismatch"
	checksumMultipartOK       checksumStatus = "multipart:ok"
	checksumMultipartMismatch checksumStatus = "multipart:mismatch"
	checksumETagUnavailable   checksumStatus = "etag:unavailable"
)

// verifyAndLogChecksum computes the MD5 of the uploaded file and validates it against the ETag the
// backend reported for the stored object. It is purely diagnostic: outcomes are logged and recorded
// on details, never returned as an error. partSize selects the calculation: 0 for a single-shot
// upload (the object ETag is the whole-file MD5), or the multipart part size for a multipart upload
// (the ETag is the MD5 of the concatenated part digests, suffixed with the part count).
func (u *Uploader) verifyAndLogChecksum(path string, partSize int64, details *TransferDetails) {
	md5sum, multipartETag, err := fileChecksums(path, partSize)
	if err != nil {
		u.logger.Warnf("Failed to compute checksum of %s: %s", filepath.Base(path), err)
		return
	}
	details.MD5 = md5sum

	if details.ETag == "" {
		details.ChecksumStatus = string(checksumETagUnavailable)
		// Some responses (e.g. empty objects) carry no ETag; only flag it for non-empty files.
		if details.Size > 0 {
			u.logger.Warnf("No ETag in upload response for %s (md5=%s)", filepath.Base(path), md5sum)
		}
		return
	}

	status := validateChecksum(md5sum, multipartETag, details.ETag)
	details.ChecksumStatus = string(status)

	switch status {
	case checksumSingleMismatch:
		u.logger.Warnf("Checksum mismatch for %s: md5=%s, etag=%s", filepath.Base(path), md5sum, details.ETag)
	case checksumMultipartMismatch:
		u.logger.Warnf("Multipart ETag mismatch for %s: md5=%s, recomputed=%s, etag=%s (object may use a non-MD5 ETag or a different upload part size)", filepath.Base(path), md5sum, multipartETag, details.ETag)
	default:
		u.logger.Printf("Checksum for %s: md5=%s, etag=%s, validation=%s", filepath.Base(path), md5sum, details.ETag, status)
	}
}

// validateChecksum compares the local checksums against the reported ETag: an ETag of the form
// "<hash>-<N>" is multipart and compared against multipartETag, otherwise it is a plain MD5 compared
// against md5sum.
func validateChecksum(md5sum, multipartETag, etag string) checksumStatus {
	if strings.Contains(etag, "-") {
		if strings.EqualFold(multipartETag, etag) {
			return checksumMultipartOK
		}
		return checksumMultipartMismatch
	}

	if strings.EqualFold(md5sum, etag) {
		return checksumSingleOK
	}
	return checksumSingleMismatch
}

// fileChecksums reads the file once and returns the hex-encoded MD5 of the whole file and, when
// partSize > 0, the S3/R2 multipart ETag for that part size: the MD5 of the concatenated binary MD5
// digests of each part, hex-encoded and suffixed with the part count. Computing both in a single
// pass avoids re-reading large files during checksum validation.
func fileChecksums(path string, partSize int64) (md5hex, multipartETag string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()

	fullHash := md5.New() //nolint:gosec // S3 ETag algorithm; not used for security

	if partSize <= 0 {
		if _, err := io.Copy(fullHash, f); err != nil {
			return "", "", err
		}
		return fmt.Sprintf("%x", fullHash.Sum(nil)), "", nil
	}

	// Hash each part separately while feeding every byte to the whole-file hash.
	partHash := md5.New() //nolint:gosec // S3 ETag algorithm; not used for security
	combined := io.MultiWriter(fullHash, partHash)
	var partDigests []byte
	partCount := 0
	for {
		partHash.Reset()
		n, err := io.CopyN(combined, f, partSize)
		if n > 0 {
			partDigests = append(partDigests, partHash.Sum(nil)...)
			partCount++
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", "", err
		}
	}

	sum := md5.Sum(partDigests)
	return fmt.Sprintf("%x", fullHash.Sum(nil)), fmt.Sprintf("%x-%d", sum, partCount), nil
}
