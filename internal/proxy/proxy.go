package proxy

import (
	"io"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	client    *http.Client
	targetUrl string
}

// newHandler creates a new reverse proxy handler.
func newHandler(targetUrl string, changes ...func(*Handler)) *Handler {
	h := &Handler{
		client:    http.DefaultClient,
		targetUrl: targetUrl,
	}
	for _, change := range changes {
		change(h)
	}
	return h
}

// WithTimeout sets the timeout for the reverse proxy.
func withTimeout(timeout time.Duration) func(*Handler) {
	return func(p *Handler) {
		p.client.Timeout = timeout
	}
}

// ServeHTTP handles the request by forwarding it to the target URL.
func (p *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a new request to the target URL.
	req, err := http.NewRequest(r.Method, p.targetUrl+r.RequestURI, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy the request headers and remove hop-by-hop headers.
	copyHeaders(r.Header, req.Header)

	log.Println("Forwarding request to", req.URL)

	// Set the Host header to the target URL host.
	req.Host = req.URL.Host

	// Send the request.
	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy the response headers and body.
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// List of hop-by-hop headers. These headers must not be forwarded.
var hopByHopHeaders = map[string]struct{}{
	"Connection":          {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {}, // canonicalized version of "TE"
	"Trailers":            {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func copyHeaders(src, dst http.Header) {
	// Copy the headers.
	for name, values := range src {
		// Skip hop-by-hop headers.
		if !isHopByHop(name) {
			for _, value := range values {
				dst.Add(name, value)
			}
		}
	}
}

// isHopByHop checks if a header is a hop-by-hop header.
func isHopByHop(header string) bool {
	if _, ok := hopByHopHeaders[http.CanonicalHeaderKey(header)]; ok {
		return true
	}
	return false
}

// NewServer creates a new reverse proxy server.
func NewServer(targetURL string, address string, timeout time.Duration) *http.Server {
	return &http.Server{
		Handler: newHandler(targetURL, withTimeout(timeout)),
		Addr:    address,
	}
}
