package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// conflictMaxAttempts is the number of times Delete retries on HTTP 409 Conflict.
// Copied from aria provider internal/provider/utils_client_core.go DeleteIt().
const conflictMaxAttempts = 60

// Client wraps resty with auth and structured logging following aria conventions.
type Client struct {
	host     string
	token    string
	insecure bool
	resty    *resty.Client
	okLevel  string
	koLevel  string
}

// NewClient creates a resty-backed Client configured with auth and TLS settings.
// No global HTTP timeout is set; context deadlines are the sole enforcement mechanism.
func NewClient(host, token string, insecure bool, okLevel, koLevel string) (*Client, error) {
	if okLevel == "" {
		okLevel = "TRACE"
	}
	if koLevel == "" {
		koLevel = "ERROR"
	}

	rc := resty.New()
	rc.SetBaseURL(host)
	rc.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: insecure}) //nolint:gosec
	if token != "" {
		rc.SetAuthToken(token)
	}

	return &Client{
		host:     host,
		token:    token,
		insecure: insecure,
		resty:    rc,
		okLevel:  okLevel,
		koLevel:  koLevel,
	}, nil
}

// Create POSTs body to path and returns the decoded response.
func (self *Client) Create(
	ctx context.Context,
	path string,
	body map[string]any,
) (map[string]any, error) {
	var result map[string]any
	resp, err := self.resty.R().SetContext(ctx).SetBody(body).SetResult(&result).Post(path)
	if err = self.handle(ctx, resp, err, http.StatusCreated, http.StatusOK); err != nil {
		return nil, fmt.Errorf("POST %s: %w", path, err)
	}
	return result, nil
}

// Read GETs path and returns the decoded response. Returns found=false on 404.
// readTimeout is applied as a per-call sub-context deadline (bounded by the parent ctx deadline).
func (self *Client) Read(ctx context.Context, path string, readTimeout time.Duration) (map[string]any, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	var result map[string]any
	resp, err := self.resty.R().SetContext(ctx).SetResult(&result).Get(path)
	if err != nil {
		return nil, false, fmt.Errorf("GET %s: %w", path, err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		self.logCall(ctx, self.okLevel, resp, nil)
		return nil, false, nil
	}
	if err = self.handle(ctx, resp, err, http.StatusOK); err != nil {
		return nil, false, fmt.Errorf("GET %s: %w", path, err)
	}
	return result, true, nil
}

// Update sends body to path using the given HTTP method (PUT or PATCH).
func (self *Client) Update(
	ctx context.Context,
	path, method string,
	body map[string]any,
) (map[string]any, error) {
	var result map[string]any
	resp, err := self.resty.R().SetContext(ctx).SetBody(body).SetResult(&result).Execute(method, path)
	if err = self.handle(ctx, resp, err, http.StatusOK); err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	return result, nil
}

// List GETs the collection path and returns items as a slice of maps.
// Handles both a direct JSON array response and an object wrapping an array in any of its top-level
// keys (e.g. {"results": [...]} or {"items": [...]}).
// readTimeout is applied as a per-call sub-context deadline (bounded by the parent ctx deadline).
func (self *Client) List(ctx context.Context, path string, readTimeout time.Duration) ([]map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	var raw any
	resp, err := self.resty.R().SetContext(ctx).SetResult(&raw).Get(path)
	if err = self.handle(ctx, resp, err, http.StatusOK); err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	return extractList(raw), nil
}

// extractList normalises a decoded JSON value into a flat slice of maps.
func extractList(v any) []map[string]any {
	switch t := v.(type) {
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		for _, val := range t {
			if arr, ok := val.([]any); ok {
				out := make([]map[string]any, 0, len(arr))
				for _, item := range arr {
					if m, ok := item.(map[string]any); ok {
						out = append(out, m)
					}
				}
				return out
			}
		}
	}
	return nil
}

// Delete sends a DELETE request to path, retrying on 409 Conflict and polling until gone.
// The outer ctx carries the delete operation deadline; readTimeout bounds each individual
// polling GET so one hung request cannot exhaust the whole delete budget.
// All sleeps respect context cancellation so the x-timeout deadline is honoured.
// Copied/modified from aria provider internal/provider/utils_client_core.go DeleteIt().
func (self *Client) Delete(ctx context.Context, path string, readTimeout time.Duration) error {
	for attempt := 0; attempt <= conflictMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("DELETE %s: %w", path, err)
		}
		resp, err := self.resty.R().SetContext(ctx).Delete(path)
		if err = self.handle(ctx, resp, err, http.StatusNoContent, http.StatusOK); err != nil {
			if attempt < conflictMaxAttempts && resp != nil &&
				resp.StatusCode() == http.StatusConflict {
				select {
				case <-ctx.Done():
					return fmt.Errorf("DELETE %s: %w", path, ctx.Err())
				case <-time.After(3 * time.Second):
				}
				continue
			}
			return fmt.Errorf("DELETE %s: %w", path, err)
		}

		for retry := range []int{0, 1, 2, 3, 4} {
			if retry > 0 {
				select {
				case <-ctx.Done():
					return fmt.Errorf("DELETE %s poll: %w", path, ctx.Err())
				case <-time.After(time.Duration(retry) * time.Second):
				}
			}
			_, found, err := self.Read(ctx, path, readTimeout)
			if err != nil {
				return fmt.Errorf("DELETE %s poll: %w", path, err)
			}
			if !found {
				return nil
			}
		}
	}
	return fmt.Errorf(
		"DELETE %s: resource still present after %d attempts",
		path,
		conflictMaxAttempts)
}

