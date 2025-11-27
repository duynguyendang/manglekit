package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
)

type Tool struct {
	opts Options
	deps diapi.CoreDeps
}

func NewTool(deps diapi.CoreDeps, opts Options) (*Tool, error) {
	return &Tool{
		opts: opts,
		deps: deps,
	}, nil
}

func (t *Tool) Execute(ctx context.Context, execCtx *core.ExecutionContext) error {
	var bodyReader io.Reader
	// Check if there's a body to send, typically for POST, PUT, PATCH.
	if bodyData, ok := execCtx.CurrentStepParams["body"]; ok && bodyData != nil {
		// Attempt to marshal the body data to JSON.
		jsonData, err := json.Marshal(bodyData)
		if err != nil {
			return fmt.Errorf("failed to marshal http tool request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(t.opts.Method), t.opts.Endpoint, bodyReader)
	if err != nil {
		return err
	}

	// Add default content-type header if a body is present.
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range t.opts.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if execCtx.Meta == nil {
		execCtx.Meta = make(map[string]any)
	}
	// Try to unmarshal the response as JSON, but fall back to string.
	var responseData any
	if err := json.Unmarshal(body, &responseData); err != nil {
		responseData = string(body)
	}

	execCtx.Meta["http_response"] = responseData
	execCtx.Meta["http_status_code"] = resp.StatusCode

	return nil
}
