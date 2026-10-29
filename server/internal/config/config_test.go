package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	// os.Getenv cannot distinguish unset from empty, so setting an empty
	// value exercises the "not provided" path.
	t.Setenv("LUMEMOR_LISTEN", "")
	t.Setenv("LUMEMOR_DATABASE_URL", "postgres://lumemor:lumemor@localhost:5432/lumemor")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.Listen != ":8244" {
		t.Errorf("cfg.Listen = %q, want %q", cfg.Listen, ":8244")
	}
	wantURL := "postgres://lumemor:lumemor@localhost:5432/lumemor"
	if cfg.DatabaseURL != wantURL {
		t.Errorf("cfg.DatabaseURL = %q, want %q", cfg.DatabaseURL, wantURL)
	}
}

func TestLoadListenOverride(t *testing.T) {
	t.Setenv("LUMEMOR_LISTEN", "127.0.0.1:9000")
	t.Setenv("LUMEMOR_DATABASE_URL", "postgres://lumemor:lumemor@localhost:5432/lumemor")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.Listen != "127.0.0.1:9000" {
		t.Errorf("cfg.Listen = %q, want %q", cfg.Listen, "127.0.0.1:9000")
	}
}

func TestLoadMissingDatabaseURL(t *testing.T) {
	t.Setenv("LUMEMOR_DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want an error when LUMEMOR_DATABASE_URL is missing")
	}
}
