package uploaders

import (
	"github.com/bitrise-io/go-utils/v2/analytics"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

type TransferType int

const (
	Intermediate TransferType = iota
	Artifact
)

type tracker struct {
	tracker analytics.Tracker
	logger  log.Logger
}

func newTracker(envRepo env.Repository, logger log.Logger) tracker {
	p := analytics.Properties{
		"step_id":    "deploy-to-bitrise-io",
		"build_slug": envRepo.Get("BITRISE_BUILD_SLUG"),
		"app_slug":   envRepo.Get("BITRISE_APP_SLUG"),
	}
	return tracker{
		tracker: analytics.NewDefaultTracker(logger, envRepo, p),
		logger:  logger,
	}
}

func (t *tracker) logFileTransfer(transferType TransferType, details TransferDetails, err error, isArtifact, isIntermediateFile bool) {
	properties := analytics.Properties{
		"storage_host": details.Hostname,
		"duration_ms":  details.Duration.Milliseconds(),
		"size_bytes":   details.Size,
	}
	if err != nil {
		properties["error"] = err.Error()
	}

	var eventName string
	switch transferType {
	case Intermediate:
		eventName = "intermediate_file_uploaded"
		properties["is_artifact"] = isArtifact
	case Artifact:
		eventName = "artifact_uploaded"
		properties["is_intermediate_file"] = isIntermediateFile
	default:
		t.logger.Warnf("Unknown transfer type: %d", transferType)
		return
	}

	t.tracker.Enqueue(eventName, properties)
}

func (t *tracker) wait() {
	t.tracker.Wait()
}
