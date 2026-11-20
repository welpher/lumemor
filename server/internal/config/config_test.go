package config

import (
	"strings"
	"testing"
	"time"
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

func TestLoadFolderSyncInterval(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{"unset keeps the queue default", "", 0, false},
		{"zero disables the sweep", "0", -1, false},
		{"off disables the sweep", "off", -1, false},
		{"a positive duration passes", "90s", 90 * time.Second, false},
		{"minutes parse too", "5m", 5 * time.Minute, false},
		{"nonsense fails the boot", "nonsense", 0, true},
		{"negative fails the boot", "-5s", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvFolderSyncInterval, tc.raw)
			got, err := loadFolderSyncInterval()
			if tc.wantErr != (err != nil) {
				t.Fatalf("loadFolderSyncInterval(%q) error = %v, wantErr %v", tc.raw, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("loadFolderSyncInterval(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

// TestLoadFolderWatch pins the watcher's switch: on when unset (the
// shipped default -- the accelerator is what makes an active house see a
// change in seconds), off for the documented falsy spellings, and a boot
// failure for anything else: an on-by-default switch must not be
// disarmed by a typo any more than an off-by-default one may be armed.
func TestLoadFolderWatch(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    bool
		wantErr bool
	}{
		{"unset keeps the accelerator on", "", true, false},
		{"on", "on", true, false},
		{"1", "1", true, false},
		{"zero disables it", "0", false, false},
		{"off disables it", "off", false, false},
		{"nonsense fails the boot", "maybe", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvFolderWatch, tc.raw)
			got, err := loadFolderWatch()
			if tc.wantErr != (err != nil) {
				t.Fatalf("loadFolderWatch(%q) error = %v, wantErr %v", tc.raw, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("loadFolderWatch(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

// TestLoadPublicURL pins the canonical-origin rule: unset or empty
// stays empty (the request's own TLS state decides), a bare http(s)
// origin is normalized without its trailing slash, and anything that
// is not an origin -- a path, a query, a fragment, credentials, a
// non-http scheme -- fails the boot naming the variable.
func TestLoadPublicURL(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"unset", "", "", false},
		{"https origin", "https://photos.example.com", "https://photos.example.com", false},
		{"http origin", "http://photos.example.com", "http://photos.example.com", false},
		{"port kept", "https://photos.example.com:8443", "https://photos.example.com:8443", false},
		{"trailing slash normalized", "https://photos.example.com/", "https://photos.example.com", false},
		{"surrounding space trimmed", "  https://photos.example.com ", "https://photos.example.com", false},
		{"a path is not an origin", "https://photos.example.com/photos", "", true},
		{"a query is not an origin", "https://photos.example.com?x=1", "", true},
		{"a fragment is not an origin", "https://photos.example.com#top", "", true},
		{"credentials are refused", "https://user:pass@photos.example.com", "", true},
		{"a non-http scheme is refused", "ftp://photos.example.com", "", true},
		{"a relative value is refused", "photos.example.com", "", true},
		{"a hostless value is refused", "https://", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvPublicURL, tc.raw)
			got, err := loadPublicURL()
			if tc.wantErr != (err != nil) {
				t.Fatalf("loadPublicURL(%q) error = %v, wantErr %v", tc.raw, err, tc.wantErr)
			}
			if err != nil {
				if !strings.Contains(err.Error(), EnvPublicURL) {
					t.Errorf("error = %v, want it to name %s", err, EnvPublicURL)
				}
				return
			}
			if got != tc.want {
				t.Errorf("loadPublicURL(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestLoadTrustedProxies pins the CIDR list: unset or empty trusts the
// socket peer only, v4 and v6 prefixes parse (masked, so a host-bits
// value names its network), a bare address names one host, and any
// malformed or empty entry fails the boot naming the variable.
func TestLoadTrustedProxies(t *testing.T) {
	t.Run("unset trusts the socket peer only", func(t *testing.T) {
		t.Setenv(EnvTrustedProxies, "")
		got, err := loadTrustedProxies()
		if err != nil {
			t.Fatalf("loadTrustedProxies() error = %v, want nil", err)
		}
		if len(got) != 0 {
			t.Errorf("loadTrustedProxies() = %v, want no prefixes", got)
		}
	})

	t.Run("v4 and v6 prefixes plus a bare host", func(t *testing.T) {
		t.Setenv(EnvTrustedProxies, "10.0.0.0/8, 192.168.1.5/24 , 2001:db8::/32, ::1")
		got, err := loadTrustedProxies()
		if err != nil {
			t.Fatalf("loadTrustedProxies() error = %v, want nil", err)
		}
		want := []string{"10.0.0.0/8", "192.168.1.0/24", "2001:db8::/32", "::1/128"}
		if len(got) != len(want) {
			t.Fatalf("loadTrustedProxies() = %v, want %d prefixes", got, len(want))
		}
		for i, prefix := range got {
			if prefix.String() != want[i] {
				t.Errorf("prefix %d = %q, want %q", i, prefix.String(), want[i])
			}
		}
	})

	for _, raw := range []string{"10.0.0.0/8,nonsense", "10.0.0.0/8,,192.168.0.0/16", "300.0.0.0/8", "10.0.0.0/33"} {
		t.Run("refuses "+raw, func(t *testing.T) {
			t.Setenv(EnvTrustedProxies, raw)
			if _, err := loadTrustedProxies(); err == nil {
				t.Fatalf("loadTrustedProxies(%q) error = nil, want a boot failure", raw)
			} else if !strings.Contains(err.Error(), EnvTrustedProxies) {
				t.Errorf("error = %v, want it to name %s", err, EnvTrustedProxies)
			}
		})
	}
}

// TestLoadDeploymentDefaults pins the unset case through Load: both
// deployment knobs stay empty, so the direct/LAN behaviour is
// untouched.
func TestLoadDeploymentDefaults(t *testing.T) {
	t.Setenv(EnvDatabaseURL, "postgres://lumemor:lumemor@localhost:5432/lumemor")
	t.Setenv(EnvPublicURL, "")
	t.Setenv(EnvTrustedProxies, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.PublicURL != "" {
		t.Errorf("cfg.PublicURL = %q, want empty", cfg.PublicURL)
	}
	if len(cfg.TrustedProxies) != 0 {
		t.Errorf("cfg.TrustedProxies = %v, want empty", cfg.TrustedProxies)
	}
}

// TestLoadRefusesMalformedDeployment pins that a malformed knob stops
// the boot through Load, not only through the helpers.
func TestLoadRefusesMalformedDeployment(t *testing.T) {
	t.Setenv(EnvDatabaseURL, "postgres://lumemor:lumemor@localhost:5432/lumemor")
	t.Setenv(EnvPublicURL, "https://photos.example.com/photos")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), EnvPublicURL) {
		t.Fatalf("Load() error = %v, want it to name %s", err, EnvPublicURL)
	}

	t.Setenv(EnvPublicURL, "")
	t.Setenv(EnvTrustedProxies, "not-a-cidr")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), EnvTrustedProxies) {
		t.Fatalf("Load() error = %v, want it to name %s", err, EnvTrustedProxies)
	}
}
