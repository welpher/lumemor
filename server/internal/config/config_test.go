package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// os.Getenv cannot distinguish unset from empty, so setting an empty
	// value exercises the "not provided" path.
	t.Setenv("LUMEMOR_LISTEN", "")
	t.Setenv("LUMEMOR_DATABASE_URL", "postgres://lumemor:lumemor@localhost:5432/lumemor")
	t.Setenv("LUMEMOR_DATA_DIR", "")
	t.Setenv("LUMEMOR_CONFIG_DIR", "")
	t.Setenv("LUMEMOR_UPLOAD_LIMIT_BYTES", "")
	t.Setenv("LUMEMOR_GEODATA_DIR", "")

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
	if cfg.DataDir != "/var/lib/lumemor/data" {
		t.Errorf("cfg.DataDir = %q, want %q", cfg.DataDir, "/var/lib/lumemor/data")
	}
	if cfg.ConfigDir != "/var/lib/lumemor/config" {
		t.Errorf("cfg.ConfigDir = %q, want %q", cfg.ConfigDir, "/var/lib/lumemor/config")
	}
	if cfg.UploadLimitBytes != 2147483648 {
		t.Errorf("cfg.UploadLimitBytes = %d, want %d", cfg.UploadLimitBytes, 2147483648)
	}
	// The geodata override defaults to empty: the local-development
	// default keeps the place resolution honestly off (the Docker
	// image sets the variable to the bake it carries).
	if cfg.GeodataDir != "" {
		t.Errorf("cfg.GeodataDir = %q, want empty", cfg.GeodataDir)
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

func TestLoadStorageDirOverrides(t *testing.T) {
	t.Setenv("LUMEMOR_DATABASE_URL", "postgres://lumemor:lumemor@localhost:5432/lumemor")
	t.Setenv("LUMEMOR_DATA_DIR", "/srv/photos/data")
	t.Setenv("LUMEMOR_CONFIG_DIR", "/srv/lumemor-config")
	t.Setenv("LUMEMOR_GEODATA_DIR", "/usr/local/share/lumemor/geo")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.DataDir != "/srv/photos/data" {
		t.Errorf("cfg.DataDir = %q, want %q", cfg.DataDir, "/srv/photos/data")
	}
	if cfg.ConfigDir != "/srv/lumemor-config" {
		t.Errorf("cfg.ConfigDir = %q, want %q", cfg.ConfigDir, "/srv/lumemor-config")
	}
	if cfg.GeodataDir != "/usr/local/share/lumemor/geo" {
		t.Errorf("cfg.GeodataDir = %q, want %q", cfg.GeodataDir, "/usr/local/share/lumemor/geo")
	}
}

func TestLoadUploadLimitBytes(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr string
	}{
		{name: "unset falls back to default", value: "", want: 2147483648},
		{name: "override", value: "1048576", want: 1048576},
		{name: "zero allowed", value: "0", want: 0},
		{name: "negative rejected", value: "-1", wantErr: "LUMEMOR_UPLOAD_LIMIT_BYTES"},
		{name: "non-numeric rejected", value: "2GB", wantErr: "LUMEMOR_UPLOAD_LIMIT_BYTES"},
		{name: "overflow rejected", value: "9223372036854775808", wantErr: "LUMEMOR_UPLOAD_LIMIT_BYTES"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LUMEMOR_DATABASE_URL", "postgres://lumemor:lumemor@localhost:5432/lumemor")
			t.Setenv("LUMEMOR_UPLOAD_LIMIT_BYTES", tt.value)

			cfg, err := Load()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Load() error = nil, want an error for LUMEMOR_UPLOAD_LIMIT_BYTES=%q", tt.value)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Load() error = %v, want it to mention %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			if cfg.UploadLimitBytes != tt.want {
				t.Errorf("cfg.UploadLimitBytes = %d, want %d", cfg.UploadLimitBytes, tt.want)
			}
		})
	}
}
