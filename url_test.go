package util_test

import (
	"testing"

	"github.com/pitabwire/util"
)

func TestValidateHTTPURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{"valid https", "https://example.com/path", false},
		{"valid http", "http://localhost:8080/api", false},
		{"missing scheme", "example.com/path", true},
		{"ftp scheme", "ftp://example.com", true},
		{"empty string", "", true},
		{"missing host", "http:///path", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := util.ValidateHTTPURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHTTPURL(%q) error = %v, wantErr %v", tt.rawURL, err, tt.wantErr)
				return
			}
			if !tt.wantErr && u == nil {
				t.Errorf("ValidateHTTPURL(%q) returned nil URL without error", tt.rawURL)
			}
		})
	}
}
