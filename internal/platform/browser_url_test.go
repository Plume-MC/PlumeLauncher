package platform

import "testing"

func TestValidateBrowserURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "ely by http", url: "http://account.ely.by/code", want: true},
		{name: "ely by https", url: "https://account.ely.by/code", want: true},
		{name: "wrong host", url: "https://example.com/code", want: false},
		{name: "lookalike host", url: "https://account.ely.by.example.com/code", want: false},
		{name: "host in user info", url: "https://account.ely.by@example.com/code", want: false},
		{name: "custom port", url: "https://account.ely.by:8443/code", want: false},
		{name: "javascript scheme", url: "javascript:alert(1)", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBrowserURL(tt.url)
			if (err == nil) != tt.want {
				t.Fatalf("validateBrowserURL(%q) error = %v, want valid = %t", tt.url, err, tt.want)
			}
		})
	}
}
