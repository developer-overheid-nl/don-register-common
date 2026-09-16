// Package auth provides reusable authentication for DON service clients.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// ClientCredentialsConfig describes an OAuth 2.0 client_credentials grant.
type ClientCredentialsConfig struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// NewClientCredentialsHTTPClient returns a copy of base that obtains and renews
// OAuth 2.0 access tokens using the client_credentials grant.
func NewClientCredentialsHTTPClient(ctx context.Context, config ClientCredentialsConfig, base *http.Client) (*http.Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	config.TokenURL = strings.TrimSpace(config.TokenURL)
	config.ClientID = strings.TrimSpace(config.ClientID)
	config.ClientSecret = strings.TrimSpace(config.ClientSecret)
	if config.TokenURL == "" || config.ClientID == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("token URL, client ID and client secret are required")
	}
	u, err := url.Parse(config.TokenURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Fragment != "" {
		return nil, fmt.Errorf("token URL must be an HTTP(S) URL without embedded credentials or fragment")
	}
	scopes := make([]string, 0, len(config.Scopes))
	for _, scope := range config.Scopes {
		if scope = strings.TrimSpace(scope); scope != "" {
			scopes = append(scopes, scope)
		}
	}
	if base == nil {
		base = &http.Client{Timeout: 30 * time.Second}
	}
	baseTransport := base.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	// Token requests use the supplied client's transport and policy without
	// recursively applying the OAuth transport to the token endpoint itself.
	tokenClient := &http.Client{
		Transport:     baseTransport,
		CheckRedirect: base.CheckRedirect,
		Jar:           base.Jar,
		Timeout:       base.Timeout,
	}
	tokenContext := context.WithValue(ctx, oauth2.HTTPClient, tokenClient)
	tokenSource := (&clientcredentials.Config{
		ClientID: config.ClientID, ClientSecret: config.ClientSecret,
		TokenURL: config.TokenURL, Scopes: scopes,
	}).TokenSource(tokenContext)

	return &http.Client{
		Transport:     &oauth2.Transport{Source: tokenSource, Base: baseTransport},
		CheckRedirect: base.CheckRedirect,
		Jar:           base.Jar,
		Timeout:       base.Timeout,
	}, nil
}
