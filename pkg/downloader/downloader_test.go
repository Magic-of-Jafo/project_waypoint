package downloader

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"waypoint_archive_scripts/pkg/config"

	"golang.org/x/text/encoding/charmap"
)

// Helper function to create a mock server
func newMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// Helper function to create a default config for tests
func newTestConfig() *config.Config {
	return &config.Config{
		UserAgent:       "TestAgent/1.0",
		PolitenessDelay: 0, // No delay for tests unless specifically testing delay
	}
}

func TestMain(m *testing.M) {
	// Disable logging output for tests to keep test output clean
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestFetchPage_SuccessfulDownload_UTF8(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "TestAgent/1.0" {
			t.Errorf("Expected User-Agent 'TestAgent/1.0', got '%s'", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><body>Hello, UTF-8!</body></html>")
	})
	defer server.Close()

	cfg := newTestConfig()
	d := NewDownloader(cfg)

	content, err := d.FetchPage(server.URL)
	if err != nil {
		t.Fatalf("FetchPage failed: %v", err)
	}

	expectedContent := "<html><body>Hello, UTF-8!</body></html>"
	if string(content) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, string(content))
	}
}

func TestFetchPage_SuccessfulDownload_NonUTF8Encoding(t *testing.T) {
	// ISO-8859-1 is a common non-UTF-8 encoding
	rawISOBytes := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x2c, 0x20, 0x49, 0x53, 0x4f, 0x2d, 0x38, 0x38, 0x35, 0x39, 0x2d, 0x31, 0x21, 0xA1} // "Hello, ISO-8859-1!¡"
	// Convert ISO-8859-1 bytes to UTF-8 for comparison
	utf8Reader := bytes.NewReader(rawISOBytes)
	decodedReader := charmap.ISO8859_1.NewDecoder().Reader(utf8Reader)
	expectedUTF8Bytes, _ := io.ReadAll(decodedReader)

	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=iso-8859-1")
		w.Write(rawISOBytes)
	})
	defer server.Close()

	cfg := newTestConfig()
	d := NewDownloader(cfg)

	content, err := d.FetchPage(server.URL)
	if err != nil {
		t.Fatalf("FetchPage failed: %v", err)
	}

	if !bytes.Equal(content, expectedUTF8Bytes) {
		t.Errorf("Expected content (decoded to UTF-8) '%s', got '%s'", string(expectedUTF8Bytes), string(content))
	}
}

func TestFetchPage_SuccessfulDownload_NoCharsetHeader(t *testing.T) {
	// Should default to UTF-8 or pass through raw bytes if undecipherable (which for plain ASCII is fine)
	expectedContent := "<html><body>Hello, No Charset!</body></html>"
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html") // No charset specified
		fmt.Fprint(w, expectedContent)
	})
	defer server.Close()

	cfg := newTestConfig()
	d := NewDownloader(cfg)

	content, err := d.FetchPage(server.URL)
	if err != nil {
		t.Fatalf("FetchPage failed: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, string(content))
	}
}

func TestFetchPage_SuccessfulDownload_UncertainEncodingRawBytes(t *testing.T) {
	// Simulate an unusual content type that charset.DetermineEncoding might be uncertain about
	// but where raw bytes should be preserved.
	// Using a known sequence of bytes that might be problematic if misinterepreted.
	// For this test, we'll send bytes that ARE valid UTF-8, but with a Content-Type
	// that might make the library uncertain.
	rawBytes := []byte("<html><title>Test Âøñ</title></html>") // Contains some multi-byte UTF-8 chars

	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream-variant") // Made-up, likely uncertain
		w.Write(rawBytes)
	})
	defer server.Close()

	cfg := newTestConfig()
	d := NewDownloader(cfg)

	// Capture log output to verify behavior for uncertain encoding
	var logBuffer bytes.Buffer
	originalLogOutput := log.Writer()      // Save current log output
	log.SetOutput(&logBuffer)              // Redirect log to buffer for this test
	defer log.SetOutput(originalLogOutput) // Restore original log output (which is io.Discard due to TestMain)

	content, err := d.FetchPage(server.URL)
	if err != nil {
		t.Fatalf("FetchPage failed: %v", err)
	}

	if !bytes.Equal(content, rawBytes) {
		t.Errorf("Expected raw bytes to be preserved. Expected %v, got %v", rawBytes, content)
		t.Errorf("Expected content (string): %s, got: %s", string(rawBytes), string(content))
	}

	logOutput := logBuffer.String()
	// Check if the log indicates uncertainty and attempt to read raw bytes
	if !(strings.Contains(logOutput, "is uncertain") && strings.Contains(logOutput, "Attempting to read raw bytes.")) {
		t.Errorf("Expected log message about uncertain encoding and reading raw bytes, but not found. Log: %s", logOutput)
	}
}

