package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func overrideOAuthURL(t *testing.T, url string) {
	t.Helper()
	prev := msOAuthTokenURL
	msOAuthTokenURL = url
	t.Cleanup(func() { msOAuthTokenURL = prev })
}

func TestGenerateKeyProducesP256Coordinates(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if key.Key == nil || key.X == "" || key.Y == "" {
		t.Fatal("key, X and Y must be populated")
	}
	if len(key.X) != 43 || len(key.Y) != 43 {
		t.Fatalf("X/Y must be 32-byte base64url (43 chars), got %d/%d", len(key.X), len(key.Y))
	}
}

func TestMakePKCEChallengeMatchesVerifier(t *testing.T) {
	verifier, challenge := makePKCE()
	if len(verifier) != 64 {
		t.Fatalf("verifier must be 64 hex chars, got %q", verifier)
	}
	h := sha256.Sum256([]byte(verifier))
	if challenge != base64.RawURLEncoding.EncodeToString(h[:]) {
		t.Fatal("challenge must be base64url(sha256(verifier))")
	}
}

func mockOAuthServer(t *testing.T, check func(r *http.Request, form string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if check != nil {
			check(r, string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "ms-access",
			"refresh_token": "ms-refresh",
			"expires_in":    3600,
		})
	}))
}

func TestOAuthTokenExchangePostsAuthorizationCodeGrant(t *testing.T) {
	var sawForm, sawCT string
	srv := mockOAuthServer(t, func(r *http.Request, form string) {
		sawForm, sawCT = form, r.Header.Get("Content-Type")
	})
	defer srv.Close()
	overrideOAuthURL(t, srv.URL)

	resp, err := OAuthTokenExchange("auth-code", "verifier-abc")
	if err != nil {
		t.Fatalf("OAuthTokenExchange: %v", err)
	}
	if resp.AccessToken != "ms-access" || resp.RefreshToken != "ms-refresh" || resp.ExpiresIn != 3600 {
		t.Fatalf("unexpected token response: %+v", resp)
	}
	for _, want := range []string{"grant_type=authorization_code", "code=auth-code", "code_verifier=verifier-abc"} {
		if !strings.Contains(sawForm, want) {
			t.Fatalf("form missing %q: %s", want, sawForm)
		}
	}
	if sawCT != "application/x-www-form-urlencoded" {
		t.Fatalf("content type = %q", sawCT)
	}
}

func TestOAuthRefreshPostsRefreshGrant(t *testing.T) {
	var sawForm string
	srv := mockOAuthServer(t, func(r *http.Request, form string) { sawForm = form })
	defer srv.Close()
	overrideOAuthURL(t, srv.URL)

	resp, err := OAuthRefresh("old-refresh")
	if err != nil {
		t.Fatalf("OAuthRefresh: %v", err)
	}
	if resp.RefreshToken == "" {
		t.Fatal("expected refreshed token")
	}
	for _, want := range []string{"grant_type=refresh_token", "refresh_token=old-refresh"} {
		if !strings.Contains(sawForm, want) {
			t.Fatalf("form missing %q: %s", want, sawForm)
		}
	}
}

func overrideChainURLs(t *testing.T, device, sisuAuth, sisuAuthz, xsts, mcLogin string) {
	t.Helper()
	prev := []struct {
		ptr *string
		val string
	}{
		{&deviceTokenURL, device}, {&sisuAuthURL, sisuAuth},
		{&sisuAuthzURL, sisuAuthz}, {&xstsAuthzURL, xsts}, {&mcLoginURL, mcLogin},
	}
	saved := make([]string, len(prev))
	for i, p := range prev {
		saved[i] = *p.ptr
		*p.ptr = p.val
	}
	t.Cleanup(func() {
		for i, p := range prev {
			*p.ptr = saved[i]
		}
	})
}

func tokenJSON(token string) string {
	xui, _ := json.Marshal([]map[string]string{{"uhs": "userhash123"}})
	return `{"Token":` + strconv.Quote(token) + `,"DisplayClaims":{"xui":` + string(xui) + `}}`
}

func TestXboxChainDeviceToMinecraftToken(t *testing.T) {
	var sawSig, sawContract bool
	mux := http.NewServeMux()
	mux.HandleFunc("/device/authenticate", func(w http.ResponseWriter, r *http.Request) {
		sawSig = r.Header.Get("Signature") != ""
		sawContract = r.Header.Get("x-xbl-contract-version") == "1"
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(tokenJSON("device-tok")))
	})
	mux.HandleFunc("/authenticate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-SessionId", "sess-1")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"MSAOAuthRedirect":"https://login.live.com/oauth?x=1"}`))
	})
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"TitleToken":` + tokenJSON("title-tok") + `,"UserToken":` + tokenJSON("user-tok") + `}`))
	})
	mux.HandleFunc("/xsts/authorize", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(tokenJSON("xsts-tok")))
	})
	mux.HandleFunc("/launcher/login", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "XBL3.0 x=userhash123;xsts-tok") {
			t.Errorf("xtoken malformed: %s", body)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing user agent")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"mc-access","username":"Steve"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	overrideChainURLs(t, srv.URL+"/device/authenticate", srv.URL+"/authenticate",
		srv.URL+"/authorize", srv.URL+"/xsts/authorize", srv.URL+"/launcher/login")

	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	dev, err := GetDeviceToken(key)
	if err != nil {
		t.Fatalf("GetDeviceToken: %v", err)
	}
	if dev.Token != "device-tok" {
		t.Fatalf("device token = %q", dev.Token)
	}
	if !sawSig || !sawContract {
		t.Fatal("signed POST must carry Signature + x-xbl-contract-version headers")
	}
	sess, redirect, err := SisuAuthenticate(dev.Token, "challenge", key)
	if err != nil {
		t.Fatalf("SisuAuthenticate: %v", err)
	}
	if sess != "sess-1" || redirect == "" {
		t.Fatalf("sisu = %q %q", sess, redirect)
	}
	authz, err := SisuAuthorize(sess, "ms-access", dev.Token, key)
	if err != nil {
		t.Fatalf("SisuAuthorize: %v", err)
	}
	xsts, err := XSTSAuthorize(authz.UserToken.Token, authz.TitleToken.Token, dev.Token, key)
	if err != nil {
		t.Fatalf("XSTSAuthorize: %v", err)
	}
	mc, err := GetMinecraftToken(xsts)
	if err != nil {
		t.Fatalf("GetMinecraftToken: %v", err)
	}
	if mc != "mc-access" {
		t.Fatalf("mc token = %q", mc)
	}
}

func TestSisuAuthenticateRejectsMissingSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"MSAOAuthRedirect":"https://x"}`))
	}))
	defer srv.Close()
	overrideChainURLs(t, srv.URL, srv.URL, srv.URL, srv.URL, srv.URL)

	key, _ := GenerateKey()
	if _, _, err := SisuAuthenticate("dev", "c", key); err == nil {
		t.Fatal("expected error when X-SessionId header is missing")
	}
}

func TestGetMinecraftTokenRejectsMissingUHS(t *testing.T) {
	if _, err := GetMinecraftToken(&DeviceToken{Token: "x"}); err == nil {
		t.Fatal("expected error when xui/uhs is absent")
	}
}

func TestOAuthTokenExchangeRejectsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()
	overrideOAuthURL(t, srv.URL)

	if _, err := OAuthTokenExchange("bad-code", "v"); err == nil {
		t.Fatal("expected error for 400 response")
	}
	if _, err := OAuthRefresh("bad-refresh"); err == nil {
		t.Fatal("expected error for 400 response")
	}
}
