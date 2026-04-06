package logger

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SplunkWriter implements io.Writer to send logs to Splunk HTTP Event Collector (HEC).
//
// SplunkWriter allows logs to be sent directly to Splunk for centralized log
// aggregation and analysis. It wraps log entries in Splunk's HEC event format
// and sends them via HTTP POST requests.
//
// Example usage:
//
//	splunkWriter := logger.NewSplunkWriter(
//	    "https://splunk.example.com:8088/services/collector",
//	    "your-hec-token-here",
//	)
//
//	// Use with zerolog MultiLevelWriter
//	multi := zerolog.MultiLevelWriter(os.Stdout, splunkWriter)
//	log := logger.New(multi, false)
//
//	// All logs now go to both stdout and Splunk
//	log.Info("Application started", map[string]interface{}{"version": "1.0"})
type SplunkWriter struct {
	Endpoint string
	Token    string
	Client   *http.Client
}

// NewSplunkWriter creates a new SplunkWriter configured for Splunk HTTP Event Collector.
//
// Parameters:
//   - endpoint: The full URL of the Splunk HEC endpoint (e.g., "https://splunk.example.com:8088/services/collector")
//   - token: The HEC authentication token (configured in Splunk)
//
// The writer uses a 5-second timeout for HTTP requests. If Splunk is unreachable
// or returns an error, Write operations will fail.
// a log entry to Splunk HTTP Event Collector.
//
// This method implements the io.Writer interface, allowing SplunkWriter to be
// used with any logger that supports io.Writer output. It wraps the log data
// in Splunk's HEC event format: {"event": <log_data>}
//
// The log entry (p) should be a JSON-formatted byte slice, typically produced
// by a structured logger like zerolog.
//
// Parameters:
//   - p: The log data to send (typically JSON-formatted)
//
// Returns:
//   - n: The number of bytes written (len(p) on success)
//   - err: Any error encountered during the HTTP request
//
// If Splunk returns a non-2xx status code, an error is returned with details
// from the response body.
//
// Example usage with zerolog:
//
//	splunkWriter := logger.NewSplunkWriter(hecEndpoint, hecToken)
//	multi := zerolog.MultiLevelWriter(os.Stdout, splunkWriter)
//	log := zerolog.New(multi).With().Timestamp().Logger()
//
//	log.Info().Msg("This goes to both stdout and Splunk")
//
// Example:
//
//	writer := logger.NewSplunkWriter(
//	    "https://splunk.example.com:8088/services/collector",
//	    "12345678-1234-1234-1234-123456789012",
//	)

func NewSplunkWriter(endpoint, token string) *SplunkWriter {
	return &SplunkWriter{
		Endpoint: endpoint,
		Token:    token,
		Client:   &http.Client{Timeout: 5 * time.Second},
	}
}

// Write sends the log entry to Splunk
func (sw *SplunkWriter) Write(p []byte) (n int, err error) {
	// Wrap log in Splunk HEC event format
	payload := fmt.Sprintf(`{"event":%s}`, p)

	req, err := http.NewRequest("POST", sw.Endpoint, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Splunk "+sw.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sw.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("splunk HEC error: %s", string(body))
	}

	return len(p), nil
}
