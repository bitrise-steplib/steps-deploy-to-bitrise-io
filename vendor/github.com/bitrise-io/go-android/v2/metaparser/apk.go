package metaparser

import (
	"github.com/bitrise-io/go-android/v2/metaparser/androidartifact"
	"github.com/bitrise-io/go-android/v2/metaparser/androidsignature"
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

	signature, err := androidsignature.ReadAPKSignature(pth)
	if err != nil {
		m.logger.Warnf("Failed to get signature of `%s`: %s", pth, err)
	}

	return &ArtifactMetadata{
		AppInfo:        apkInfo,
		FileSizeBytes:  fileSize,
		Module:         info.Module,
		ProductFlavour: info.ProductFlavour,
		BuildType:      info.BuildType,
		SignedBy:       signature,
	}, nil
}
