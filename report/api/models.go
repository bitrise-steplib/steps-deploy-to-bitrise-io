package api

type CreateReportParameters struct {
	Title  string              `json:"title"`
	Assets []CreateReportAsset `json:"assets"`
}

type CreateReportAsset struct {
	RelativePath string `json:"relative_path"`
	FileSize     int64  `json:"file_size_bytes"`
	ContentType  string `json:"content_type"`
}

type CreateReportResponse struct {
	Identifier string            `json:"id"`
	AssetURLs  []CreateReportURL `json:"assets"`
}

type CreateReportURL struct {
	RelativePath string `json:"relative_path"`
	URL          string `json:"upload_url"`
}
