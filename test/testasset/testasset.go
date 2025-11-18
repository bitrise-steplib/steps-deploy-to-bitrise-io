package testasset

import (
	"os"
	"path/filepath"
	"slices"
)

var AssetTypes = []string{".jpg", ".jpeg", ".png", ".txt", ".log"}
var VideoTypes = []string{".mp4", ".webm", ".ogg"} // These video types are also supported on the UI

func IsSupportedAssetType(fileName string) bool {
	ext := filepath.Ext(fileName)

	if slices.Contains(AssetTypes, ext) {
		return true
	}

	if os.Getenv("ENABLE_TEST_VIDEO_UPLOAD") == "true" {
		return slices.Contains(VideoTypes, ext)
	}

	return false
}
