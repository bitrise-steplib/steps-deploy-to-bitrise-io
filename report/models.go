package report

type Report struct {
	Name   string
	Assets []Asset
}

type Asset struct {
	Path                string
	TestDirRelativePath string
	FileSize            int64
	ContentType         string
}

type ServerReport struct {
	Identifier string
	AssetURLs  map[string]string
}
