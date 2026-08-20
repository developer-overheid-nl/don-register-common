package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewJSONLoggerDefaultsToInfoAndAddsApplicationIdentity(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewJSONLogger(&output, "oss-register", "")
	if err != nil {
		t.Fatal(err)
	}

	logger.Debug("hidden diagnostic", "component", "test", "operation", "emit")
	logger.Info(
		"visible event",
		"component", "repository_active",
		"operation", "refresh",
		"repository_id", "repository-123",
	)

	events := decodeLogEvents(t, output.Bytes())
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	event := events[0]
	assertLogField(t, event, "level", "INFO")
	assertLogField(t, event, "msg", "visible event")
	assertLogField(t, event, "app", "oss-register")
	assertLogField(t, event, "component", "repository_active")
	assertLogField(t, event, "operation", "refresh")
	assertLogField(t, event, "repository_id", "repository-123")
	if _, ok := event["time"]; !ok {
		t.Fatal("event has no time field")
	}
}

func TestNewJSONLoggerHonoursSupportedMinimumLevels(t *testing.T) {
	tests := []struct {
		configured string
		minimum    slog.Level
	}{
		{configured: "debug", minimum: slog.LevelDebug},
		{configured: " INFO ", minimum: slog.LevelInfo},
		{configured: "Warn", minimum: slog.LevelWarn},
		{configured: "error", minimum: slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.configured, func(t *testing.T) {
			logger, err := NewJSONLogger(&bytes.Buffer{}, "oss-register", tt.configured)
			if err != nil {
				t.Fatal(err)
			}
			if !logger.Enabled(t.Context(), tt.minimum) {
				t.Fatalf("configured level %q does not enable %s", tt.configured, tt.minimum)
			}
			if tt.minimum > slog.LevelDebug && logger.Enabled(t.Context(), tt.minimum-4) {
				t.Fatalf("configured level %q enables lower level %s", tt.configured, tt.minimum-4)
			}
		})
	}
}

func TestNewJSONLoggerRejectsUnknownLevel(t *testing.T) {
	logger, err := NewJSONLogger(&bytes.Buffer{}, "oss-register", "verbose")

	if logger != nil {
		t.Fatal("logger is non-nil for an unsupported level")
	}
	if err == nil || err.Error() != `unsupported LOG_LEVEL "verbose"; use debug, info, warn or error` {
		t.Fatalf("error = %v, want unsupported-level error", err)
	}
}

func TestNewJSONLoggerRejectsBlankApplicationName(t *testing.T) {
	logger, err := NewJSONLogger(&bytes.Buffer{}, "  ", "info")

	if logger != nil {
		t.Fatal("logger is non-nil for a blank application name")
	}
	if err == nil || err.Error() != "app name must not be empty" {
		t.Fatalf("error = %v, want blank-app error", err)
	}
}

func TestGinMiddlewareRecordsRequestFieldsAndClassifiesServerErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger, err := NewJSONLogger(&output, "oss-register", "info")
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.Use(NewGinMiddleware(logger))
	router.GET("/ok/:id", func(c *gin.Context) {
		c.String(http.StatusAccepted, "ok")
	})
	router.GET("/failed", func(c *gin.Context) {
		c.String(http.StatusServiceUnavailable, "down")
	})

	for _, path := range []string{
		"/ok/repository-123?token=must-not-be-logged",
		"/failed",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
	}

	events := decodeLogEvents(t, output.Bytes())
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}

	okEvent := events[0]
	assertLogField(t, okEvent, "level", "INFO")
	assertLogField(t, okEvent, "msg", "HTTP request completed")
	assertLogField(t, okEvent, "app", "oss-register")
	assertLogField(t, okEvent, "component", "http_server")
	assertLogField(t, okEvent, "operation", "request")
	assertLogField(t, okEvent, "method", http.MethodGet)
	assertLogField(t, okEvent, "route", "/ok/:id")
	assertLogField(t, okEvent, "path", "/ok/repository-123")
	assertLogField(t, okEvent, "status_code", float64(http.StatusAccepted))
	assertLogField(t, okEvent, "response_bytes", float64(len("ok")))
	if duration, ok := okEvent["duration_ms"].(float64); !ok || duration < 0 {
		t.Fatalf("duration_ms = %#v, want non-negative number", okEvent["duration_ms"])
	}
	if strings.Contains(output.String(), "must-not-be-logged") {
		t.Fatal("query-string value leaked into request log")
	}

	failedEvent := events[1]
	assertLogField(t, failedEvent, "level", "ERROR")
	assertLogField(t, failedEvent, "status_code", float64(http.StatusServiceUnavailable))
}

func TestSlogWriterConvertsLineOutputToStructuredEvent(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewJSONLogger(&output, "oss-register", "debug")
	if err != nil {
		t.Fatal(err)
	}
	writer := NewSlogWriter(logger, slog.LevelWarn, "http_server", "recovery")

	line := "panic recovered\n"
	written, err := writer.Write([]byte(line))
	if err != nil {
		t.Fatal(err)
	}
	if written != len(line) {
		t.Fatalf("written = %d, want %d", written, len(line))
	}

	events := decodeLogEvents(t, output.Bytes())
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	event := events[0]
	assertLogField(t, event, "level", "WARN")
	assertLogField(t, event, "msg", "panic recovered")
	assertLogField(t, event, "component", "http_server")
	assertLogField(t, event, "operation", "recovery")
}

func TestCronLoggerAddsSchedulerContextAndPreservesSeverity(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewJSONLogger(&output, "api-register", "debug")
	if err != nil {
		t.Fatal(err)
	}
	cronLogger := NewCronLogger(logger, "harvest")

	cronLogger.Info("job delayed", "duration", "1s")
	cronLogger.Error(errors.New("panic value"), "job panicked", "stack", "trace")

	events := decodeLogEvents(t, output.Bytes())
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	assertLogField(t, events[0], "level", "INFO")
	assertLogField(t, events[0], "component", "harvest")
	assertLogField(t, events[0], "operation", "scheduler")
	assertLogField(t, events[0], "duration", "1s")
	assertLogField(t, events[1], "level", "ERROR")
	assertLogField(t, events[1], "error", "panic value")
	assertLogField(t, events[1], "stack", "trace")
}

func decodeLogEvents(t *testing.T, raw []byte) []map[string]any {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(raw))
	var events []map[string]any
	for decoder.More() {
		var event map[string]any
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return events
}

func assertLogField(t *testing.T, event map[string]any, key string, want any) {
	t.Helper()
	if got := event[key]; got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}
