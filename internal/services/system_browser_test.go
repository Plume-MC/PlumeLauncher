package services_test

import (
	"strings"
	"testing"

	"plumelauncher/internal/services"
)

func TestOpenBrowserURLRejectsUntrustedHost(t *testing.T) {
	err := (&services.SystemService{}).OpenBrowserURL("https://example.com/code")
	if err == nil {
		t.Fatal("OpenBrowserURL should reject an untrusted host")
	}
	if !strings.Contains(err.Error(), "Unable to open the Ely.by sign-in page") {
		t.Fatalf("error = %q", err)
	}
}
