package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Entra (Azure AD consumers tenant) OAuth endpoints. The consumers tenant
// owns the XboxLive.signin scope and serves personal Microsoft accounts. It
// routes authorize through login.live.com, which accepts a registered
// http://localhost redirect URI with any port.
var (
	entraAuthorizeURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize"
	entraTokenURL     = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
	entraDeviceURL    = "https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode"
)

const (
	// EntraClientID is the launcher's own app registration. The redirect URI
	// registered for it is "http://localhost" (any port).
	EntraClientID  = "3270dced-41e9-4727-8d86-0058e36a7e4f"
	entraScope     = "XboxLive.signin offline_access"
	entraMaxRobody = 1 << 20
)

// EntraToken is the OAuth token set returned by the Entra token endpoint.
type EntraToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// Login flow sentinels shared by the Entra flows.
var (
	// ErrLoginInProgress marks that another login flow is already running.
	ErrLoginInProgress = errors.New("an authentication flow is already running")
	// ErrLoginCancelled marks a cancelled or declined login flow.
	ErrLoginCancelled = errors.New("authentication was cancelled")
	// ErrLoginPending marks a device-code flow the user has not approved yet.
	ErrLoginPending = errors.New("authorization pending")
)

// EntraError is a non-2xx response from an Entra OAuth endpoint. The raw
// description is truncated and never carries request material.
type EntraError struct {
	Code        string
	Description string
}

func (e *EntraError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("entra oauth: %s", e.Code)
	}
	return "entra oauth: request rejected"
}

// Is reports invalid_grant as ErrInvalidGrant so callers can distinguish a
// dead session from a transient failure.
func (e *EntraError) Is(target error) bool {
	return target == ErrInvalidGrant && e.Code == "invalid_grant"
}

// EntraDeviceCode is the user-facing part of a device-code session.
type EntraDeviceCode struct {
	UserCode        string
	VerificationURI string
	Message         string
	ExpiresIn       int
	Interval        int
}

