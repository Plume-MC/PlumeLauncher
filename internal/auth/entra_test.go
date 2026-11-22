package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func overrideEntraURLs(t *testing.T, authorize, token, device string) {
	t.Helper()
	prevA, prevT, prevD := entraAuthorizeURL, entraTokenURL, entraDeviceURL
	entraAuthorizeURL, entraTokenURL, entraDeviceURL = authorize, token, device
	t.Cleanup(func() { entraAuthorizeURL, entraTokenURL, entraDeviceURL = prevA, prevT, prevD })
}

func entraMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	overrideEntraURLs(t, srv.URL+"/authorize", srv.URL+"/token", srv.URL+"/devicecode")
	return srv
}

func TestEntraRefreshPostsRefreshGrant(t *testing.T) {
	var sawForm, sawCT string
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sawForm, sawCT = string(body), r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"ms-access","refresh_token":"ms-refresh-new","expires_in":3600}`))
	})

	token, err := EntraRefresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("EntraRefresh: %v", err)
	}
	if token.AccessToken != "ms-access" || token.RefreshToken != "ms-refresh-new" {
		t.Fatalf("unexpected token: %+v", token)
	}
	if time.Until(token.ExpiresAt) < time.Minute {
		t.Fatal("expiry must be in the future")
	}
	for _, want := range []string{"grant_type=refresh_token", "refresh_token=old-refresh", EntraClientID, "XboxLive.signin"} {
		if !strings.Contains(sawForm, url.QueryEscape(want)) && !strings.Contains(sawForm, want) {
			t.Fatalf("form missing %q: %s", want, sawForm)
		}
	}
	if sawCT != "application/x-www-form-urlencoded" {
		t.Fatalf("content type = %q", sawCT)
	}
}

func TestEntraRefreshKeepsTokenWhenRotationEmpty(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"ms-access","expires_in":3600}`))
	})

	token, err := EntraRefresh(context.Background(), "stable-refresh")
	if err != nil {
		t.Fatalf("EntraRefresh: %v", err)
	}
	if token.RefreshToken != "stable-refresh" {
		t.Fatalf("empty rotation must keep old token, got %q", token.RefreshToken)
	}
}

func TestEntraRefreshInvalidGrantIsTyped(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"token expired","access_token":"must-never-leak"}`))
	})

	_, err := EntraRefresh(context.Background(), "dead-refresh")
	if err == nil || !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("invalid_grant must match ErrInvalidGrant, got %v", err)
	}
	if strings.Contains(err.Error(), "must-never-leak") {
		t.Fatalf("entra error must not echo response body: %v", err)
	}
}

func TestEntraRefreshRejectsEmptyToken(t *testing.T) {
	if _, err := EntraRefresh(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty refresh token")
	}
}

func TestEntraDeviceCodeStartReturnsUserCode(t *testing.T) {
	var sawForm string
	var opened string
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sawForm = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"device_code":"dev-1","user_code":"ABCD-EFGH","verification_uri":"https://microsoft.com/link","expires_in":900,"interval":5}`))
	})

	var info EntraDeviceCode
	_, err := EntraDeviceCodeStart(context.Background(), func(u string) error { opened = u; return nil }, func(i EntraDeviceCode) { info = i })
	if err != nil {
		t.Fatalf("EntraDeviceCodeStart: %v", err)
	}
	if info.UserCode != "ABCD-EFGH" || info.VerificationURI != "https://microsoft.com/link" || info.Interval != 5 {
		t.Fatalf("unexpected device info: %+v", info)
	}
	if opened != "https://microsoft.com/link" {
		t.Fatalf("verification page must open, got %q", opened)
	}
	if !strings.Contains(sawForm, EntraClientID) || !strings.Contains(sawForm, "XboxLive.signin") {
		t.Fatalf("form missing client id or scope: %s", sawForm)
	}
}

