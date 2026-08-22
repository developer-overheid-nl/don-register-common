package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestConfigureLoggingSuppressesExpectedMissesAndStructuresDatabaseErrors(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	db := &gorm.DB{Config: &gorm.Config{}}
	ConfigureLogging(db, logger)

	db.Logger.Trace(t.Context(), time.Now(), func() (string, int64) {
		return "SELECT * FROM repositories WHERE id = $1", 0
	}, gorm.ErrRecordNotFound)
	if output.Len() != 0 {
		t.Fatalf("expected lookup miss produced log output: %s", output.String())
	}

	db.Logger.Trace(t.Context(), time.Now(), func() (string, int64) {
		return "SELECT * FROM repositories WHERE id = $1", 0
	}, errors.New("database unavailable"))

	event := decodeDatabaseEvent(t, output.Bytes())
	assertDatabaseLogField(t, event, "level", "ERROR")
	assertDatabaseLogField(t, event, "msg", "SQL executed")
	assertDatabaseLogField(t, event, "component", "database")
	assertDatabaseLogField(t, event, "operation", "query")
}

func TestConfigureDefaultLoggingStructuresStartupDatabaseErrors(t *testing.T) {
	previous := gormlogger.Default
	t.Cleanup(func() {
		gormlogger.Default = previous
	})

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	ConfigureDefaultLogging(logger)

	gormlogger.Default.Trace(t.Context(), time.Now(), func() (string, int64) {
		return "ALTER TABLE repositories ADD COLUMN archived boolean", 0
	}, errors.New("migration failed"))

	event := decodeDatabaseEvent(t, output.Bytes())
	assertDatabaseLogField(t, event, "level", "ERROR")
	assertDatabaseLogField(t, event, "component", "database")
}

func TestConfigureLoggingFiltersDatabaseParameters(t *testing.T) {
	db := &gorm.DB{Config: &gorm.Config{}}
	ConfigureLogging(db, slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))

	filter, ok := db.Logger.(gorm.ParamsFilter)
	if !ok {
		t.Fatal("configured logger does not implement gorm.ParamsFilter")
	}
	sql, params := filter.ParamsFilter(context.Background(), "SELECT * FROM repositories WHERE id = $1", "secret-id")
	if sql != "SELECT * FROM repositories WHERE id = $1" {
		t.Fatalf("sql = %q, want parameterized query", sql)
	}
	if params != nil {
		t.Fatalf("params = %#v, want nil", params)
	}
}

func decodeDatabaseEvent(t *testing.T, raw []byte) map[string]any {
	t.Helper()

	var event map[string]any
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatal(err)
	}
	return event
}

func assertDatabaseLogField(t *testing.T, event map[string]any, key string, want any) {
	t.Helper()
	if got := event[key]; got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}
