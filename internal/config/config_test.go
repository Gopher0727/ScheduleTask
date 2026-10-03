package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCanRetryAfterMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if _, err := Load(path); err == nil {
		t.Fatal("missing config accepted")
	}
	contents, err := os.ReadFile("../../config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.Port != 8082 {
		t.Fatal("successful retry left config invalid")
	}
}

func TestLoadRejectsInvalidRanges(t *testing.T) {
	contents, err := os.ReadFile("../../config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{
		{"port = 8082", "port = 65535"},
		{"bucket_num = 5", "bucket_num = 0"},
		{"zrange_gap_seconds = 1", "zrange_gap_seconds = 0"},
		{"migrate_step_minutes = 5", "migrate_step_minutes = 0"},
	} {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(strings.Replace(string(contents), pair[0], pair[1], 1)), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("accepted invalid setting %q", pair[1])
		}
	}
}

func TestLoadEnvironmentOverrides(t *testing.T) {
	t.Setenv("SCHEDULETASK_PORT", "9000")
	t.Setenv("SCHEDULETASK_POSTGRES_DSN", "host=postgres user=custom dbname=custom")
	t.Setenv("SCHEDULETASK_REDIS_ADDR", "redis:6380")
	t.Setenv("SCHEDULETASK_REDIS_PASSWORD", "custom")
	cfg, err := Load("../../config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9000 || cfg.Postgres.DSN != "host=postgres user=custom dbname=custom" || cfg.Redis.Addr != "redis:6380" || cfg.Redis.Password != "custom" {
		t.Fatal("environment overrides were not applied")
	}
}
