package httpReq

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Simple test handler that returns a success message
func testHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from %s", r.Host)
}

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		body   io.Reader
		wantErr bool
	}{
		{
			name:    "valid GET request",
			method:  "GET",
			url:     "http://example.localhost/test",
			body:    nil,
			wantErr: false,
		},
		{
			name:    "valid POST request",
			method:  "POST",
			url:     "http://api.localhost/data",
			body:    strings.NewReader("test data"),
			wantErr: false,
		},
		{
			name:    "invalid method",
			method:  "INVALID METHOD",
			url:     "http://test.localhost",
			body:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := NewRequest(tt.method, tt.url, tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && req == nil {
				t.Error("NewRequest() returned nil request without error")
			}
			if !tt.wantErr && req.Method != tt.method {
				t.Errorf("NewRequest() method = %v, want %v", req.Method, tt.method)
			}
			if !tt.wantErr && req.URL.String() != tt.url {
				t.Errorf("NewRequest() URL = %v, want %v", req.URL.String(), tt.url)
			}
		})
	}
}

func TestExecuteRequestWithLocalhostResolution(t *testing.T) {
	// Start a test server on a random port
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()

	// Extract the port from the test server URL
	serverURL := server.URL
	port := strings.Split(serverURL, ":")[2]

	tests := []struct {
		name     string
		url      string
		wantErr  bool
		wantHost string
	}{
		{
			name:     "localhost domain resolution",
			url:      fmt.Sprintf("http://test.localhost:%s/path", port),
			wantErr:  false,
			wantHost: fmt.Sprintf("test.localhost:%s", port),
		},
		{
			name:     "subdomain localhost resolution",
			url:      fmt.Sprintf("http://api.app.localhost:%s/data", port),
			wantErr:  false,
			wantHost: fmt.Sprintf("api.app.localhost:%s", port),
		},
		{
			name:     "normal domain should work",
			url:      fmt.Sprintf("http://127.0.0.1:%s/test", port),
			wantErr:  false,
			wantHost: fmt.Sprintf("127.0.0.1:%s", port),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := ExecuteRequestWithLocalhostResolution(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteRequestWithLocalhostResolution() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				defer resp.Body.Close()
				
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				expectedResponse := fmt.Sprintf("Hello from %s", tt.wantHost)
				if string(body) != expectedResponse {
					t.Errorf("Expected response '%s', got '%s'", expectedResponse, string(body))
				}
			}
		})
	}
}

func TestLocalhostResolutionWithRealServer(t *testing.T) {
	// This test demonstrates how to use with a server on a specific port
	// Note: To run on port 80, you would need root privileges
	
	// Create a server that we control
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Server running on %s, request from %s", r.Host, r.RemoteAddr)
	})

	server := &http.Server{
		Addr:    ":8080", // Using 8080 instead of 80 to avoid privilege issues
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server failed to start: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Test requests to .localhost domains
	testURLs := []string{
		"http://app.localhost:8080/",
		"http://api.localhost:8080/data",
		"http://test.sub.localhost:8080/path",
	}

	for _, url := range testURLs {
		t.Run(fmt.Sprintf("Testing %s", url), func(t *testing.T) {
			req, err := NewRequest("GET", url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := ExecuteRequestWithLocalhostResolution(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			t.Logf("Response from %s: %s", url, string(body))
		})
	}
}

// Benchmark test to measure performance impact of custom resolution
func BenchmarkExecuteRequestWithLocalhostResolution(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()

	port := strings.Split(server.URL, ":")[2]
	url := fmt.Sprintf("http://test.localhost:%s/", port)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := NewRequest("GET", url, nil)
		if err != nil {
			b.Fatal(err)
		}

		resp, err := ExecuteRequestWithLocalhostResolution(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
} 