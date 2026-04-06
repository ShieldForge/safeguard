package logger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewSplunkWriter(t *testing.T) {
	sw := NewSplunkWriter("https://splunk.example.com:8088/services/collector", "test-token")
	if sw == nil {
		t.Fatal("NewSplunkWriter() returned nil")
	}
	if sw.Endpoint != "https://splunk.example.com:8088/services/collector" {
		t.Errorf("Endpoint = %v", sw.Endpoint)
	}
	if sw.Token != "test-token" {
		t.Errorf("Token = %v", sw.Token)
	}
	if sw.Client == nil {
		t.Error("Client is nil")
	}
}

func TestSplunkWriter_Write_Success(t *testing.T) {
	var receivedBody string
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		receivedBody = string(buf)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sw := NewSplunkWriter(server.URL, "my-hec-token")
	logData := []byte(`{"level":"info","message":"hello"}`)

	n, err := sw.Write(logData)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(logData) {
		t.Errorf("Write() returned %d, want %d", n, len(logData))
	}
	if receivedAuth != "Splunk my-hec-token" {
		t.Errorf("Authorization header = %v, want 'Splunk my-hec-token'", receivedAuth)
	}
	if !strings.Contains(receivedBody, `"event":`) {
		t.Errorf("Body should wrap in HEC event format, got: %s", receivedBody)
	}
}

func TestSplunkWriter_Write_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	sw := NewSplunkWriter(server.URL, "my-token")
	_, err := sw.Write([]byte(`{"msg":"test"}`))
	if err == nil {
		t.Error("Write() expected error for non-2xx status")
	}
	if !strings.Contains(err.Error(), "splunk HEC error") {
		t.Errorf("Error should mention HEC error, got: %v", err)
	}
}

func TestSplunkWriter_Write_ConnectionError(t *testing.T) {
	sw := NewSplunkWriter("http://localhost:1", "token")
	_, err := sw.Write([]byte(`{"msg":"test"}`))
	if err == nil {
		t.Error("Write() expected error for connection failure")
	}
}
