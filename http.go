package llmite

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type HTTPClientOptions struct {
	// LogRequests indicates whether to log HTTP requests.
	LogRequests bool
	// Logger is the logger to use for logging HTTP requests and responses.
	Logger *slog.Logger
	// Config is the configuration for logging HTTP requests and responses.
	Config *LoggingConfig
}

// NewHTTPClient creates an http.Client with the provided options
func NewHTTPClient(options HTTPClientOptions) *http.Client {
	if options.LogRequests == false && options.Logger == nil {
		return http.DefaultClient
	}

	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.Config == nil {
		options.Config = &LoggingConfig{
			LogHeaders:      true,
			LogRequestBody:  true,
			LogResponseBody: true,
			MaxBodySize:     1024, // Default 1KB max body logging
			StreamingLog:    false, // Disabled by default
		}
	}
	return &http.Client{
		Transport: NewLoggingRoundTripper(http.DefaultTransport, options.Logger, *options.Config),
	}
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

// NewStreamingHTTPClientWithLogging creates an http.Client with streaming logging enabled
func NewStreamingHTTPClientWithLogging(logger *slog.Logger, maxBodySize int64) *http.Client {
	if logger == nil {
		logger = slog.Default()
	}
	if maxBodySize == 0 {
		maxBodySize = 1024 * 4 // Default 4KB for streaming
	}
	
	return &http.Client{
		Transport: NewLoggingRoundTripper(nil, logger, LoggingConfig{
			LogHeaders:      true,
			LogRequestBody:  true,
			LogResponseBody: true,
			MaxBodySize:     maxBodySize,
			StreamingLog:    true,
		}),
	}
}

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
	StreamingLog    bool  // Enable streaming body logging (logs chunks as they're read)
}

func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		LogHeaders:      true,
		LogRequestBody:  true,
		LogResponseBody: true,
		MaxBodySize:     1024, // Default 1KB max body logging
		StreamingLog:    false, // Disabled by default for backward compatibility
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
		StreamingLog:    false,
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
		if t.config.StreamingLog {
			// For streaming responses, wrap the body with a logging reader
			resp.Body = t.newStreamingBodyLogger(resp.Body, requestID, t.config.MaxBodySize)
		} else {
			// For non-streaming responses, capture the entire body
			if bodyBytes, newBody, err := t.captureResponseBody(resp.Body, t.config.MaxBodySize); err == nil {
				respAttrs = append(respAttrs, slog.String("response_body", string(bodyBytes)))
				resp.Body = newBody
			}
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

// streamingBodyLogger wraps an io.ReadCloser to log chunks as they're read
type streamingBodyLogger struct {
	body      io.ReadCloser
	logger    *slog.Logger
	requestID string
	maxSize   int64
	totalRead int64
	buffer    *bytes.Buffer
}

// newStreamingBodyLogger creates a new streaming body logger
func (t *LoggingRoundTripper) newStreamingBodyLogger(body io.ReadCloser, requestID string, maxSize int64) io.ReadCloser {
	return &streamingBodyLogger{
		body:      body,
		logger:    t.logger,
		requestID: requestID,
		maxSize:   maxSize,
		buffer:    &bytes.Buffer{},
	}
}

// Read implements io.Reader and logs chunks as they're read
func (s *streamingBodyLogger) Read(p []byte) (int, error) {
	n, err := s.body.Read(p)
	
	if n > 0 {
		// Log the chunk if we haven't exceeded the max size
		if s.totalRead < s.maxSize {
			remaining := s.maxSize - s.totalRead
			toLog := int64(n)
			if toLog > remaining {
				toLog = remaining
			}
			
			s.buffer.Write(p[:toLog])
			s.totalRead += int64(n)
			
			// Log the chunk
			chunk := string(p[:n])
			if strings.Contains(chunk, "\n") {
				// For multi-line chunks (like SSE), log each line
				lines := strings.Split(strings.TrimRight(chunk, "\n"), "\n")
				for _, line := range lines {
					if line != "" {
						s.logger.LogAttrs(nil, slog.LevelDebug, "HTTP streaming chunk",
							slog.String("request_id", s.requestID),
							slog.String("chunk", line),
							slog.Int64("bytes_read", s.totalRead))
					}
				}
			} else {
				s.logger.LogAttrs(nil, slog.LevelDebug, "HTTP streaming chunk",
					slog.String("request_id", s.requestID),
					slog.String("chunk", chunk),
					slog.Int64("bytes_read", s.totalRead))
			}
		}
	}
	
	return n, err
}

// Close implements io.Closer and logs the final summary
func (s *streamingBodyLogger) Close() error {
	// Log final summary
	if s.buffer.Len() > 0 {
		s.logger.LogAttrs(nil, slog.LevelInfo, "HTTP streaming body complete",
			slog.String("request_id", s.requestID),
			slog.Int64("total_bytes", s.totalRead),
			slog.String("logged_content", s.buffer.String()))
	}
	
	return s.body.Close()
}
