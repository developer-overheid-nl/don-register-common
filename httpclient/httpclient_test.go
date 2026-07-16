package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestCorsGetSetsOriginHeader(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", req.Method)
		}
		if got := req.Header.Get("Origin"); got != "https://frontend.example.test" {
			t.Fatalf("Origin = %q, want frontend URL", got)
		}
		return textResponse(http.StatusOK, "ok"), nil
	})}

	resp, err := CorsGet(client, "https://api.example.test/items", "https://frontend.example.test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close response body: %v", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestFetchOrganisationLabelBuildsURIAndReturnsDutchLabel(t *testing.T) {
	oldClient := HTTPClient
	defer func() { HTTPClient = oldClient }()

	HTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		wantURL := "https://identifier.overheid.nl/tooi/id/gemeente/gm9999"
		if req.URL.String() != wantURL {
			t.Fatalf("request URL = %q, want %q", req.URL.String(), wantURL)
		}
		if got := req.Header.Get("Accept"); got != "application/ld+json" {
			t.Fatalf("Accept = %q, want application/ld+json", got)
		}
		return jsonResponse(http.StatusOK, `[{"@graph":[{"@id":"https://identifier.overheid.nl/tooi/id/gemeente/gm9999","http://www.w3.org/2000/01/rdf-schema#label":[{"@value":"Gemeente Test","@language":"nl"},{"@value":"Test municipality","@language":"en"}]}]}]`), nil
	})}

	label, err := FetchOrganisationLabel(context.Background(), "gemeente", "gm9999")
	if err != nil {
		t.Fatal(err)
	}
	if label != "Gemeente Test" {
		t.Fatalf("label = %q, want Dutch label", label)
	}
}

func TestFetchOrganisationLabelSupportsFullURIAndFallbackLabel(t *testing.T) {
	oldClient := HTTPClient
	defer func() { HTTPClient = oldClient }()

	uri := "https://identifier.overheid.nl/tooi/id/ministerie/mnre1034"
	HTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `[{"@graph":[{"@id":"`+uri+`","http://www.w3.org/2000/01/rdf-schema#label":[{"@value":"English label","@language":"en"}]}]}]`), nil
	})}

	label, err := FetchOrganisationLabel(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if label != "English label" {
		t.Fatalf("label = %q, want fallback label", label)
	}
}

func TestFetchOrganisationLabelErrors(t *testing.T) {
	t.Run("invalid arguments", func(t *testing.T) {
		if _, err := FetchOrganisationLabel(context.Background(), "gemeente"); err == nil {
			t.Fatal("FetchOrganisationLabel() error = nil, want error")
		}
	})

	t.Run("request failure", func(t *testing.T) {
		oldClient := HTTPClient
		defer func() { HTTPClient = oldClient }()
		HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})}

		if _, err := FetchOrganisationLabel(context.Background(), "gemeente", "gm9999"); err == nil {
			t.Fatal("FetchOrganisationLabel() error = nil, want request error")
		}
	})

	t.Run("non ok status", func(t *testing.T) {
		oldClient := HTTPClient
		defer func() { HTTPClient = oldClient }()
		HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return textResponse(http.StatusNotFound, "missing"), nil
		})}

		if _, err := FetchOrganisationLabel(context.Background(), "gemeente", "gm9999"); err == nil {
			t.Fatal("FetchOrganisationLabel() error = nil, want not found error")
		}
	})

	t.Run("decode error", func(t *testing.T) {
		oldClient := HTTPClient
		defer func() { HTTPClient = oldClient }()
		HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{`), nil
		})}

		if _, err := FetchOrganisationLabel(context.Background(), "gemeente", "gm9999"); err == nil {
			t.Fatal("FetchOrganisationLabel() error = nil, want decode error")
		}
	})
}

func jsonResponse(status int, body string) *http.Response {
	resp := textResponse(status, body)
	resp.Header.Set("Content-Type", "application/ld+json")
	return resp
}

func textResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