func TestEntraDeviceCodeStartRejectsIncompleteResponse(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"device_code":"dev-1","user_code":"ABCD"}`))
	})
	if _, err := EntraDeviceCodeStart(context.Background(), nil, nil); err == nil {
		t.Fatal("expected error for missing verification_uri")
	}
}

func TestEntraDeviceCodePollPendingThenSuccess(t *testing.T) {
	calls := 0
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"authorization_pending"}`))
			return
		}
		w.Write([]byte(`{"access_token":"ms-access","refresh_token":"ms-refresh","expires_in":3600}`))
	})

	// Interval 0 collapses to the timer minimum; the poll must report pending first.
	_, interval, err := EntraDeviceCodePoll(context.Background(), "dev-1", 0)
	if err == nil || !errors.Is(err, ErrLoginPending) {
		t.Fatalf("first poll must be pending, got %v (interval %d)", err, interval)
	}

	token, _, err := EntraDeviceCodePoll(context.Background(), "dev-1", 0)
	if err != nil {
		t.Fatalf("second poll: %v", err)
	}
	if token.AccessToken != "ms-access" {
		t.Fatalf("unexpected token: %+v", token)
	}
}

func TestEntraDeviceCodePollSlowDownIncreasesInterval(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"slow_down"}`))
	})

	_, interval, err := EntraDeviceCodePoll(context.Background(), "dev-1", 5)
	if err == nil || !errors.Is(err, ErrLoginPending) {
		t.Fatalf("slow_down must surface as pending, got %v", err)
	}
	if interval != 10 {
		t.Fatalf("slow_down must raise interval by 5, got %d", interval)
	}
}

func TestEntraDeviceCodePollDeclinedCancels(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"authorization_declined"}`))
	})
	if _, _, err := EntraDeviceCodePoll(context.Background(), "dev-1", 0); !errors.Is(err, ErrLoginCancelled) {
		t.Fatalf("declined flow must be ErrLoginCancelled, got %v", err)
	}
}

func TestEntraPKCELoginFullFlow(t *testing.T) {
	var sawExchange string
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sawExchange = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"ms-access","refresh_token":"ms-refresh","expires_in":3600}`))
	})

	// The opener extracts the loopback redirect URI and state from the
	// authorize URL, then hits the loopback server the way Entra would.
	token, err := EntraPKCELogin(context.Background(), func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		redirect := u.Query().Get("redirect_uri")
		state := u.Query().Get("state")
		if redirect == "" || state == "" {
			t.Fatalf("authorize URL must carry redirect_uri and state: %s", authURL)
		}
		resp, err := http.Get(redirect + "/?code=auth-code&state=" + state)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	})
	if err != nil {
		t.Fatalf("EntraPKCELogin: %v", err)
	}
	if token.AccessToken != "ms-access" || token.RefreshToken != "ms-refresh" {
		t.Fatalf("unexpected token: %+v", token)
	}
	for _, want := range []string{"grant_type=authorization_code", "code=auth-code", "code_verifier="} {
		if !strings.Contains(sawExchange, want) {
			t.Fatalf("token exchange missing %q: %s", want, sawExchange)
		}
	}
}

func TestEntraPKCELoginRejectsBrowserFailure(t *testing.T) {
	entraMockServer(t, func(w http.ResponseWriter, r *http.Request) { t.Error("token endpoint must not be called") })
	if _, err := EntraPKCELogin(context.Background(), func(string) error { return errors.New("no browser") }); err == nil {
		t.Fatal("expected error when the browser cannot open")
	}
}

func overrideEntraChainURLs(t *testing.T, xbox, xsts, mcLogin, entitle string) {
	t.Helper()
	prevX, prevS, prevL, prevE := entraXboxAuthURL, entraXstsURL, entraMCLoginURL, entraMCEntitleURL
	entraXboxAuthURL, entraXstsURL, entraMCLoginURL, entraMCEntitleURL = xbox, xsts, mcLogin, entitle
	t.Cleanup(func() {
		entraXboxAuthURL, entraXstsURL, entraMCLoginURL, entraMCEntitleURL = prevX, prevS, prevL, prevE
	})
}

func chainTokenJSON(token string) string {
	return `{"Token":` + strconv.Quote(token) + `,"DisplayClaims":{"xui":[{"uhs":"userhash123"}]}}`
}

func TestEntraBuildMinecraftSessionFullChain(t *testing.T) {
	var sawRPS, sawSandbox, sawIdentity, sawBearer string
	mux := http.NewServeMux()
	mux.HandleFunc("/user/authenticate", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sawRPS = string(body)
		if r.Header.Get("x-xbl-contract-version") != "1" {
			t.Error("missing x-xbl-contract-version")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(chainTokenJSON("xbox-tok")))
	})
	mux.HandleFunc("/xsts/authorize", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sawSandbox = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(chainTokenJSON("xsts-tok")))
	})
	mux.HandleFunc("/authentication/login_with_xbox", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sawIdentity = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"mc-access","expires_in":3600}`))
	})
	mux.HandleFunc("/entitlements/mcstore", func(w http.ResponseWriter, r *http.Request) {
		sawBearer = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"name":"product_minecraft"},{"name":"game_minecraft"}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	overrideEntraChainURLs(t, srv.URL+"/user/authenticate", srv.URL+"/xsts/authorize",
		srv.URL+"/authentication/login_with_xbox", srv.URL+"/entitlements/mcstore")

	mcToken, err := EntraBuildMinecraftSession(context.Background(), "ms-access")
	if err != nil {
		t.Fatalf("EntraBuildMinecraftSession: %v", err)
	}
	if mcToken != "mc-access" {
		t.Fatalf("mc token = %q", mcToken)
	}
	if !strings.Contains(sawRPS, "d=ms-access") || !strings.Contains(sawRPS, `"AuthMethod":"RPS"`) {
		t.Fatalf("xbox request must carry RPS ticket: %s", sawRPS)
	}
	if !strings.Contains(sawSandbox, "rp://api.minecraftservices.com/") || !strings.Contains(sawSandbox, "xbox-tok") {
		t.Fatalf("xsts request malformed: %s", sawSandbox)
	}
	if !strings.Contains(sawIdentity, "XBL3.0 x=userhash123;xsts-tok") {
		t.Fatalf("identity token malformed: %s", sawIdentity)
	}
	if sawBearer != "Bearer mc-access" {
		t.Fatalf("entitlement bearer = %q", sawBearer)
	}
}