// handle checks the response status against expected codes, logs the call, and returns a
// descriptive error if the status does not match.
func (self *Client) handle(
	ctx context.Context,
	resp *resty.Response,
	err error,
	expected ...int,
) error {
	if err != nil {
		self.logCall(ctx, self.koLevel, resp, err)
		return err
	}
	if !codeIn(resp.StatusCode(), expected) {
		err = fmt.Errorf("HTTP %d: %s", resp.StatusCode(), resp.String())
	}
	level := self.okLevel
	if err != nil {
		level = self.koLevel
	}
	self.logCall(ctx, level, resp, err)
	return err
}

// logCall emits a structured log entry for an API call at the given level.
// Copied/modified from aria provider internal/provider/utils_client_core.go LogAPIResponseInfo().
func (self *Client) logCall(ctx context.Context, level string, resp *resty.Response, err error) {
	if resp == nil {
		fields := map[string]any{"error": fmt.Sprintf("%v", err)}
		tflog.Error(ctx, "API call failed (no response)", fields)
		return
	}

	req := resp.Request
	requestBody, reqErr := json.MarshalIndent(req.Body, "", "\t")
	if reqErr != nil {
		requestBody = []byte("<body>")
	}
	requestBody = redactJSON(requestBody)
	responseBody := redactJSON(resp.Body())

	errStr := "<nil>"
	if err != nil {
		errStr = err.Error()
	}

	msg := strings.Join([]string{
		"",
		"Request Info:",
		fmt.Sprintf("  URL         : %s", req.URL),
		fmt.Sprintf("  Method      : %s", req.Method),
		fmt.Sprintf("  Body        : %s", requestBody),
		"Response Info:",
		fmt.Sprintf("  Error       : %s", errStr),
		fmt.Sprintf("  Status Code : %d", resp.StatusCode()),
		fmt.Sprintf("  Status      : %s", resp.Status()),
		fmt.Sprintf("  Time        : %s", resp.Time()),
		fmt.Sprintf("  Body        : %s", responseBody),
	}, "\n")

	switch level {
	case "DEBUG":
		tflog.Debug(ctx, msg)
	case "INFO":
		tflog.Info(ctx, msg)
	case "WARN":
		tflog.Warn(ctx, msg)
	case "ERROR":
		tflog.Error(ctx, msg)
	default:
		tflog.Trace(ctx, msg)
	}
}
