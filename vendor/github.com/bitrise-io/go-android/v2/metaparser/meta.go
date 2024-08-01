package parser

import "github.com/bitrise-io/go-android/v2/metaparser/androidartifact"

type ArtifactMetadata struct {
	AppInfo        androidartifact.Info `json:"app_info"`
	FileSizeBytes  int64                `json:"file_size_bytes"`
	Module         string               `json:"module"`
	ProductFlavour string               `json:"product_flavour"`
	BuildType      string               `json:"build_type"`
	SignedBy       string               `json:"signed_by"`
	Warnings       []string             `json:"warnings"`
	androidartifact.Artifact
}
