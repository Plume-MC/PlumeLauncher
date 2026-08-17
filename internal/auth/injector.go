package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const AuthlibInjectorVersion = "1.2.7"
const AuthlibInjectorSHA256 = "eaf14bc5acffc7d885bd5bd5942b99f36d6299302beae356b2fc5807fe42652b"
const authlibInjectorURL = "https://github.com/yushijinhun/authlib-injector/releases/download/v1.2.7/authlib-injector-1.2.7.jar"

// EnsureAuthlibInjector returns a cache entry only after its pinned release hash verifies.
func EnsureAuthlibInjector(ctx context.Context, cacheRoot string, client *http.Client) (string, error) {
	return ensureAuthlibInjector(ctx, cacheRoot, client, authlibInjectorURL, AuthlibInjectorSHA256)
}

func ensureAuthlibInjector(ctx context.Context, cacheRoot string, client *http.Client, source, expectedSHA256 string) (string, error) {
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.Path != "/yushijinhun/authlib-injector/releases/download/v1.2.7/authlib-injector-1.2.7.jar" {
		return "", fmt.Errorf("untrusted authlib-injector origin")
	}
	path := filepath.Join(cacheRoot, "authlib-injector-"+AuthlibInjectorVersion+".jar")
	if err := verifySHA256(path, expectedSHA256); err == nil {
		return path, nil
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return "", err
	}
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download authlib-injector: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download authlib-injector: %s", response.Status)
	}
	temporary, err := os.CreateTemp(cacheRoot, ".authlib-injector-*.jar")
	if err != nil {
		return "", err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := io.Copy(temporary, response.Body); err != nil {
		temporary.Close()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	if err := verifySHA256(temporaryPath, expectedSHA256); err != nil {
		return "", fmt.Errorf("authlib-injector integrity check failed: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return "", err
	}
	return path, nil
}

func verifySHA256(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != expected {
		return fmt.Errorf("SHA-256 mismatch")
	}
	return nil
}
