package bundletool

import (
	"github.com/bitrise-io/go-utils/log"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

// RetryLogAdaptor adapts the retryablehttp.Logger interface to the go-utils logger.
type RetryLogAdaptor struct{}

// Printf implements the retryablehttp.Logger interface
func (*RetryLogAdaptor) Printf(fmtStr string, vars ...interface{}) {
	log.Debugf(fmtStr, vars...)
}

// NewRetryableClient returns a retryable HTTP client
func NewRetryableClient() *retryablehttp.Client {
	client := retryablehttp.NewClient()
	client.Logger = &RetryLogAdaptor{}
	client.ErrorHandler = retryablehttp.PassthroughErrorHandler

	return client
}
