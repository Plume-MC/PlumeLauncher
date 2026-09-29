package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Device-code OAuth endpoints (account.ely.by).
// Contract source: Ely.by OAuth docs (docs.ely.by/en/oauth.html),
// elyby/accounts DeviceCodeCest.php (referenced by Ely.by as documentation),
// and elyby/league-oauth2-ely Provider.php.
const (
	ElyOAuthBaseURL      = "https://account.ely.by"
	ElyOAuthDevicePath   = "/api/oauth2/v1/devicecode"
	ElyOAuthTokenPath    = "/api/oauth2/v1/token"
	ElyOAuthAccountPath  = "/api/account/v1/info"
	ElyOAuthDeviceGrant  = "urn:ietf:params:oauth:grant-type:device_code"
	ElyOAuthRefreshGrant = "refresh_token"
	// ElyOAuthScope mirrors third-party launchers (account + game session
	// + offline refresh). offline_access is what yields a refresh_token.
	ElyOAuthScope = "account_info minecraft_server_session offline_access"
	// ElyOAuthDefaultClientID is a placeholder. The maintainer must register
	// the application at https://account.ely.by/dev/ and configure the real
	// client ID (see AccountService.ElyOAuthClientID). Unregistered IDs fail
	// with invalid_client.
	ElyOAuthDefaultClientID = "plume-launcher"
)

// Device-code flow errors returned by the token endpoint.
var (
	ErrOAuthPending = errors.New("ely.by authorization pending")
	ErrOAuthExpired = errors.New("ely.by device code expired")
	ErrOAuthDenied  = errors.New("ely.by authorization denied")
)

// ElyOAuthError is a typed device-code failure. Code is the raw
// error identifier from Ely.by (or a transport label); it never
// carries tokens, codes, or secrets.
type ElyOAuthError struct {
	Code    string
	Message string
}

func (e *ElyOAuthError) Error() string { return "ely.by oauth: " + e.Message }

