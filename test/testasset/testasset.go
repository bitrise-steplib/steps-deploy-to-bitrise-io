package testasset

import (
	"path/filepath"
	"slices"
	"strings"
)

var AssetTypes = []string{".jpg", ".jpeg", ".png", ".txt", ".log", ".mp4", ".webm", ".ogg"}

func IsSupportedAssetType(fileName string) bool {
	ext := filepath.Ext(fileName)
	return slices.Contains(AssetTypes, strings.ToLower(ext))
}
