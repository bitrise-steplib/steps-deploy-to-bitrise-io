package report

// Report ...
type Report struct {
	Name   string
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
