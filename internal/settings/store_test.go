package settings

import (
	"path/filepath"
	"testing"

	"file-transfer-client/internal/model"
)

func TestStoreLoadDefaultsWhenMissing(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "config.json"))

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.LastRemotePath != "/" {
		t.Fatalf("LastRemotePath = %q, want /", got.LastRemotePath)
	}
	if got.DefaultDownloadDir == "" {
		t.Fatalf("DefaultDownloadDir should not be empty")
	}
}

func TestStoreSaveAndLoad(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "config.json"))

	want := model.Settings{
		LastServerURL:      "http://127.0.0.1:8080",
		LastRemotePath:     "/releases/",
		DefaultDownloadDir: t.TempDir(),
		RecentServers: []string{
			"http://127.0.0.1:8080",
			"http://192.168.1.20:8080",
			"http://127.0.0.1:8080",
		},
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.LastServerURL != want.LastServerURL {
		t.Fatalf("LastServerURL = %q, want %q", got.LastServerURL, want.LastServerURL)
	}
	if got.LastRemotePath != want.LastRemotePath {
		t.Fatalf("LastRemotePath = %q, want %q", got.LastRemotePath, want.LastRemotePath)
	}
	if got.DefaultDownloadDir != want.DefaultDownloadDir {
		t.Fatalf("DefaultDownloadDir = %q, want %q", got.DefaultDownloadDir, want.DefaultDownloadDir)
	}
	if len(got.RecentServers) != 2 {
		t.Fatalf("RecentServers length = %d, want 2", len(got.RecentServers))
	}
}
