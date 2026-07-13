package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyWebDist(t *testing.T) {
	t.Run("missing directory returns error", func(t *testing.T) {
		if err := verifyWebDist(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
			t.Fatal("expected error for missing web/dist directory, got nil")
		}
	})

	t.Run("directory without index.html returns error", func(t *testing.T) {
		if err := verifyWebDist(t.TempDir()); err == nil {
			t.Fatal("expected error when index.html is absent, got nil")
		}
	})

	t.Run("directory with index.html succeeds", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o600); err != nil {
			t.Fatalf("failed to write test index.html: %v", err)
		}
		if err := verifyWebDist(dir); err != nil {
			t.Fatalf("expected no error when index.html exists, got %v", err)
		}
	})
}
