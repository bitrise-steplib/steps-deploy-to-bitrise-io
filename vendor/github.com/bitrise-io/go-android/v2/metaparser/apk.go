package metaparser

import (
	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"
)

// ParseAPKData ...
func (m *Parser) ParseAPKData(pth string) (*ArtifactMetadata, error) {
	apkInfo, err := androidartifact.GetAPKInfoWithFallback(m.logger, pth)
	if err != nil {
		return nil, err
	}

	fileSize, err := m.fileManager.FileSizeInBytes(pth)
	if err != nil {
		m.logger.Warnf("Failed to get apk size, error: %s", err)
	}

	info := androidartifact.ParseArtifactPath(pth)

	return &ArtifactMetadata{
		AppInfo:        apkInfo,
		FileSizeBytes:  fileSize,
		Module:         info.Module,
		ProductFlavour: info.ProductFlavour,
		BuildType:      info.BuildType,
	}, nil
}
