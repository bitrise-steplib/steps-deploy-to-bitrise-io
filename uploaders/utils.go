package uploaders

import (
	"fmt"
	"math"

	"github.com/bitrise-io/go-utils/pathutil"
)

func round(f float64) float64 {
	return math.Floor(f + .5)
}

func roundPlus(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return round(f*shift) / shift
}

func fileSizeInBytes(pth string) (float64, error) {
	info, exist, err := pathutil.PathCheckAndInfos(pth)
	if err != nil {
		return 0, err
	}
	if !exist {
		return 0, fmt.Errorf("file not exist at: %s", pth)
	}
	return float64(info.Size()), nil
}
