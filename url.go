package util

import (
	"fmt"
	"net/url"
)

// ValidateHTTPURL parses rawURL and ensures it has an http or https scheme
// and a non-empty host. Returns the parsed URL or an error.
func ValidateHTTPURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q in %q", u.Scheme, rawURL)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("missing host in URL %q", rawURL)
	}
	return u, nil
}