func TestEntraBuildMinecraftSessionRejectsNoLicense(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/authenticate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(chainTokenJSON("xbox-tok")))
	})
	mux.HandleFunc("/xsts/authorize", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(chainTokenJSON("xsts-tok")))
	})
	mux.HandleFunc("/authentication/login_with_xbox", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"mc-access","expires_in":3600}`))
	})
	mux.HandleFunc("/entitlements/mcstore", func(w http.ResponseWriter, r *http.Request) {
		// Valid response but no Minecraft items: the account owns nothing.
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"name":"some_other_game"}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	overrideEntraChainURLs(t, srv.URL+"/user/authenticate", srv.URL+"/xsts/authorize",
		srv.URL+"/authentication/login_with_xbox", srv.URL+"/entitlements/mcstore")

	if _, err := EntraBuildMinecraftSession(context.Background(), "ms-access"); !errors.Is(err, ErrNotOwned) {
		t.Fatalf("empty entitlements must be ErrNotOwned, got %v", err)
	}
}

func TestEntraBuildMinecraftSessionSurfacesXErr(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/authenticate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(chainTokenJSON("xbox-tok")))
	})
	mux.HandleFunc("/xsts/authorize", func(w http.ResponseWriter, r *http.Request) {
		// XErr 2148916233 = the account has no Xbox profile.
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"XErr":2148916233,"Message":"Xbox profile missing"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	overrideEntraChainURLs(t, srv.URL+"/user/authenticate", srv.URL+"/xsts/authorize", srv.URL+"/unused", srv.URL+"/unused")

	_, err := EntraBuildMinecraftSession(context.Background(), "ms-access")
	var svcErr *ServiceError
	if !errors.As(err, &svcErr) || !strings.Contains(svcErr.Error(), "XErr:2148916233") {
		t.Fatalf("XErr must surface in the error, got %v", err)
	}
}

func TestEntraXboxAuthenticateRejectsMissingUHS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Token":"tok","DisplayClaims":{"xui":[]}}`))
	}))
	defer srv.Close()
	overrideEntraChainURLs(t, srv.URL, srv.URL, srv.URL, srv.URL)

	if _, err := EntraXboxAuthenticate(context.Background(), "ms-access"); err == nil {
		t.Fatal("expected error when the user hash is absent")
	}
}
