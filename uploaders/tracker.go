package uploaders

import (
	"github.com/bitrise-io/go-utils/v2/analytics"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

type tracker struct {
	tracker analytics.Tracker
}

func newTracker(envRepo env.Repository, logger log.Logger) tracker {
	p := analytics.Properties{
		"step_id":    "deploy-to-bitrise-io",
		"build_slug": envRepo.Get("BITRISE_BUILD_SLUG"),
		"app_slug":   envRepo.Get("BITRISE_APP_SLUG"),
	}
	return tracker{
		tracker: analytics.NewDefaultTracker(logger, p),
	}
}

func (t *tracker) logFileTransfer(details TransferDetails, err error, intermediateFileTransfer bool) {
	properties := analytics.Properties{
		"storage_host": details.Hostname,
		"duration_ms":  details.Duration.Milliseconds(),
		"size_bytes":   details.Size,
	}
	if err != nil {
		properties["error"] = err.Error()
	}

	eventName := "artifact_uploaded"
	if intermediateFileTransfer {
		eventName = "intermediate_file_uploaded"
	}

	t.tracker.Enqueue(eventName, properties)
}

func (t *tracker) wait() {
	t.tracker.Wait()
}
