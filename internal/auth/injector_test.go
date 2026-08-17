package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureAuthlibInjectorVerifiesCacheAndDownload(t *testing.T) {
	contents := []byte("verified jar")
	hash := sha256.Sum256(contents)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(contents) }))
	defer server.Close()
	// The generic helper intentionally refuses any origin other than the pinned GitHub release.
	_, err := ensureAuthlibInjector(context.Background(), t.TempDir(), server.Client(), server.URL, fmt.Sprintf("%x", hash))
	if err == nil {
		t.Fatal("expected untrusted origin rejection")
	}

	cache := t.TempDir()
	path := filepath.Join(cache, "authlib-injector-"+AuthlibInjectorVersion+".jar")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ensureAuthlibInjector(context.Background(), cache, nil, authlibInjectorURL, fmt.Sprintf("%x", hash))
	if err != nil || got != path {
		t.Fatalf("verified cache = %q, %v", got, err)
	}
}
