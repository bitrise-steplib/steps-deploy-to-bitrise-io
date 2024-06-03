package androidartifact

import (
	"fmt"
	"regexp"
)

// ParsePackageInfo parses package name, version code and name from the input string.
func ParsePackageInfo(input string, packageNameKey string) (string, string, string) {
	return parsePackageField(input, packageNameKey),
		parsePackageField(input, "versionCode"),
		parsePackageField(input, "versionName")
}

func parsePackageField(input, key string) string {
	pattern := fmt.Sprintf(`%s=['"](.*?)['"]`, key)
	return getByPattern(input, pattern)
}

func getByPattern(input string, pattern string) string {
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(input); len(matches) == 2 {
		return matches[1]
	}
	return ""
}
