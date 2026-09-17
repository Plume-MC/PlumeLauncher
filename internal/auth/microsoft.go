package auth

import (
	"bytes"
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

func appendUint32BE(buf []byte, v uint32) []byte {
	return append(buf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func appendUint64BE(buf []byte, v uint64) []byte {
	return append(buf, byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func appendInt32BE(buf []byte, v int32) []byte {
	return appendUint32BE(buf, uint32(v))
}

func signAndPost(url, urlPath string, body []byte, key *DeviceTokenKey, authorization string) ([]byte, http.Header, error) {
	now := time.Now()
	// Windows FILETIME (100-ns intervals since 1601-01-01)
	windowsTime := uint64(now.Unix()+11644473600) * 10000000

	var buf []byte
	buf = appendUint32BE(buf, 1)
	buf = append(buf, 0)
	buf = appendUint64BE(buf, windowsTime)
	buf = append(buf, 0)
	buf = append(buf, []byte("POST")...)
	buf = append(buf, 0)
	buf = append(buf, []byte(urlPath)...)
	buf = append(buf, 0)
	if authorization != "" {
		buf = append(buf, []byte(authorization)...)
	}
	buf = append(buf, 0)
	buf = append(buf, body...)
	buf = append(buf, 0)

	// Xbox expects raw ES256 (r||s), not ASN.1. Sign the SHA-256 of the policy buffer.
	hash := sha256.Sum256(buf)
	r, s, err := ecdsa.Sign(rand.Reader, key.Key, hash[:])
	if err != nil {
		return nil, nil, fmt.Errorf("sign request: %w", err)
	}

	sigPayload := make([]byte, 0, 4+8+32+32)
	sigPayload = appendInt32BE(sigPayload, 1)
	sigPayload = appendUint64BE(sigPayload, windowsTime)
	sigPayload = append(sigPayload, pad32(r.Bytes())...)
	sigPayload = append(sigPayload, pad32(s.Bytes())...)

	signature := base64.StdEncoding.EncodeToString(sigPayload)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Signature", signature)
	if url != sisuAuthzURL {
		req.Header.Set("x-xbl-contract-version", "1")
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("POST %s: %d %s", url, resp.StatusCode, string(respBody))
	}
	return respBody, resp.Header, nil
}

// GetDeviceToken requests an Xbox Live device token.
func GetDeviceToken(key *DeviceTokenKey) (*DeviceToken, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"Properties": map[string]interface{}{
			"AuthMethod": "ProofOfPossession",
			"Id":         "{" + strings.ToUpper(key.ID.String()) + "}",
			"DeviceType": "Win32",
			"Version":    "10.16.0",
			"ProofKey": map[string]string{
				"kty": "EC",
				"x":   key.X,
				"y":   key.Y,
				"crv": "P-256",
				"alg": "ES256",
				"use": "sig",
			},
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    DeviceTokenType,
	})

	respBody, _, err := signAndPost(deviceTokenURL, "/device/authenticate", body, key, "")
	if err != nil {
		return nil, fmt.Errorf("device token: %w", err)
	}
	var token DeviceToken
	if err := json.Unmarshal(respBody, &token); err != nil {
		return nil, fmt.Errorf("decode device token: %w", err)
	}
	return &token, nil
}

// SisuAuthenticate initiates the SISU authentication flow.
func SisuAuthenticate(deviceToken string, challenge string, key *DeviceTokenKey) (sessionID string, redirectURI string, err error) {
	state := generateRandomHex(32)
	body, _ := json.Marshal(map[string]interface{}{
		"AppId":       MicrosoftClientID,
		"DeviceToken": deviceToken,
		"Offers":      []string{MicrosoftScope},
		"Query": map[string]string{
			"code_challenge":        challenge,
			"code_challenge_method": "S256",
			"state":                 state,
			"prompt":                "select_account",
		},
		"RedirectUri": microsoftRedirectURI,
		"Sandbox":     "RETAIL",
		"TokenType":   "code",
		"TitleId":     "1794566092",
	})

	respBody, headers, err := signAndPost(sisuAuthURL, "/authenticate", body, key, "")
	if err != nil {
		return "", "", fmt.Errorf("sisu authenticate: %w", err)
	}

	sessionID = headers.Get("X-SessionId")
	if sessionID == "" {
		return "", "", fmt.Errorf("no X-SessionId header")
	}

	var result struct {
		MSAOAuthRedirect string `json:"MSAOAuthRedirect"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("decode sisu response: %w", err)
	}
	if result.MSAOAuthRedirect == "" {
		return "", "", fmt.Errorf("sisu response did not include MSAOAuthRedirect")
	}
	return sessionID, result.MSAOAuthRedirect, nil
}

// SisuAuthorize performs SISU authorization with the OAuth token.
func SisuAuthorize(sessionID, accessToken, deviceToken string, key *DeviceTokenKey) (*SisuAuthorizeResponse, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"AccessToken": "t=" + accessToken,
		"AppId":       MicrosoftClientID,
		"DeviceToken": deviceToken,
		"ProofKey": map[string]string{
			"kty": "EC",
			"x":   key.X,
			"y":   key.Y,
			"crv": "P-256",
			"alg": "ES256",
			"use": "sig",
		},
		"Sandbox":           "RETAIL",
		"SessionId":         sessionID,
		"SiteName":          "user.auth.xboxlive.com",
		"RelyingParty":      "http://xboxlive.com",
		"UseModernGamertag": true,
	})

	respBody, _, err := signAndPost(sisuAuthzURL, "/authorize", body, key, "")
	if err != nil {
		return nil, fmt.Errorf("sisu authorize: %w", err)
	}
	var result SisuAuthorizeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode sisu authorize: %w", err)
	}
	return &result, nil
}

// XSTSAuthorize authorizes with XSTS for Minecraft services.
func XSTSAuthorize(userToken, titleToken, deviceToken string, key *DeviceTokenKey) (*DeviceToken, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType":    DeviceTokenType,
		"Properties": map[string]interface{}{
			"SandboxId":   "RETAIL",
			"UserTokens":  []string{userToken},
			"DeviceToken": deviceToken,
			"TitleToken":  titleToken,
		},
	})

	respBody, _, err := signAndPost(xstsAuthzURL, "/xsts/authorize", body, key, "")
	if err != nil {
		return nil, fmt.Errorf("xsts authorize: %w", err)
	}
	var token DeviceToken
	if err := json.Unmarshal(respBody, &token); err != nil {
		return nil, fmt.Errorf("decode xsts token: %w", err)
	}
	return &token, nil
}

// GetMinecraftToken exchanges an XSTS token for a Minecraft access token.
func GetMinecraftToken(xstsToken *DeviceToken) (string, error) {
	uhs := getXUHS(xstsToken)
	if uhs == "" {
		return "", fmt.Errorf("no user hash in xsts token")
	}
	token := xstsToken.Token

	body, _ := json.Marshal(map[string]string{
		"platform": "PC_LAUNCHER",
		"xtoken":   "XBL3.0 x=" + uhs + ";" + token,
	})

	req, err := http.NewRequest("POST", mcLoginURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", MinecraftUserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("minecraft login: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("minecraft login: %d %s", resp.StatusCode, string(respBody))
	}
	var result MinecraftLoginResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decode minecraft token: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("minecraft login returned empty access token: %s", string(respBody))
	}
	return result.AccessToken, nil
}

func getXUHS(token *DeviceToken) string {
	var xui []struct {
		UHS string `json:"uhs"`
	}
	if err := json.Unmarshal(token.DisplayClaims["xui"], &xui); err != nil || len(xui) == 0 {
		return ""
	}
	return xui[0].UHS
}

// CheckMinecraftEntitlements verifies the account owns Minecraft.
// A non-2xx response means no license: the caller must reject the account,
// not retry silently.
func CheckMinecraftEntitlements(accessToken string) error {
	req, err := http.NewRequest("GET", mcEntitleURL+"?requestId="+uuid.New().String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", MinecraftUserAgent)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("entitlements: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("entitlements: %d %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// GetMinecraftProfile fetches the player profile (read-only; no skin upload).
func GetMinecraftProfile(accessToken string) (*MinecraftProfileResponse, error) {
	req, err := http.NewRequest("GET", mcProfileURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", MinecraftUserAgent)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch profile: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch profile: %d %s", resp.StatusCode, string(respBody))
	}
	var profile MinecraftProfileResponse
	if err := json.Unmarshal(respBody, &profile); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	return &profile, nil
}

// LoginBegin starts the Microsoft login flow. The returned flow state
// (key + device token) must be reused for LoginFinish.
func LoginBegin() (*MicrosoftLoginFlow, error) {
	key, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	verifier, challenge := makePKCE()

	deviceToken, err := GetDeviceToken(key)
	if err != nil {
		return nil, err
	}

	sessionID, redirectURI, err := SisuAuthenticate(deviceToken.Token, challenge, key)
	if err != nil {
		return nil, err
	}

	return &MicrosoftLoginFlow{
		Verifier:       verifier,
		Challenge:      challenge,
		SessionID:      sessionID,
		AuthRequestURI: redirectURI,
		Key:            key,
		DeviceToken:    deviceToken.Token,
	}, nil
}

// LoginFinish completes the Microsoft login after receiving the auth code.
// The refresh token is stored in the OS keyring; the returned credentials
// are transient and must never be written to JSON, logs, or bindings.
func LoginFinish(code string, flow *MicrosoftLoginFlow, store Keyring) (*MicrosoftCredentials, error) {
	if flow == nil || flow.Key == nil || flow.DeviceToken == "" {
		return nil, fmt.Errorf("microsoft login flow is incomplete")
	}

	oauthToken, err := OAuthTokenExchange(code, flow.Verifier)
	if err != nil {
		return nil, err
	}

	sisuAuth, err := SisuAuthorize(flow.SessionID, oauthToken.AccessToken, flow.DeviceToken, flow.Key)
	if err != nil {
		return nil, err
	}

	xstsToken, err := XSTSAuthorize(sisuAuth.UserToken.Token, sisuAuth.TitleToken.Token, flow.DeviceToken, flow.Key)
	if err != nil {
		return nil, err
	}

	mcToken, err := GetMinecraftToken(xstsToken)
	if err != nil {
		return nil, err
	}

	if err := CheckMinecraftEntitlements(mcToken); err != nil {
		return nil, err
	}

	profile, err := GetMinecraftProfile(mcToken)
	if err != nil {
		return nil, err
	}

	creds := &MicrosoftCredentials{
		AccessToken:  mcToken,
		RefreshToken: oauthToken.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(oauthToken.ExpiresIn) * time.Second),
		UUID:         profile.ID,
		Username:     profile.Name,
	}

	if store != nil {
		if err := store.Set(MicrosoftRefreshKey(profile.ID), oauthToken.RefreshToken); err != nil {
			return nil, fmt.Errorf("store microsoft session: %w", err)
		}
	}

	return creds, nil
}

// RefreshMicrosoft refreshes Microsoft credentials using a refresh token.
func RefreshMicrosoft(refreshToken string, store Keyring) (*MicrosoftCredentials, error) {
	oauthToken, err := OAuthRefresh(refreshToken)
	if err != nil {
		return nil, err
	}

	key, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	deviceToken, err := GetDeviceToken(key)
	if err != nil {
		return nil, err
	}

	sisuAuth, err := SisuAuthorize("", oauthToken.AccessToken, deviceToken.Token, key)
	if err != nil {
		return nil, err
	}

	xstsToken, err := XSTSAuthorize(sisuAuth.UserToken.Token, sisuAuth.TitleToken.Token, deviceToken.Token, key)
	if err != nil {
		return nil, err
	}

	mcToken, err := GetMinecraftToken(xstsToken)
	if err != nil {
		return nil, err
	}

	profile, err := GetMinecraftProfile(mcToken)
	if err != nil {
		return nil, err
	}

	if store != nil {
		if err := store.Set(MicrosoftRefreshKey(profile.ID), oauthToken.RefreshToken); err != nil {
			return nil, fmt.Errorf("store microsoft session: %w", err)
		}
	}

	return &MicrosoftCredentials{
		AccessToken:  mcToken,
		RefreshToken: oauthToken.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(oauthToken.ExpiresIn) * time.Second),
		UUID:         profile.ID,
		Username:     profile.Name,
	}, nil
}

// EnsureValidMicrosoftToken returns refreshed credentials when the token
// expires within 5 minutes, or empty access/refresh on a still-valid token.
func EnsureValidMicrosoftToken(refreshToken string, expiresAt time.Time, store Keyring) (string, string, time.Time, error) {
	if time.Until(expiresAt) > 5*time.Minute {
		return "", "", expiresAt, nil
	}
	creds, err := RefreshMicrosoft(refreshToken, store)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return creds.AccessToken, creds.RefreshToken, creds.ExpiresAt, nil
}
