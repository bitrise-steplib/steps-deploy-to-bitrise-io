package testasset

import (
	"path/filepath"
	"slices"
	"strings"
)

var AssetTypes = []string{".jpg", ".jpeg", ".png", ".txt", ".log", ".mp4", ".webm", ".ogg"}

func IsSupportedAssetType(fileName string) bool {
	ext := filepath.Ext(fileName)

	if slices.Contains(AssetTypes, strings.ToLower(ext)) {
		return true
	}

	return false
}
