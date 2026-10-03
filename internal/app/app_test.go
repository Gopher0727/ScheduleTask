package app

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRunRejectsMissingConfiguration(t *testing.T) {
	if err := Run(context.Background(), filepath.Join(t.TempDir(), "missing.toml")); err == nil {
		t.Fatal("missing configuration did not stop startup")
	}
}
