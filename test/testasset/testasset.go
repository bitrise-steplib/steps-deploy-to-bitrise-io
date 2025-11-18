package testasset

import (
	"os"
	"path/filepath"
	"slices"
)

var AssetTypes = []string{".jpg", ".jpeg", ".png", ".txt", ".log"}

// NOTE: These video types are also supported on the UI
var AssetTypesWithVideo = append(AssetTypes, ".mp4", ".webm", ".ogg")

func IsSupportedAssetType(fileName string) bool {
	ext := filepath.Ext(fileName)

	if os.Getenv("ENABLE_TEST_VIDEO_UPLOAD") == "true" {
		return slices.Contains(AssetTypesWithVideo, ext)
	}

	return slices.Contains(AssetTypes, ext)
}
