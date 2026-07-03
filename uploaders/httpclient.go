package uploaders

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	logV2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/retryhttp"
	"github.com/hashicorp/go-retryablehttp"
)

const maxDebugDumpBodyBytes = 64 * 1024

// maxResponseBodyBytes bounds how much of a response body we'll ever buffer into memory.
// All expected responses (JSON task lists, upload acks) are well under this; it exists to
// stop a misbehaving endpoint (e.g. a proxy error page) from being read without limit.
const maxResponseBodyBytes = 10 * 1024 * 1024

type HTTPClient struct {
	logger logV2.Logger
	client *retryablehttp.Client
}

func NewHTTPClient(logger logV2.Logger) *HTTPClient {
	h := &HTTPClient{logger: logger}

	client := retryhttp.NewClient(logger)
	client.RetryMax = 2
	client.RetryWaitMin = 5 * time.Second
	client.RetryWaitMax = 5 * time.Second
	client.RequestLogHook = h.logRequest
	client.ResponseLogHook = h.logResponse
	h.client = client

	return h
}

func (h *HTTPClient) logRequest(_ retryablehttp.Logger, req *http.Request, retryNumber int) {
	if retryNumber > 0 {
		h.logger.Warnf("%d attempt failed, retrying: %s %s", retryNumber, req.Method, req.URL)
	}

	if req.ContentLength < 0 || req.ContentLength > maxDebugDumpBodyBytes {
		h.logger.Debugf("request: %s %s (body omitted, %d bytes)", req.Method, req.URL, req.ContentLength)
		return
	}
	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		h.logger.Debugf("request:\n%s", dump)
	}
}

func (h *HTTPClient) logResponse(_ retryablehttp.Logger, resp *http.Response) {
	// Response bodies here are always small (JSON or an empty upload ack), so always safe to dump.
	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		h.logger.Debugf("response:\n%s", dump)
	}
}

func (h *HTTPClient) doFormRequest(method, uri string, data url.Values) ([]byte, error) {
	req, err := retryablehttp.NewRequest(method, uri, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request (%s): %s", uri, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return h.doRequest(req)
}

func (h *HTTPClient) doRequest(req *retryablehttp.Request) ([]byte, error) {
	_, body, err := h.doRequestWithResponse(req)
	return body, err
}

func (h *HTTPClient) doRequestWithResponse(req *retryablehttp.Request) (*http.Response, []byte, error) {
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to perform request (%s), error: %s", req.URL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			h.logger.Warnf("Failed to close response body, error: %s", err)
		}
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body (%s), error: %s", req.URL, err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp, body, fmt.Errorf("non success status code: %d, url: %s, body: %s", resp.StatusCode, req.URL, body)
	}

	return resp, body, nil
}
