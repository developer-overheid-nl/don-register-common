package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientCredentialsHTTPClientAuthenticatesAndReusesToken(t *testing.T) {
	tokenRequests := 0
	resourceRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenRequests++
			assertTokenRequest(t, r, "repositories:read scans:write")
			w.Header().Set("Content-Type", "application/json")
			if _, err := fmt.Fprint(w, `{"access_token":"token-1","token_type":"Bearer","expires_in":3600}`); err != nil {
				t.Error(err)
			}
		case "/resource":
			resourceRequests++
			if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
				t.Errorf("Authorization = %q", got)
			}
			if _, err := fmt.Fprint(w, "ok"); err != nil {
				t.Error(err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	base := server.Client()
	base.Timeout = 7 * time.Second
	client, err := NewClientCredentialsHTTPClient(context.Background(), ClientCredentialsConfig{
		TokenURL: server.URL + "/token", ClientID: "runner", ClientSecret: "secret",
		Scopes: []string{"repositories:read", "scans:write"},
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	if client.Timeout != base.Timeout {
		t.Fatalf("timeout = %s, want %s", client.Timeout, base.Timeout)
	}
	for range 2 {
		response, err := client.Get(server.URL + "/resource")
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(response.Body)
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if readErr != nil || string(data) != "ok" {
			t.Fatalf("resource response = %q, %v", data, readErr)
		}
	}
	if tokenRequests != 1 || resourceRequests != 2 {
		t.Fatalf("token requests=%d, resource requests=%d", tokenRequests, resourceRequests)
	}
}

func TestClientCredentialsHTTPClientRenewsExpiringToken(t *testing.T) {
	tokenRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			tokenRequests++
			w.Header().Set("Content-Type", "application/json")
			if _, err := fmt.Fprintf(w, `{"access_token":"token-%d","token_type":"Bearer","expires_in":1}`, tokenRequests); err != nil {
				t.Error(err)
			}
			return
		}
		if got, want := r.Header.Get("Authorization"), fmt.Sprintf("Bearer token-%d", tokenRequests); got != want {
			t.Errorf("Authorization = %q, want %q", got, want)
		}
		if _, err := fmt.Fprint(w, "ok"); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client, err := NewClientCredentialsHTTPClient(context.Background(), ClientCredentialsConfig{
		TokenURL: server.URL + "/token", ClientID: "runner", ClientSecret: "secret",
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		response, err := client.Get(server.URL + "/resource")
		if err != nil {
			t.Fatal(err)
		}
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if tokenRequests != 2 {
		t.Fatalf("token was not renewed: requests=%d", tokenRequests)
	}
}

func TestClientCredentialsHTTPClientRejectsIncompleteConfiguration(t *testing.T) {
	valid := ClientCredentialsConfig{TokenURL: "https://auth.example.test/token", ClientID: "runner", ClientSecret: "secret"}
	tests := []ClientCredentialsConfig{
		{},
		{TokenURL: valid.TokenURL, ClientID: valid.ClientID},
		{TokenURL: "file:///tmp/token", ClientID: valid.ClientID, ClientSecret: valid.ClientSecret},
		{TokenURL: "https://user:password@auth.example.test/token", ClientID: valid.ClientID, ClientSecret: valid.ClientSecret},
	}
	for _, config := range tests {
		if _, err := NewClientCredentialsHTTPClient(context.Background(), config, nil); err == nil {
			t.Fatalf("invalid config accepted: %+v", config)
		}
	}
}

func assertTokenRequest(t *testing.T, r *http.Request, scope string) {
	t.Helper()
	if r.Method != http.MethodPost {
		t.Errorf("method = %s", r.Method)
	}
	if err := r.ParseForm(); err != nil {
		t.Errorf("parse token request: %v", err)
		return
	}
	if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("scope") != scope {
		t.Errorf("token form = %v", r.Form)
	}
	clientID, clientSecret, basic := r.BasicAuth()
	if !basic {
		clientID, clientSecret = r.Form.Get("client_id"), r.Form.Get("client_secret")
	}
	if clientID != "runner" || clientSecret != "secret" {
		t.Errorf("client credentials = %q / %q", clientID, clientSecret)
	}
	if contentType := r.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		t.Errorf("Content-Type = %q", contentType)
	}
}
