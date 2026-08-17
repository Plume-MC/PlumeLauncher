package auth_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"plumelauncher/internal/auth"
)

func TestElyByClientSessionLifecycle(t *testing.T) {
	requests := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, r.URL.Path+":"+string(body))
		if r.URL.Path == "/auth/invalidate" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"accessToken":"secret","selectedProfile":{"id":"uuid","name":"Player"}}`))
	}))
	defer server.Close()
	client := auth.ElyByClient{BaseURL: server.URL + "/auth/"}
	session, err := client.Authenticate("player", "password")
	if err != nil || session.SelectedProfile.Name != "Player" {
		t.Fatalf("Authenticate = %#v, %v", session, err)
	}
	if _, err := client.Refresh(session.AccessToken); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if err := client.Invalidate(session.AccessToken); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
	if strings.Contains(strings.Join(requests, " "), "\"password\"") == false {
		t.Fatal("authenticate request omitted password")
	}
}
