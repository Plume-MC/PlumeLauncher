package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MicrosoftClientID  = "00000000402b5328"
	MicrosoftScope     = "service::user.auth.xboxlive.com::MBI_SSL"
	MinecraftUserAgent = "PlumeLauncher/1.0 (support@plumemc.dev; https://plumemc.dev)"
	DeviceTokenType    = "JWT"

	microsoftRedirectURI = "https://login.live.com/oauth20_desktop.srf"
)

var (
	deviceTokenURL  = "https://device.auth.xboxlive.com/device/authenticate"
	sisuAuthURL     = "https://sisu.xboxlive.com/authenticate"
	sisuAuthzURL    = "https://sisu.xboxlive.com/authorize"
	xstsAuthzURL    = "https://xsts.auth.xboxlive.com/xsts/authorize"
	mcLoginURL      = "https://api.minecraftservices.com/launcher/login"
	mcProfileURL    = "https://api.minecraftservices.com/minecraft/profile"
	mcEntitleURL    = "https://api.minecraftservices.com/entitlements/license"
	msOAuthTokenURL = "https://login.live.com/oauth20_token.srf"
)

type DeviceTokenKey struct {
	ID  uuid.UUID
	Key *ecdsa.PrivateKey
	X   string
	Y   string
}

type DeviceToken struct {
	IssueInstant  time.Time                  `json:"IssueInstant"`
	NotAfter      time.Time                  `json:"NotAfter"`
	Token         string                     `json:"Token"`
	DisplayClaims map[string]json.RawMessage `json:"DisplayClaims"`
}

type OAuthTokenResponse struct {
	ExpiresIn    int64  `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type SisuAuthorizeResponse struct {
	TitleToken DeviceToken `json:"TitleToken"`
	UserToken  DeviceToken `json:"UserToken"`
}

type MinecraftLoginResponse struct {
	AccessToken string `json:"access_token"`
	Username    string `json:"username"`
	ExpiresIn   int64  `json:"expires_in"`
}

type MinecraftProfileResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Skins []struct {
		ID      string `json:"id"`
		State   string `json:"state"`
		URL     string `json:"url"`
		Variant string `json:"variant"`
	} `json:"skins"`
	Capes []struct {
		ID    string `json:"id"`
		State string `json:"state"`
		URL   string `json:"url"`
		Alias string `json:"alias"`
	} `json:"capes"`
}

type MicrosoftLoginFlow struct {
	Verifier       string
	Challenge      string
	SessionID      string
	AuthRequestURI string
	// Key and DeviceToken must be reused for LoginFinish; SISU session is bound to them.
	Key         *DeviceTokenKey `json:"-"`
	DeviceToken string          `json:"-"`
}

// MicrosoftCredentials is transient: handed to the caller in memory and never
// persisted to JSON, logs, or bindings. Only the refresh token is stored,
// in the OS keyring under MicrosoftRefreshKey.
type MicrosoftCredentials struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	UUID         string
	Username     string
}

// GenerateKey creates a new ECDSA P-256 key pair for device authentication.
func GenerateKey() (*DeviceTokenKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ECDSA key: %w", err)
	}
	id := uuid.New()
	return &DeviceTokenKey{
		ID:  id,
		Key: key,
		X:   base64.RawURLEncoding.EncodeToString(pad32(key.PublicKey.X.Bytes())),
		Y:   base64.RawURLEncoding.EncodeToString(pad32(key.PublicKey.Y.Bytes())),
	}, nil
}

func pad32(b []byte) []byte {
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// makePKCE creates a PKCE code verifier and its S256 challenge.
func makePKCE() (verifier, challenge string) {
	verifier = generateRandomHex(64)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge
}

// OAuthTokenExchange exchanges an authorization code for OAuth tokens.
func OAuthTokenExchange(code, verifier string) (*OAuthTokenResponse, error) {
	form := map[string]string{
		"client_id":     MicrosoftClientID,
		"code":          code,
		"code_verifier": verifier,
		"grant_type":    "authorization_code",
		"redirect_uri":  microsoftRedirectURI,
		"scope":         MicrosoftScope,
	}
	return postOAuthForm(msOAuthTokenURL, form)
}

// OAuthRefresh refreshes the OAuth token using a refresh token.
func OAuthRefresh(refreshToken string) (*OAuthTokenResponse, error) {
	form := map[string]string{
		"client_id":     MicrosoftClientID,
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
		"redirect_uri":  microsoftRedirectURI,
		"scope":         MicrosoftScope,
	}
	return postOAuthForm(msOAuthTokenURL, form)
}

func generateRandomHex(length int) string {
	b := make([]byte, length/2)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func postOAuthForm(url string, form map[string]string) (*OAuthTokenResponse, error) {
	encoded := encodeForm(form)
	req, err := http.NewRequest("POST", url, strings.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth request: %d %s", resp.StatusCode, string(respBody))
	}
	var result OAuthTokenResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode oauth response: %w", err)
	}
	return &result, nil
}

func encodeForm(values map[string]string) string {
	var b strings.Builder
	first := true
	for k, v := range values {
		if !first {
			b.WriteByte('&')
		}
		first = false
		b.WriteString(url.QueryEscape(k))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(v))
	}
	return b.String()
}
