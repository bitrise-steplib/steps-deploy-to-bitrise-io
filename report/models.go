package report

// Report ...
type Report struct {
	Name   string
	Info   Info
	Assets []Asset
}

// Asset ...
type Asset struct {
	Path                string
	TestDirRelativePath string
	FileSize            int64
	ContentType         string
}

// ServerReport ...
type ServerReport struct {
	Identifier string
	AssetURLs  map[string]string
}

// Info ...
type Info struct {
	Category string `json:"category"`
}
