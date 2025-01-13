package metaparser

import (
	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"
	"github.com/bitrise-io/go-android/v2/metaparser/androidsignature"
)

// ParseAABData ...
func (m *Parser) ParseAABData(pth string) (*ArtifactMetadata, error) {
	aabInfo, err := androidartifact.GetAABInfo(m.bundletoolPath, pth)
	if err != nil {
		m.logger.Warnf("Failed to parse AAB info: %s", err)
		m.logger.AABParseWarnf("aab-parse", "aabparser package failed to parse AAB, error: %s", err)
		return nil, err
	}

	fileSize, err := m.fileManager.FileSizeInBytes(pth)
	if err != nil {
		m.logger.Warnf("Failed to get apk size, error: %s", err)
	}

	info := androidartifact.ParseArtifactPath(pth)

	signature, err := androidsignature.ReadAABSignature(pth)
	if err != nil {
		m.logger.Warnf("Failed to get signature of `%s`: %s", pth, err)
	}

	return &ArtifactMetadata{
		AppInfo:        aabInfo,
		FileSizeBytes:  fileSize,
		Module:         info.Module,
		ProductFlavour: info.ProductFlavour,
		BuildType:      info.BuildType,
		SignedBy:       signature,
	}, nil
}
