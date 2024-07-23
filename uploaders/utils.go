package uploaders

import (
	"os"
)

func fileSizeInBytes(pth string) (int64, error) {
	finfo, err := os.Stat(pth)
	if err != nil {
		return 0, err
	}
	return finfo.Size(), nil
}
