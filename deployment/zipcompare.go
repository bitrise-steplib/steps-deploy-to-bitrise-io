package deployment

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/log"
)

type ZipFileInfo struct {
	Name               string
	UncompressedSize64 uint64
	CRC32              uint32
}

func (i ZipFileInfo) equals(info ZipFileInfo) bool {
	return i.Name == info.Name &&
		i.UncompressedSize64 == info.UncompressedSize64 &&
		i.CRC32 == info.CRC32
}

type CompareResult struct {
	removed  []string
	changed  []string
	added    []string
	matching []string
}

func (r CompareResult) hasChanges() bool {
	return len(r.removed) > 0 ||
		len(r.changed) > 0 ||
		len(r.added) > 0
}

func (r CompareResult) String() string {
	if !r.hasChanges() {
		return "No removed, changed or added files found"
	}

	builder := strings.Builder{}
	if len(r.removed) > 0 {
		builder.WriteString("removed files:\n")
		for _, pth := range r.removed {
			builder.WriteString(fmt.Sprintf("- %s\n", pth))
		}
	}
	if len(r.changed) > 0 {
		builder.WriteString("changed files:\n")
		for _, pth := range r.changed {
			builder.WriteString(fmt.Sprintf("- %s\n", pth))
		}
	}
	if len(r.added) > 0 {
		builder.WriteString("added files:\n")
		for _, pth := range r.added {
			builder.WriteString(fmt.Sprintf("- %s\n", pth))
		}
	}
	return builder.String()
}

func sameZips(aZip, bZip string) (bool, error) {
	aDescriptor, err := newZipDescriptor(aZip)
	if err != nil {
		return false, err
	}

	bDescriptor, err := newZipDescriptor(bZip)
	if err != nil {
		return false, err
	}

	result := compareZipDescriptors(aDescriptor, bDescriptor)
	hasChanges := result.hasChanges()
	if hasChanges {
		log.Debugf("%s and %s are not the same:\n%s", aZip, bZip, result)
	}
	return !hasChanges, nil
}

func newZipDescriptor(pth string) (map[string]ZipFileInfo, error) {
	reader, err := zip.OpenReader(pth)
	if err != nil {
		return nil, err
	}

	descriptor := make(map[string]ZipFileInfo, len(reader.File))

	for _, f := range reader.File {
		info := ZipFileInfo{
			UncompressedSize64: f.FileHeader.UncompressedSize64,
			CRC32:              f.FileHeader.CRC32,
			Name:               f.FileHeader.Name,
		}

		descriptor[info.Name] = info
	}

	return descriptor, nil
}

func compareZipDescriptors(aDescriptor, bDescriptor map[string]ZipFileInfo) CompareResult {
	bDescriptorCopy := make(map[string]ZipFileInfo, len(bDescriptor))
	for k, v := range bDescriptor {
		bDescriptorCopy[k] = v
	}

	var result CompareResult
	for aPth, aInfo := range aDescriptor {
		bInfo, ok := bDescriptorCopy[aPth]
		if !ok {
			result.removed = append(result.removed, aPth)
			continue
		}

		if aInfo.equals(bInfo) {
			result.matching = append(result.matching, aPth)
		} else {
			result.changed = append(result.changed, aPth)
		}

		delete(bDescriptorCopy, aPth)
	}

	for bPth := range bDescriptorCopy {
		result.added = append(result.added, bPth)
	}

	return result
}