// EntraPKCELogin runs the authorization-code flow with PKCE against Entra.
// It starts a loopback listener on a dynamic port, opens the system browser
// via openURL, and returns the tokens after Entra redirects back.
func EntraPKCELogin(ctx context.Context, openURL func(string) error) (EntraToken, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return EntraToken{}, fmt.Errorf("start entra oauth loopback listener: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := "http://localhost:" + strconv.Itoa(port)

	verifier := generateRandomHex(64)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	state := generateRandomHex(32)

	callbackResult := make(chan callbackData, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid OAuth state. Return to the launcher and retry.", http.StatusBadRequest)
			return
		}
		if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
			desc := r.URL.Query().Get("error_description")
			http.Error(w, "Microsoft sign-in was not completed. You can close this tab.", http.StatusBadRequest)
			select {
			case callbackResult <- callbackData{err: &EntraError{Code: oauthErr, Description: safeEntraDescription(desc, oauthErr)}}:
			default:
			}
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code. You can close this tab.", http.StatusBadRequest)
			select {
			case callbackResult <- callbackData{err: fmt.Errorf("entra callback did not contain an authorization code")}:
			default:
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html><html><head><meta charset=utf-8><title>Sign-in complete</title></head><body><h2>Sign-in complete</h2><p>You can close this tab and return to your launcher.</p></body></html>"))
		select {
		case callbackResult <- callbackData{code: code}:
		default:
		}
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       5 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		_ = server.Serve(listener)
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		<-serverDone
	}()

	params := url.Values{
		"client_id":             {EntraClientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"response_mode":         {"query"},
		"scope":                 {entraScope},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	if err := openURL(entraAuthorizeURL + "?" + params.Encode()); err != nil {
		return EntraToken{}, fmt.Errorf("open system browser: %w", err)
	}

	var callback callbackData
	select {
	case <-ctx.Done():
		return EntraToken{}, ErrLoginCancelled
	case callback = <-callbackResult:
	}
	if callback.err != nil {
		return EntraToken{}, callback.err
	}

	form := url.Values{
		"client_id":     {EntraClientID},
		"grant_type":    {"authorization_code"},
		"code":          {callback.code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
		"scope":         {entraScope},
	}
	token, err := entraPostForm(ctx, entraTokenURL, form)
	if err != nil {
		return EntraToken{}, err
	}
	return normalizeEntraToken(token, ""), nil
}

type callbackData struct {
	code string
	err  error
}

// EntraDeviceCodeStart requests a device code. onCode receives the user code
// to confirm on another device; the verification page is opened when openURL
// is non-nil.
func EntraDeviceCodeStart(ctx context.Context, openURL func(string) error, onCode func(EntraDeviceCode)) (EntraDeviceCode, error) {
	form := url.Values{
		"client_id": {EntraClientID},
		"scope":     {entraScope},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, entraDeviceURL, strings.NewReader(form.Encode()))
	if err != nil {
		return EntraDeviceCode{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return EntraDeviceCode{}, fmt.Errorf("request entra device code: %w", err)
	}
	defer resp.Body.Close()

	var device struct {
		DeviceCode       string `json:"device_code"`
		UserCode         string `json:"user_code"`
		VerificationURI  string `json:"verification_uri"`
		Message          string `json:"message"`
		ExpiresIn        int    `json:"expires_in"`
		Interval         int    `json:"interval"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, entraMaxRobody))
	if err != nil {
		return EntraDeviceCode{}, fmt.Errorf("read entra device response: %w", err)
	}
	if err := json.Unmarshal(body, &device); err != nil {
		return EntraDeviceCode{}, fmt.Errorf("decode entra device response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return EntraDeviceCode{}, &EntraError{Code: device.Error, Description: safeEntraDescription(device.ErrorDescription, device.Error)}
	}
	if device.DeviceCode == "" || device.UserCode == "" || device.VerificationURI == "" {
		return EntraDeviceCode{}, fmt.Errorf("entra returned an incomplete device-code response")
	}

	info := EntraDeviceCode{
		UserCode:        device.UserCode,
		VerificationURI: device.VerificationURI,
		Message:         device.Message,
		ExpiresIn:       device.ExpiresIn,
		Interval:        device.Interval,
	}
	if onCode != nil {
		onCode(info)
	}
	if openURL != nil {
		// Opening the URL is a convenience; the user code remains visible in the UI.
		_ = openURL(device.VerificationURI)
	}
	return info, nil
}

// EntraDeviceCodePoll exchanges one pending device code for tokens. It
// returns ErrLoginPending while the user has not approved yet; callers poll
// at the returned interval until approval, denial, expiry, or cancel.
func EntraDeviceCodePoll(ctx context.Context, deviceCode string, interval int) (EntraToken, int, error) {
	timer := time.NewTimer(time.Duration(interval) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return EntraToken{}, interval, ErrLoginCancelled
	case <-timer.C:
	}

	form := url.Values{
		"client_id":   {EntraClientID},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {deviceCode},
	}
	token, err := entraPostForm(ctx, entraTokenURL, form)
	if err == nil {
		return normalizeEntraToken(token, ""), interval, nil
	}

	var entraErr *EntraError
	if !errors.As(err, &entraErr) {
		return EntraToken{}, interval, err
	}
	switch entraErr.Code {
	case "authorization_pending":
		return EntraToken{}, interval, ErrLoginPending
	case "slow_down":
		return EntraToken{}, interval + 5, ErrLoginPending
	case "authorization_declined", "access_denied":
		return EntraToken{}, interval, ErrLoginCancelled
	case "expired_token":
		return EntraToken{}, interval, fmt.Errorf("entra device code expired; start sign-in again")
	default:
		return EntraToken{}, interval, entraErr
	}
}

// EntraRefresh exchanges a refresh token for a fresh token set. Entra may
// omit refresh_token when it does not rotate; the working token is kept.
func EntraRefresh(ctx context.Context, refreshToken string) (EntraToken, error) {
	if refreshToken == "" {
		return EntraToken{}, fmt.Errorf("no entra refresh token is available")
	}
	form := url.Values{
		"client_id":     {EntraClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {entraScope},
	}
	token, err := entraPostForm(ctx, entraTokenURL, form)
	if err != nil {
		return EntraToken{}, err
	}
	return normalizeEntraToken(token, refreshToken), nil
}

func normalizeEntraToken(raw entraTokenResponse, oldRefresh string) EntraToken {
	refresh := raw.RefreshToken
	if refresh == "" {
		refresh = oldRefresh
	}
	return EntraToken{
		AccessToken:  raw.AccessToken,
		RefreshToken: refresh,
		ExpiresAt:    time.Now().Add(time.Duration(raw.ExpiresIn) * time.Second),
	}
}

type entraTokenResponse struct {
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`
}

// EntraToken carries expiresIn only between the raw response and
// normalizeEntraToken; it is unexported to keep it out of callers' hands.
type expiresIn = int64

func (t EntraToken) expiresIn() int64 { return 0 }

// entraPostForm posts a form to an Entra endpoint and decodes the token
// response. Errors map to EntraError so invalid_grant stays typed.
func entraPostForm(ctx context.Context, endpoint string, form url.Values) (entraTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return entraTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return entraTokenResponse{}, fmt.Errorf("entra oauth network request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, entraMaxRobody))
	if err != nil {
		return entraTokenResponse{}, fmt.Errorf("read entra oauth response: %w", err)
	}

	var payload entraTokenResponse
	_ = json.Unmarshal(body, &payload)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return entraTokenResponse{}, &EntraError{Code: payload.Error, Description: safeEntraDescription(payload.Description, payload.Error)}
	}
	if payload.AccessToken == "" {
		return entraTokenResponse{}, fmt.Errorf("entra oauth returned an empty access token")
	}
	return payload, nil
}

func safeEntraDescription(description, code string) string {
	// Microsoft descriptions are useful diagnostics, but never forward arbitrary body text.
	if description == "" {
		return "The identity provider rejected the request"
	}
	if len(description) > 240 {
		description = description[:240]
	}
	return strings.ReplaceAll(description, "\n", " ")
}