func TestFetchPage_HTTPError_404(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Not Found Here")
	})
	defer server.Close()

	cfg := newTestConfig()
	d := NewDownloader(cfg)

	_, err := d.FetchPage(server.URL)
	if err == nil {
		t.Fatal("Expected an error for 404 status, got nil")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("Expected HTTPError type, got %T: %v", err, err)
	}

	if httpErr.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, httpErr.StatusCode)
	}
	if httpErr.URL != server.URL {
		t.Errorf("Expected URL '%s', got '%s'", server.URL, httpErr.URL)
	}
}

func TestFetchPage_HTTPError_500(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	cfg := newTestConfig()
	d := NewDownloader(cfg)

	_, err := d.FetchPage(server.URL)
	if err == nil {
		t.Fatal("Expected an error for 500 status, got nil")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("Expected HTTPError type, got %T", err)
	}

	if httpErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, httpErr.StatusCode)
	}
}

func TestFetchPage_NetworkError_ConnectionRefused(t *testing.T) {
	// Attempt to connect to a port that is not listening
	cfg := newTestConfig()
	d := NewDownloader(cfg)

	// Find a free port, then close the listener immediately so the FetchPage fails.
	// Wrap with a standard Go import
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find a free port: %v", err)
	}
	address := listener.Addr().String()
	listener.Close() // Ensure port is not listening

	// Capture log output to prevent it from cluttering test results for this expected error
	var logBuffer bytes.Buffer
	originalLogOutput := log.Writer()
	log.SetOutput(&logBuffer)
	defer log.SetOutput(originalLogOutput)

	_, err = d.FetchPage("http://" + address)
	if err == nil {
		t.Fatal("Expected a network error, got nil")
	}

	// We expect some kind of network error (e.g., syscall.ECONNREFUSED on Linux/macOS, or similar on Windows)
	// Checking for *HTTPError should fail
	if _, ok := err.(*HTTPError); ok {
		t.Errorf("Expected a non-HTTPError network error, but got an HTTPError: %v", err)
	}
	// A more robust check might involve checking the error string for parts of "connection refused" or similar,
	// but this can be OS-dependent. For now, ensuring it's not nil and not an HTTPError is a good start.
	t.Logf("Received expected network error: %v", err) // Log for visibility
}

func TestFetchPage_PolitenessDelay(t *testing.T) {
	delay := 100 * time.Millisecond
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})
	defer server.Close()

	cfg := newTestConfig()
	cfg.PolitenessDelay = delay // Set a specific delay
	d := NewDownloader(cfg)

	startTime := time.Now()
	_, err := d.FetchPage(server.URL)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("FetchPage failed: %v", err)
	}

	// Check if the duration is at least the politeness delay
	// Allow for a small margin for execution time
	if duration < delay {
		t.Errorf("Expected delay of at least %v, but got %v", delay, duration)
	}
	t.Logf("Politeness delay test: requested %v, actual call duration %v", delay, duration)
}

func TestFetchPage_CustomUserAgent(t *testing.T) {
	customUA := "MyCustomTestAgent/2.0"
	var receivedUA string

	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		fmt.Fprint(w, "OK")
	})
	defer server.Close()

	cfg := newTestConfig()
	cfg.UserAgent = customUA // Set a custom User-Agent
	d := NewDownloader(cfg)

	_, err := d.FetchPage(server.URL)
	if err != nil {
		t.Fatalf("FetchPage failed: %v", err)
	}

	if receivedUA != customUA {
		t.Errorf("Expected User-Agent '%s', got '%s'", customUA, receivedUA)
	}
}

func TestNewDownloader(t *testing.T) {
	cfg := &config.Config{
		UserAgent:       "TestUA",
		PolitenessDelay: 5 * time.Second,
	}
	d := NewDownloader(cfg)

	if d.Client == nil {
		t.Error("Expected Client to be initialized, got nil")
	}
	if d.Client.Timeout != 30*time.Second { // Check default timeout set in NewDownloader
		t.Errorf("Expected client timeout of 30s, got %v", d.Client.Timeout)
	}
	if d.UserAgent != cfg.UserAgent {
		t.Errorf("Expected UserAgent '%s', got '%s'", cfg.UserAgent, d.UserAgent)
	}
	if d.PolitenessDelay != cfg.PolitenessDelay {
		t.Errorf("Expected PolitenessDelay %v, got %v", cfg.PolitenessDelay, d.PolitenessDelay)
	}
}
