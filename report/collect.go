package report

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
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

			if d.IsDir() || d.Name() == ".DS_Store" {
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

		reports = append(reports, Report{
			Name:   entry.Name(),
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

	return http.DetectContentType(buff)
}
