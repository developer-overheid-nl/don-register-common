package typesense

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfigFromEnvUsesDefaultsAndEnvironment(t *testing.T) {
	t.Setenv("TYPESENSE_ENDPOINT", " ")
	t.Setenv("TYPESENSE_BASE_URL", " https://typesense.example.test/ ")
	t.Setenv("TYPESENSE_API_KEY", " secret ")
	t.Setenv("TYPESENSE_COLLECTION", "")
	t.Setenv("TYPESENSE_DETAIL_BASE_URL", "")
	t.Setenv("TYPESENSE_LANGUAGE", "")
	t.Setenv("TYPESENSE_ITEM_PRIORITY", "7")
	t.Setenv("TYPESENSE_DEFAULT_TAGS", " api, register, ")
	t.Setenv("ENABLE_TYPESENSE", "yes")

	cfg := LoadConfigFromEnv(Defaults{
		Collection:    "default_collection",
		DetailBaseURL: "https://developer.overheid.nl",
		Language:      "nl",
		ItemPriority:  2,
		DefaultTags:   []string{"default"},
	})

	want := Config{
		Endpoint:       "https://typesense.example.test/",
		APIKey:         "secret",
		Collection:     "default_collection",
		DetailBaseURL:  "https://developer.overheid.nl",
		Language:       "nl",
		ItemPriority:   7,
		DefaultTags:    []string{"api", "register"},
		FeatureEnabled: true,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("LoadConfigFromEnv() = %#v, want %#v", cfg, want)
	}
	if !cfg.Enabled() {
		t.Fatal("Enabled() = false, want true")
	}
}

func TestLoadConfigFromEnvFallsBackForInvalidValues(t *testing.T) {
	clearTypesenseEnv(t)
	t.Setenv("TYPESENSE_ITEM_PRIORITY", "not-a-number")
	t.Setenv("TYPESENSE_DEFAULT_TAGS", " , ")
	t.Setenv("ENABLE_TYPESENSE", "off")

	cfg := LoadConfigFromEnv(Defaults{DefaultTags: []string{"default"}})

	if cfg.Language != "nl" {
		t.Fatalf("Language = %q, want nl", cfg.Language)
	}
	if cfg.ItemPriority != 1 {
		t.Fatalf("ItemPriority = %d, want default 1", cfg.ItemPriority)
	}
	if !reflect.DeepEqual(cfg.DefaultTags, []string{"default"}) {
		t.Fatalf("DefaultTags = %#v, want default tag", cfg.DefaultTags)
	}
	if cfg.FeatureEnabled {
		t.Fatal("FeatureEnabled = true, want false")
	}
	if cfg.Enabled() {
		t.Fatal("Enabled() = true, want false when disabled")
	}
}

func TestBaseDocumentAndAppendUnique(t *testing.T) {
	cfg := Config{
		DetailBaseURL: "https://developer.overheid.nl/apis/",
		Language:      "en",
		ItemPriority:  5,
	}

	doc := BaseDocument(cfg, " api-123 ")
	want := map[string]any{
		"type":               "doc",
		"language":           "en",
		"item_priority":      5,
		"id":                 "api-123",
		"url":                "https://developer.overheid.nl/apis/api-123",
		"url_without_anchor": "https://developer.overheid.nl/apis/api-123",
		"anchor":             nil,
	}
	if !reflect.DeepEqual(doc, want) {
		t.Fatalf("BaseDocument() = %#v, want %#v", doc, want)
	}

	seen := map[string]struct{}{}
	values := AppendUnique(nil, " api ", seen)
	values = AppendUnique(values, "api", seen)
	values = AppendUnique(values, " ", seen)
	values = AppendUnique(values, "register", seen)
	if !reflect.DeepEqual(values, []string{"api", "register"}) {
		t.Fatalf("AppendUnique() values = %#v, want api and register", values)
	}
}

func TestUpsertDocumentPostsJSON(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", req.Method)
		}
		wantURL := "https://typesense.example.test/collections/apis%2Fv1/documents?action=upsert"
		if req.URL.String() != wantURL {
			t.Fatalf("url = %q, want %q", req.URL.String(), wantURL)
		}
		if got := req.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		if got := req.Header.Get("X-TYPESENSE-API-KEY"); got != "secret" {
			t.Fatalf("api key = %q, want secret", got)
		}

		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["id"] != "api-1" {
			t.Fatalf("body id = %v, want api-1", body["id"])
		}
		return textResponse(http.StatusOK, `{}`), nil
	})}
	cfg := Config{FeatureEnabled: true, Endpoint: "https://typesense.example.test/", APIKey: "secret", Collection: "apis/v1"}

	if err := UpsertDocument(context.Background(), client, cfg, map[string]any{"id": "api-1"}); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertDocumentErrors(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		err := UpsertDocument(context.Background(), nil, Config{}, map[string]any{"id": "api-1"})
		if !errors.Is(err, ErrDisabled) {
			t.Fatalf("error = %v, want ErrDisabled", err)
		}
	})

	t.Run("marshal", func(t *testing.T) {
		cfg := Config{FeatureEnabled: true, Endpoint: "https://typesense.example.test", APIKey: "secret", Collection: "apis"}
		err := UpsertDocument(context.Background(), nil, cfg, map[string]any{"bad": make(chan int)})
		if err == nil || !strings.Contains(err.Error(), "marshal payload") {
			t.Fatalf("error = %v, want marshal error", err)
		}
	})

	t.Run("request", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})}
		cfg := Config{FeatureEnabled: true, Endpoint: "https://typesense.example.test", APIKey: "secret", Collection: "apis"}
		err := UpsertDocument(context.Background(), client, cfg, map[string]any{"id": "api-1"})
		if err == nil || !strings.Contains(err.Error(), "request failed") {
			t.Fatalf("error = %v, want request failed error", err)
		}
	})

	t.Run("non success", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return textResponse(http.StatusBadRequest, "invalid document"), nil
		})}
		cfg := Config{FeatureEnabled: true, Endpoint: "https://typesense.example.test", APIKey: "secret", Collection: "apis"}
		err := UpsertDocument(context.Background(), client, cfg, map[string]any{"id": "api-1"})
		if err == nil || !strings.Contains(err.Error(), "status 400: invalid document") {
			t.Fatalf("error = %v, want status error", err)
		}
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func textResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func clearTypesenseEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"TYPESENSE_ENDPOINT",
		"TYPESENSE_BASE_URL",
		"TYPESENSE_API_KEY",
		"TYPESENSE_COLLECTION",
		"TYPESENSE_DETAIL_BASE_URL",
		"TYPESENSE_LANGUAGE",
		"TYPESENSE_ITEM_PRIORITY",
		"TYPESENSE_DEFAULT_TAGS",
		"ENABLE_TYPESENSE",
	} {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
}
