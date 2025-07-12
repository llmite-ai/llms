package llmite

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// LoggingRoundTripper implements http.RoundTripper with logging
type LoggingRoundTripper struct {
	transport http.RoundTripper
	logger    *slog.Logger
	config    LoggingConfig
}

// LoggingConfig controls what gets logged
type LoggingConfig struct {
	LogHeaders      bool
	LogRequestBody  bool
	LogResponseBody bool
	MaxBodySize     int64 // Maximum body size to log in bytes
}

func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		LogHeaders:      true,
		LogRequestBody:  true,
		LogResponseBody: true,
		MaxBodySize:     1024, // Default 1KB max body logging
	}
}

// NewLoggingRoundTripper creates a new logging round tripper
func NewLoggingRoundTripper(transport http.RoundTripper, logger *slog.Logger, config LoggingConfig) *LoggingRoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if logger == nil {
		logger = slog.Default()
	}
	if config.MaxBodySize == 0 {
		config.MaxBodySize = 1024 // Default 1KB max body logging
	}

	return &LoggingRoundTripper{
		transport: transport,
		logger:    logger,
		config:    config,
	}
}

// NewDefaultLoggingRoundTripper creates a round tripper with sensible defaults
func NewDefaultLoggingRoundTripper() *LoggingRoundTripper {
	return NewLoggingRoundTripper(nil, nil, LoggingConfig{
		LogHeaders:      true,
		LogRequestBody:  true,
		LogResponseBody: true,
		MaxBodySize:     1024,
	})
}

// RoundTrip implements the http.RoundTripper interface
func (t *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	requestID := fmt.Sprintf("%d", start.UnixNano())

	// Clone the request to avoid modifying the original
	reqClone := req.Clone(req.Context())

	// Build request log attributes
	reqAttrs := []slog.Attr{
		slog.String("request_id", requestID),
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.String("user_agent", req.UserAgent()),
		slog.String("host", req.Host),
	}

	// Log request headers if enabled
	if t.config.LogHeaders && len(req.Header) > 0 {
		headers := make(map[string][]string)
		for k, v := range req.Header {
			headers[k] = v
		}
		reqAttrs = append(reqAttrs, slog.Any("headers", headers))
	}

	// Log request body if enabled
	if t.config.LogRequestBody && req.Body != nil {
		if bodyBytes, newBody, err := t.captureRequestBody(req.Body, t.config.MaxBodySize); err == nil {
			reqAttrs = append(reqAttrs, slog.String("body", string(bodyBytes)))
			reqClone.Body = newBody
		}
	}

	t.logger.LogAttrs(req.Context(), slog.LevelInfo, "HTTP request started", reqAttrs...)

	// Perform the actual request using the cloned request
	resp, err := t.transport.RoundTrip(reqClone)
	duration := time.Since(start)

	// Build base response attributes
	respAttrs := []slog.Attr{
		slog.String("request_id", requestID),
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Duration("duration", duration),
	}

	if err != nil {
		// Log error
		errorAttrs := append(respAttrs, slog.String("error", err.Error()))
		t.logger.LogAttrs(req.Context(), slog.LevelError, "HTTP request failed", errorAttrs...)
		return nil, err
	}

	// Add response-specific attributes
	respAttrs = append(respAttrs,
		slog.Int("status_code", resp.StatusCode),
		slog.String("status", resp.Status),
		slog.Int64("content_length", resp.ContentLength),
	)

	// Log response headers if enabled
	if t.config.LogHeaders && len(resp.Header) > 0 {
		headers := make(map[string][]string)
		for k, v := range resp.Header {
			headers[k] = v
		}
		respAttrs = append(respAttrs, slog.Any("response_headers", headers))
	}

	// Log response body if enabled
	if t.config.LogResponseBody && resp.Body != nil {
		if bodyBytes, newBody, err := t.captureResponseBody(resp.Body, t.config.MaxBodySize); err == nil {
			respAttrs = append(respAttrs, slog.String("response_body", string(bodyBytes)))
			resp.Body = newBody
		}
	}

	// Determine log level based on status code
	level := slog.LevelInfo
	if resp.StatusCode >= 400 {
		level = slog.LevelWarn
	}
	if resp.StatusCode >= 500 {
		level = slog.LevelError
	}

	t.logger.LogAttrs(req.Context(), level, "HTTP request completed", respAttrs...)

	return resp, nil
}

// captureRequestBody reads the request body for logging and returns a new body for the request
func (t *LoggingRoundTripper) captureRequestBody(body io.ReadCloser, maxSize int64) ([]byte, io.ReadCloser, error) {
	if body == nil {
		return nil, nil, nil
	}

	// Read the entire body
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}

	// Close the original body
	body.Close()

	// Truncate for logging if necessary
	logBytes := bodyBytes
	if int64(len(bodyBytes)) > maxSize {
		logBytes = bodyBytes[:maxSize]
	}

	// Create a new body with the full content for the actual request
	newBody := io.NopCloser(bytes.NewReader(bodyBytes))

	return logBytes, newBody, nil
}

// captureResponseBody reads the response body for logging and returns a new body for the response
func (t *LoggingRoundTripper) captureResponseBody(body io.ReadCloser, maxSize int64) ([]byte, io.ReadCloser, error) {
	if body == nil {
		return nil, nil, nil
	}

	// Read the entire body
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}

	// Close the original body
	body.Close()

	// Truncate for logging if necessary
	logBytes := bodyBytes
	if int64(len(bodyBytes)) > maxSize {
		logBytes = bodyBytes[:maxSize]
	}

	// Create a new body with the full content for the caller
	newBody := io.NopCloser(bytes.NewReader(bodyBytes))

	return logBytes, newBody, nil
}

// NewHTTPClientWithLogging creates an http.Client with logging transport
func NewHTTPClientWithLogging(logger *slog.Logger, config LoggingConfig) *http.Client {
	return &http.Client{
		Transport: NewLoggingRoundTripper(nil, logger, config),
	}
}

// NewDefaultHTTPClientWithLogging creates an http.Client with default logging configuration
func NewDefaultHTTPClientWithLogging() *http.Client {
	return &http.Client{
		Transport: NewDefaultLoggingRoundTripper(),
	}
}
