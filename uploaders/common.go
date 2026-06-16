package uploaders

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	androidparser "github.com/bitrise-io/go-android/v2/metaparser"
	iosparser "github.com/bitrise-io/go-xcode/v2/metaparser"
)

type ArtifactURLs struct {
	PublicInstallPageURL string
	PermanentDownloadURL string
	DetailsPageURL       string
}

type AppDeploymentMetaData struct {
	AndroidArtifactInfo    *androidparser.ArtifactMetadata
	IOSArtifactInfo        *iosparser.ArtifactMetadata
	NotifyUserGroups       string
	AlwaysNotifyUserGroups string
	NotifyEmails           string
	IsEnablePublicPage     bool
}

type ArtifactArgs struct {
	Path     string
	FileSize int64 // bytes
}

type TransferDetails struct {
	Size      int64
	Duration  time.Duration
	Hostname  string
	Multipart *MultipartTransferDetails
}

type MultipartTransferDetails struct {
	PartCount   int
	PartSize    int64
	Concurrency int
}

func printableAppInfo(appInfo interface{}) string {
	bytes, err := json.Marshal(appInfo)
	if err != nil {
		return fmt.Sprintf("failed to marshal app info: %+v, error: %s", appInfo, err)
	}

	return string(bytes)
}

func extractHost(downloadURL string) string {
	u, err := url.Parse(downloadURL)
	if err != nil {
		return "unknown"
	}

	return strings.TrimPrefix(u.Hostname(), "www.")
}
