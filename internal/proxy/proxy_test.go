package proxy

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReverseProxy(t *testing.T) {
	// Create a test backend server.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Hello from the test server!"))
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	}))
	defer testServer.Close()

	// Create a reverse proxy with the test backend server as target.
	proxy := newHandler(testServer.URL)

	// Create a test frontend server using the reverse proxy as handler.
	frontendServer := httptest.NewServer(proxy)
	defer frontendServer.Close()

	// Send a request to the frontend server.
	resp, err := http.Get(frontendServer.URL)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	// Check the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if string(body) != "Hello from the test server!" {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestReverseProxyWithTimeout(t *testing.T) {
	// Create a test backend server.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait for 1 second before responding.
		time.Sleep(1 * time.Second)
		_, err := w.Write([]byte("Delayed hello from the test server!"))
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	}))
	defer testServer.Close()

	// Create a reverse proxy with the test backend server as target and a timeout of 100ms.
	proxy := newHandler(testServer.URL, withTimeout(100*time.Millisecond))

	// Create a test frontend server using the reverse proxy as handler.
	frontendServer := httptest.NewServer(proxy)
	defer frontendServer.Close()

	// Send a request to the frontend server.
	_, err := http.Get(frontendServer.URL)

	// Check that the request timed out.
	if err == nil || !isTimeoutError(err) {
		t.Errorf("expected timeout error, got %v", err)
	}
}

func isTimeoutError(err error) bool {
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

func TestReverseProxyWithHeaders(t *testing.T) {
	// Create a test backend server.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the request has the expected headers.
		if r.Header.Get("Connection") != "" {
			t.Errorf("unexpected hop-by-hop header: %s", r.Header.Get("Connection"))
		}
		if r.Host == "test.example.com" {
			t.Errorf("unexpected host header: %s", r.Host)
		}
		if r.Header.Get("X-Test-Header") != "test value" {
			t.Errorf("unexpected header: %s", r.Header.Get("X-Test-Header"))
		}
		_, err := w.Write([]byte("Hello from the test server!"))
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	}))
	defer testServer.Close()

	// Create a reverse proxy with the test backend server as target.
	proxy := newHandler(testServer.URL)

	// Create a test frontend server using the reverse proxy as handler.
	frontendServer := httptest.NewServer(proxy)
	defer frontendServer.Close()

	// Send a request to the frontend server.
	req, err := http.NewRequest(http.MethodGet, frontendServer.URL, nil)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	req.Header.Set("Connection", "test value")
	req.Header.Set("X-Test-Header", "test value")
	req.Host = "test.example.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	// Check the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if string(body) != "Hello from the test server!" {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestCopyHeaders(t *testing.T) {
	// Create some headers.
	fakeHeaders := http.Header{
		"Connection":          []string{"test Connection value"},
		"Keep-Alive":          []string{"test Keep-Alive value"},
		"Proxy-Authenticate":  []string{"test Proxy-Authenticate value"},
		"Proxy-Authorization": []string{"test Proxy-Authorization value"},
		"TE":                  []string{"test TE value"},
		"Trailers":            []string{"test Trailers value"},
		"Transfer-Encoding":   []string{"test Transfer-Encoding value"},
		"Upgrade":             []string{"test Upgrade value"},
		"X-Test-Header":       []string{"test X-Test-Header value"},
		"X-Some-Other-Header": []string{"test X-Some-Other-Header value"},
	}

	// Copy the request headers and remove hop-by-hop headers.
	result := http.Header{}
	copyHeaders(fakeHeaders, result)

	// Check that the headers were copied.
	if result.Get("X-Test-Header") != "test X-Test-Header value" {
		t.Errorf("unexpected header: %s", result.Get("X-Test-Header"))
	}
	if result.Get("X-Some-Other-Header") != "test X-Some-Other-Header value" {
		t.Errorf("unexpected header: %s", result.Get("X-Some-Other-Header"))
	}

	// Check that hop-by-hop headers were removed.
	if result.Get("Connection") != "" {
		t.Errorf("unexpected header: %s", result.Get("Connection"))
	}
	if result.Get("Keep-Alive") != "" {
		t.Errorf("unexpected header: %s", result.Get("Keep-Alive"))
	}
	if result.Get("Proxy-Authenticate") != "" {
		t.Errorf("unexpected header: %s", result.Get("Proxy-Authenticate"))
	}
	if result.Get("Proxy-Authorization") != "" {
		t.Errorf("unexpected header: %s", result.Get("Proxy-Authorization"))
	}
	if result.Get("TE") != "" {
		t.Errorf("unexpected header: %s, %v", result.Get("TE"), result)
	}
	if result.Get("Trailers") != "" {
		t.Errorf("unexpected header: %s", result.Get("Trailers"))
	}
	if result.Get("Transfer-Encoding") != "" {
		t.Errorf("unexpected header: %s", result.Get("Transfer-Encoding"))
	}
	if result.Get("Upgrade") != "" {
		t.Errorf("unexpected header: %s", result.Get("Upgrade"))
	}
}
