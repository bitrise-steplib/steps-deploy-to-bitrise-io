package junitxml

import "os"

type resultReader interface {
	ReadAll() ([]byte, error)
}

type fileReader struct {
	Filename string
}

func (r *fileReader) ReadAll() ([]byte, error) {
	return os.ReadFile(r.Filename)
}

type stringReader struct {
	Contents string
}

func (r *stringReader) ReadAll() ([]byte, error) {
	return []byte(r.Contents), nil
}

// Converter holds data of the converter
type Converter struct {
	results []resultReader
}
