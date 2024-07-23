package uploaders

import (
	"math"
	"os"
)

func round(f float64) float64 {
	return math.Floor(f + .5)
}

func roundPlus(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return round(f*shift) / shift
}

func fileSizeInBytes(pth string) (int64, error) {
	finfo, err := os.Stat(pth)
	if err != nil {
		return 0, err
	}
	return finfo.Size(), nil
}
