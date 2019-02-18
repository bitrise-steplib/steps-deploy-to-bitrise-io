package gradle

import (
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/ziputil"
)

// Artifact ...
type Artifact struct {
	Path string
	Name string
}

// Export ...
func (artifact Artifact) Export(destination string) error {
	return command.CopyFile(artifact.Path, filepath.Join(destination, artifact.Name))
}

// ExportZIP ...
func (artifact Artifact) ExportZIP(destination string) error {
	return ziputil.ZipDir(artifact.Path, filepath.Join(destination, artifact.Name), true)
}
