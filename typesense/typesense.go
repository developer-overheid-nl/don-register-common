package typesense

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var ErrDisabled = errors.New("typesense indexing disabled: missing endpoint, api key or collection name")

type Config struct {
	Endpoint       string
	APIKey         string
	Collection     string
	DetailBaseURL  string
	Language       string
	ItemPriority   int
	DefaultTags    []string
	FeatureEnabled bool
}

type Defaults struct {
	Collection    string
	DetailBaseURL string
	Language      string
	ItemPriority  int
	DefaultTags   []string
}

func LoadConfigFromEnv(defaults Defaults) Config {
	endpoint := strings.TrimSpace(os.Getenv("TYPESENSE_ENDPOINT"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("TYPESENSE_BASE_URL"))
	}

	apiKey := strings.TrimSpace(os.Getenv("TYPESENSE_API_KEY"))
	collection := strings.TrimSpace(os.Getenv("TYPESENSE_COLLECTION"))
	if collection == "" {
		collection = defaults.Collection
	}

	detailBase := strings.TrimSpace(os.Getenv("TYPESENSE_DETAIL_BASE_URL"))
	if detailBase == "" {
		detailBase = defaults.DetailBaseURL
	}

	language := strings.TrimSpace(os.Getenv("TYPESENSE_LANGUAGE"))
	if language == "" {
		language = defaults.Language
	}
	if language == "" {
		language = "nl"
	}

	itemPriority := defaults.ItemPriority
	if itemPriority == 0 {
		itemPriority = 1
	}
	if raw := strings.TrimSpace(os.Getenv("TYPESENSE_ITEM_PRIORITY")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			itemPriority = v
		}
	}

	return Config{
		Endpoint:       endpoint,
		APIKey:         apiKey,
		Collection:     collection,
		DetailBaseURL:  detailBase,
		Language:       language,
		ItemPriority:   itemPriority,
		DefaultTags:    parseDefaultTags(defaults.DefaultTags),
		FeatureEnabled: isFeatureEnabled(),
	}
}

func (c Config) Enabled() bool {
	return c.FeatureEnabled && c.Endpoint != "" && c.APIKey != "" && c.Collection != ""
}

func isFeatureEnabled() bool {
	raw := strings.TrimSpace(os.Getenv("ENABLE_TYPESENSE"))
	if raw == "" {
		return true
	}
	switch strings.ToLower(raw) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func parseDefaultTags(defaultTags []string) []string {
	raw := os.Getenv("TYPESENSE_DEFAULT_TAGS")
	if strings.TrimSpace(raw) == "" {
		return append([]string(nil), defaultTags...)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), defaultTags...)
	}
	return out
}

func UpsertDocument(ctx context.Context, client *http.Client, cfg Config, document map[string]any) (err error) {
	if !cfg.Enabled() {
		return ErrDisabled
	}
	if client == nil {
		client = http.DefaultClient
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("typesense: marshal payload: %w", err)
	}

	base := strings.TrimRight(cfg.Endpoint, "/")
	target := fmt.Sprintf("%s/collections/%s/documents?action=upsert", base, url.PathEscape(cfg.Collection))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("typesense: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TYPESENSE-API-KEY", cfg.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("typesense: request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("typesense: close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("typesense: read error response: %w", readErr)
		}
		return fmt.Errorf("typesense: indexing failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func BaseDocument(cfg Config, id string) map[string]any {
	doc := map[string]any{
		"type":          "doc",
		"language":      cfg.Language,
		"item_priority": cfg.ItemPriority,
	}
	if id = strings.TrimSpace(id); id != "" {
		doc["id"] = id
	}
	detailBase := strings.TrimRight(cfg.DetailBaseURL, "/")
	if detailBase != "" && id != "" {
		detailURL := fmt.Sprintf("%s/%s", detailBase, id)
		doc["url"] = detailURL
		doc["url_without_anchor"] = detailURL
		doc["anchor"] = nil
	}
	return doc
}

func AppendUnique(values []string, value string, seen map[string]struct{}) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	if _, ok := seen[value]; ok {
		return values
	}
	seen[value] = struct{}{}
	return append(values, value)
}