// ElyDeviceCodeStart is the user-facing part of a device-code session.
// DeviceCode stays backend-side where possible; the frontend needs it
// only to drive Finish/Poll and Cancel calls.
type ElyDeviceCodeStart struct {
	DeviceCode      string `json:"deviceCode"`
	UserCode        string `json:"userCode"`
	VerificationURI string `json:"verificationUri"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
}

// ElyOAuthAccount is the minimal profile returned for an OAuth token.
type ElyOAuthAccount struct {
	ID       int64  `json:"id"`
	UUID     string `json:"uuid"`
	Username string `json:"username"`
}

// ElyOAuthClient performs the Ely.by device-code flow. ClientID must be
// registered at account.ely.by/dev. BaseURL is overridable for tests.
type ElyOAuthClient struct {
	HTTPClient *http.Client
	BaseURL    string
	ClientID   string
}

func (c ElyOAuthClient) baseURL() string {
	if c.BaseURL != "" {
		return strings.TrimSuffix(c.BaseURL, "/")
	}
	return ElyOAuthBaseURL
}

func (c ElyOAuthClient) clientID() string {
	if c.ClientID != "" {
		return c.ClientID
	}
	return ElyOAuthDefaultClientID
}

func (c ElyOAuthClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// StartDeviceCode opens a device-code session and returns the user code
// the player must confirm on the Ely.by website.
func (c ElyOAuthClient) StartDeviceCode(ctx context.Context) (ElyDeviceCodeStart, error) {
	form := url.Values{}
	form.Set("client_id", c.clientID())
	form.Set("scope", ElyOAuthScope)
	var out struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Error           string `json:"error"`
	}
	if err := c.postForm(ctx, ElyOAuthDevicePath, form, &out); err != nil {
		return ElyDeviceCodeStart{}, err
	}
	switch out.Error {
	case "":
		// continue below
	case "invalid_client":
		return ElyDeviceCodeStart{}, &ElyOAuthError{Code: "invalid_client", Message: "unknown Ely.by application registration"}
	case "invalid_scope":
		return ElyDeviceCodeStart{}, &ElyOAuthError{Code: "invalid_scope", Message: "unsupported Ely.by scope requested"}
	default:
		return ElyDeviceCodeStart{}, &ElyOAuthError{Code: out.Error, Message: "unable to start Ely.by sign-in"}
	}
	if out.DeviceCode == "" || out.UserCode == "" || out.VerificationURI == "" {
		return ElyDeviceCodeStart{}, &ElyOAuthError{Code: "invalid_response", Message: "incomplete Ely.by device response"}
	}
	interval := out.Interval
	if interval <= 0 {
		interval = 5
	}
	return ElyDeviceCodeStart{
		DeviceCode:      out.DeviceCode,
		UserCode:        out.UserCode,
		VerificationURI: out.VerificationURI,
		ExpiresIn:       out.ExpiresIn,
		Interval:        interval,
	}, nil
}

// PollDeviceToken exchanges an approved device code for an access token.
// An unapproved code returns ErrOAuthPending; callers poll at the
// server-provided interval until approval, denial, expiry, or cancel.
func (c ElyOAuthClient) PollDeviceToken(ctx context.Context, deviceCode string) (string, error) {
	if deviceCode == "" {
		return "", &ElyOAuthError{Code: "invalid_request", Message: "missing device code"}
	}
	form := url.Values{}
	form.Set("grant_type", ElyOAuthDeviceGrant)
	form.Set("client_id", c.clientID())
	form.Set("device_code", deviceCode)
	var out struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
	}
	status, err := c.postFormStatus(ctx, ElyOAuthTokenPath, form, &out)
	if err != nil {
		return "", err
	}
	if status == http.StatusOK && out.AccessToken != "" {
		return out.AccessToken, nil
	}
	switch out.Error {
	case "authorization_pending":
		return "", ErrOAuthPending
	case "expired_token":
		return "", ErrOAuthExpired
	case "access_denied":
		return "", ErrOAuthDenied
	case "invalid_client":
		return "", &ElyOAuthError{Code: "invalid_client", Message: "unknown Ely.by application registration"}
	case "":
		return "", &ElyOAuthError{Code: "invalid_response", Message: "incomplete Ely.by token response"}
	default:
		return "", &ElyOAuthError{Code: out.Error, Message: "Ely.by sign-in failed"}
	}
}

// FetchAccountInfo resolves the profile behind an OAuth access token.
func (c ElyOAuthClient) FetchAccountInfo(ctx context.Context, accessToken string) (ElyOAuthAccount, error) {
	if accessToken == "" {
		return ElyOAuthAccount{}, &ElyOAuthError{Code: "invalid_request", Message: "missing access token"}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+ElyOAuthAccountPath, nil)
	if err != nil {
		return ElyOAuthAccount{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := c.httpClient().Do(request)
	if err != nil {
		return ElyOAuthAccount{}, fmt.Errorf("ely.by account request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ElyOAuthAccount{}, &ElyOAuthError{Code: "upstream", Message: "unable to read Ely.by profile"}
	}
	var info ElyOAuthAccount
	if err := json.NewDecoder(response.Body).Decode(&info); err != nil {
		return ElyOAuthAccount{}, fmt.Errorf("decode ely.by profile: %w", err)
	}
	if info.UUID == "" || info.Username == "" {
		return ElyOAuthAccount{}, &ElyOAuthError{Code: "invalid_response", Message: "incomplete Ely.by profile"}
	}
	return info, nil
}

// RefreshOAuthToken rotates an OAuth access token. The refresh token is
// only issued when the offline_access scope was granted; an empty
// returned refresh means the previous one stays valid.
func (c ElyOAuthClient) RefreshOAuthToken(ctx context.Context, refreshToken string) (access, refresh string, err error) {
	if refreshToken == "" {
		return "", "", &ElyOAuthError{Code: "invalid_request", Message: "missing refresh token"}
	}
	form := url.Values{}
	form.Set("grant_type", ElyOAuthRefreshGrant)
	form.Set("client_id", c.clientID())
	form.Set("refresh_token", refreshToken)
	form.Set("scope", ElyOAuthScope)
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
	}
	if _, err := c.postFormStatus(ctx, ElyOAuthTokenPath, form, &out); err != nil {
		return "", "", err
	}
	if out.AccessToken == "" {
		if out.Error != "" {
			return "", "", &ElyOAuthError{Code: out.Error, Message: "Ely.by session refresh failed"}
		}
		return "", "", &ElyOAuthError{Code: "invalid_response", Message: "incomplete Ely.by refresh response"}
	}
	return out.AccessToken, out.RefreshToken, nil
}

func (c ElyOAuthClient) postForm(ctx context.Context, path string, form url.Values, out any) error {
	_, err := c.postFormStatus(ctx, path, form, out)
	return err
}

func (c ElyOAuthClient) postFormStatus(ctx context.Context, path string, form url.Values, out any) (int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+path, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.httpClient().Do(request)
	if err != nil {
		return 0, fmt.Errorf("ely.by oauth request: %w", err)
	}
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return response.StatusCode, fmt.Errorf("decode ely.by oauth response: %w", err)
	}
	return response.StatusCode, nil
}
