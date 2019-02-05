package junitxml

import (
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
)

type Handler struct {
	files       []string
	xmlFilePath string
}

func (h *Handler) SetFiles(files []string) {
	h.files = files
}

func (h *Handler) Detect() bool {
	// TODO: detect more and validate if Junit4 xml
	for _, file := range h.files {
		if strings.HasSuffix(file, ".xml") {
			h.xmlFilePath = file
			return true
		}
	}
	return false
}

func (h *Handler) XML() ([]byte, error) {
	return fileutil.ReadBytesFromFile(h.xmlFilePath)
}
