package androidartifact

type Info struct {
	AppName           string `json:"app_name"`
	PackageName       string `json:"package_name"`
	VersionCode       string `json:"version_code"`
	VersionName       string `json:"version_name"`
	MinSDKVersion     string `json:"min_sdk_version"`
	RawPackageContent string `json:"-"`
}
