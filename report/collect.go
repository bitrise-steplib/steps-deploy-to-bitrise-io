package report

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	htmlReportInfoFile = "report-info.json"
	plainContentType   = "text/plain"
)

func collectReports(dir string) ([]Report, error) {
	var reports []Report

	entries, err := os.ReadDir(dir)
	if err != nil {
		return reports, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		var assets []Asset
		testDir := filepath.Join(dir, entry.Name())
		fn := func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() || d.Name() == ".DS_Store" || d.Name() == htmlReportInfoFile {
				return nil
			}

			relativePath, pathErr := filepath.Rel(testDir, path)
			if pathErr != nil {
				return pathErr
			}

			info, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}

			contentType := detectContentType(path)

			assets = append(assets, Asset{
				Path:                path,
				TestDirRelativePath: relativePath,
				FileSize:            info.Size(),
				ContentType:         contentType,
			})

			return nil
		}
		if err := filepath.WalkDir(testDir, fn); err != nil {
			return nil, err
		}

		if len(assets) == 0 {
			continue
		}

		var reportInfo Info
		if infoFileData, err := os.ReadFile(filepath.Join(testDir, htmlReportInfoFile)); err == nil {
			if err := json.Unmarshal(infoFileData, &reportInfo); err != nil {
				return nil, fmt.Errorf("cannot parse report info file: %w", err)
			}
		}

		reports = append(reports, Report{
			Name:   entry.Name(),
			Info:   reportInfo,
			Assets: assets,
		})
	}

	return reports, nil
}

func detectContentType(path string) string {
	fallbackType := "application/octet-stream"

	file, err := os.Open(path)
	if err != nil {
		return fallbackType
	}
	defer func() {
		if err := file.Close(); err != nil {
			// This is empty on purpose to please the linter
		}
	}()

	// At most, the first 512 bytes of data are used:
	// https://golang.org/src/net/http/sniff.go?s=646:688#L11
	buff := make([]byte, 512)

	bytesRead, err := file.Read(buff)
	if err != nil && err != io.EOF {
		return fallbackType
	}

	// Slice to remove fill-up zero values which cause a wrong content type detection in the next step
	buff = buff[:bytesRead]
	contentType := http.DetectContentType(buff)
	extension := filepath.Ext(path)

	return overrideContentTypeForKnownExtensions(extension, contentType)
}

func overrideContentTypeForKnownExtensions(extension, contentType string) string {
	if strings.HasPrefix(contentType, plainContentType) == false {
		return contentType
	}

	newContentType := mime.TypeByExtension(extension)
	if newContentType != "" {
		return newContentType
	}

	return contentType
}
