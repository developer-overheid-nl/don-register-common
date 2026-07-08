package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/developer-overheid-nl/don-register-common/problem"
	"github.com/gin-gonic/gin"
)

func TestNewEngineAddsAPIVersionAndDefaultCORSHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := NewEngine("v1", CORSOptions{})
	engine.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/ok", nil)
	req.Header.Set("Origin", "https://frontend.example.test")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Origin,Content-Length,Content-Type,Authorization,Api-Version" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want default headers", got)
	}

	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if got := rec.Header().Get("API-Version"); got != "v1" {
		t.Fatalf("API-Version = %q, want v1", got)
	}
}

func TestNewEngineUsesCustomCORSHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := NewEngine("v2", CORSOptions{
		AllowHeaders:  []string{"X-Custom"},
		ExposeHeaders: []string{"X-Expose"},
	})
	engine.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "https://frontend.example.test")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("API-Version"); got != "v2" {
		t.Fatalf("API-Version = %q, want v2", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Expose" {
		t.Fatalf("Access-Control-Expose-Headers = %q, want X-Expose", got)
	}
}

func TestInstallProblemHandlersReturnsProblemResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := NewEngine("v1", CORSOptions{})
	InstallProblemHandlers(engine, "v1")
	engine.GET("/known", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("no route", func(t *testing.T) {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))

		assertProblemResponse(t, rec, http.StatusNotFound, "Resource does not exist")
	})

	t.Run("method not allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/known", nil))

		assertProblemResponse(t, rec, http.StatusMethodNotAllowed, "Method not allowed")
	})
}

func assertProblemResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, title string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d", rec.Code, status)
	}
	if got := rec.Header().Get("API-Version"); got != "v1" {
		t.Fatalf("API-Version = %q, want v1", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", got)
	}

	var body problem.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != status || body.Title != title {
		t.Fatalf("body = %#v, want status %d title %q", body, status, title)
	}
}
