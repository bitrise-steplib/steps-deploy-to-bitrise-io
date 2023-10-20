package api

// CreateReportParameters ...
type CreateReportParameters struct {
	Title  string              `json:"title"`
	Assets []CreateReportAsset `json:"assets"`
}

// CreateReportAsset ...
type CreateReportAsset struct {
	RelativePath string `json:"relative_path"`
	FileSize     int64  `json:"file_size_bytes"`
	ContentType  string `json:"content_type"`
}

// CreateReportResponse ...
type CreateReportResponse struct {
	Identifier string            `json:"id"`
	AssetURLs  []CreateReportURL `json:"assets"`
}

// CreateReportURL ...
type CreateReportURL struct {
	RelativePath string `json:"relative_path"`
	URL          string `json:"upload_url"`
}
