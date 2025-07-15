package testasset

import (
	"path/filepath"
	"slices"
)

var AssetTypes = []string{".jpg", ".jpeg", ".png"}

func IsSupportedAssetType(fileName string) bool {
	ext := filepath.Ext(fileName)
	return slices.Contains(AssetTypes, ext)
}
