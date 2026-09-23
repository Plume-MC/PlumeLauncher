package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Public profile/texture endpoints. Skin lookup never uses account tokens:
// session profile endpoints are unauthenticated by design (REQ-022).
var (
	msSessionProfileURL  = "https://sessionserver.mojang.com/session/minecraft/profile/"
	elySessionProfileURL = "https://authserver.ely.by/session/profile/"
	steveSkinURL         = "https://assets.mojang.com/SkinTemplates/steve.png"
)

// skinTextureHosts is the allowlist of hosts a texture URL may resolve to.
// Anything else is rejected instead of fetched (no open outbound requests).
// ely.by is the apex domain Ely.by uses to store custom skins
// (http://ely.by/storage/skins/<id>.png), verified live.
var skinTextureHosts = map[string]bool{
	"assets.mojang.com":      true,
	"textures.minecraft.net": true,
	"skinsystem.ely.by":      true,
	"ely.by":                 true,
}

const (
	skinCacheTTL  = 5 * time.Minute
	maxSkinBytes  = 1 << 20 // 1 MiB cap for a 64x64 PNG
	pngMagicLen   = 8
	dataURLPrefix = "data:image/png;base64,"
)

var pngMagic = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}

type skinCacheEntry struct {
	dataURL string
	expires time.Time
}

// skinCache keys rendered skin data URLs by account UUID so reopening the
// account dialog does not refetch every profile.
var skinCache sync.Map

// GetAccountSkin resolves the player's skin as a PNG data URL for the UI.
// Offline profiles resolve the official default Steve template; Ely.by and
// Microsoft profiles resolve their public session profile, then download the
// allowlisted texture host. Failures return a generic error so the UI can
// fall back to the initial-letter avatar.
func (s *AccountService) GetAccountSkin(accountUUID string) (string, error) {
	if cached, ok := skinCache.Load(accountUUID); ok {
		entry := cached.(skinCacheEntry)
		if time.Now().Before(entry.expires) {
			return entry.dataURL, nil
		}
		skinCache.Delete(accountUUID)
	}

	accounts, err := s.loadAccounts()
	if err != nil {
		return "", NewInternalError("Unable to load accounts. Restart the launcher and try again.")
	}
	var account *Account
	for i := range accounts {
		if accounts[i].UUID == accountUUID {
			account = &accounts[i]
			break
		}
	}
	if account == nil {
		return "", NewNotFoundError("Account not found.")
	}

	png, err := s.resolveSkinPNG(account)
	if err != nil {
		return "", err
	}

	dataURL := dataURLPrefix + base64.StdEncoding.EncodeToString(png)
	skinCache.Store(accountUUID, skinCacheEntry{dataURL: dataURL, expires: time.Now().Add(skinCacheTTL)})
	return dataURL, nil
}

func (s *AccountService) resolveSkinPNG(account *Account) ([]byte, error) {
	if account.Type == AccountTypeOffline {
		return s.downloadPNG(steveSkinURL)
	}

	base := msSessionProfileURL
	if account.Type == AccountTypeElyBy {
		base = elySessionProfileURL
	}
	profileURL := base + strings.ReplaceAll(account.UUID, "-", "")

	request, err := http.NewRequest(http.MethodGet, profileURL, nil)
	if err != nil {
		return nil, skinUnavailableError()
	}
	response, err := s.skinHTTPClient().Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			response.Body.Close()
		}
		return s.downloadPNG(steveSkinURL) // no custom skin yet: default Steve
	}
	defer response.Body.Close()

	var profile struct {
		Properties []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(response.Body).Decode(&profile); err != nil {
		return nil, skinUnavailableError()
	}

	textureURL, err := skinURLFromProfile(profile.Properties)
	if err != nil {
		// Profile without a custom skin: the game shows default Steve too.
		return s.downloadPNG(steveSkinURL)
	}
	normalized, err := normalizeSkinURL(textureURL)
	if err != nil {
		return nil, skinUnavailableError()
	}
	png, err := s.downloadPNG(normalized)
	if err != nil {
		return s.downloadPNG(steveSkinURL)
	}
	return png, nil
}

// skinURLFromProfile decodes the base64 "textures" property of a Yggdrasil
// session profile into the SKIN texture URL.
func skinURLFromProfile(properties []struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}) (string, error) {
	for _, property := range properties {
		if property.Name != "textures" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(property.Value)
		if err != nil {
			return "", fmt.Errorf("decode textures property: %w", err)
		}
		var payload struct {
			Textures struct {
				SKIN *struct {
					URL string `json:"url"`
				} `json:"SKIN"`
			} `json:"textures"`
		}
		if err := json.Unmarshal(decoded, &payload); err != nil {
			return "", fmt.Errorf("unmarshal textures property: %w", err)
		}
		if payload.Textures.SKIN == nil || payload.Textures.SKIN.URL == "" {
			return "", fmt.Errorf("profile has no skin")
		}
		return payload.Textures.SKIN.URL, nil
	}
	return "", fmt.Errorf("profile has no textures property")
}

// normalizeSkinURL enforces https plus the texture host allowlist. Ely.by
// still emits http:// texture URLs, so the scheme is upgraded rather than
// fetched in the clear. Loopback hosts stay http so the httptest servers in
// tests exercise the same code path.
func normalizeSkinURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty texture url")
	}
	schemeless := strings.TrimPrefix(strings.TrimPrefix(trimmed, "https://"), "http://")
	authority, path, ok := strings.Cut(schemeless, "/")
	if !ok || authority == "" {
		return "", fmt.Errorf("invalid texture url")
	}
	host := strings.ToLower(authority)
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i] // allowlist matches host without port
	}
	if !skinTextureHosts[host] {
		return "", fmt.Errorf("texture host not allowlisted")
	}
	scheme := "https"
	if host == "127.0.0.1" || host == "localhost" {
		scheme = "http"
	}
	return scheme + "://" + strings.ToLower(authority) + "/" + path, nil
}

func (s *AccountService) downloadPNG(rawURL string) ([]byte, error) {
	normalized, err := normalizeSkinURL(rawURL)
	if err != nil {
		return nil, skinUnavailableError()
	}
	request, err := http.NewRequest(http.MethodGet, normalized, nil)
	if err != nil {
		return nil, skinUnavailableError()
	}
	response, err := s.skinHTTPClient().Do(request)
	if err != nil {
		return nil, skinUnavailableError()
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, skinUnavailableError()
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxSkinBytes))
	if err != nil || len(body) < pngMagicLen || string(body[:pngMagicLen]) != string(pngMagic) {
		return nil, skinUnavailableError()
	}
	return body, nil
}

// skinHTTPClient prefers the shared client and falls back to a bounded
// default so skin fetches never hang the dialog.
func (s *AccountService) skinHTTPClient() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func skinUnavailableError() *ServiceError {
	return NewUpstreamError("Unable to load the player skin right now.")
}
