// Package config loads the server configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// EnvListen is the environment variable overriding the HTTP listen address.
const EnvListen = "LUMEMOR_LISTEN"

// EnvDatabaseURL is the required environment variable holding the
// PostgreSQL connection string.
const EnvDatabaseURL = "LUMEMOR_DATABASE_URL"

// EnvDataDir is the environment variable overriding the photo data
// root: originals, upload staging, and derived media files.
const EnvDataDir = "LUMEMOR_DATA_DIR"

// EnvConfigDir is the environment variable overriding the instance
// configuration root: ML models today, settings files later.
const EnvConfigDir = "LUMEMOR_CONFIG_DIR"

// EnvUploadLimitBytes is the environment variable overriding the
// maximum accepted size of a single uploaded file.
const EnvUploadLimitBytes = "LUMEMOR_UPLOAD_LIMIT_BYTES"

// DefaultListen is the listen address used when EnvListen is unset or empty.
const DefaultListen = ":8244"

// DefaultDataDir is the photo data root used when EnvDataDir is unset
// or empty.
const DefaultDataDir = "/var/lib/lumemor/data"

// DefaultConfigDir is the configuration root used when EnvConfigDir is
// unset or empty.
const DefaultConfigDir = "/var/lib/lumemor/config"

// DefaultUploadLimitBytes is the per-file upload size limit used when
// EnvUploadLimitBytes is unset or empty: 2 GiB.
const DefaultUploadLimitBytes int64 = 2147483648

// Config holds the runtime configuration of the lumemor server.
type Config struct {
	// Listen is the address the HTTP server binds to, for example ":8244".
	Listen string
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string
	// DataDir is the root directory holding photo data: content
	// addressed originals, upload staging, and derived media files.
	DataDir string
	// ConfigDir is the root directory holding instance configuration:
	// ML models today, settings files later.
	ConfigDir string
	// UploadLimitBytes is the maximum accepted size in bytes of a
	// single uploaded file.
	UploadLimitBytes int64
}

// Load reads the configuration from the environment. It returns an
// error when EnvDatabaseURL is missing or when EnvUploadLimitBytes is
// not a non-negative integer. Listen, DataDir and ConfigDir fall back
// to their defaults when the corresponding environment variables are
// unset or empty.
func Load() (Config, error) {
	databaseURL := os.Getenv(EnvDatabaseURL)
	if databaseURL == "" {
		return Config{}, fmt.Errorf("%s is required", EnvDatabaseURL)
	}

	listen := os.Getenv(EnvListen)
	if listen == "" {
		listen = DefaultListen
	}

	dataDir := os.Getenv(EnvDataDir)
	if dataDir == "" {
		dataDir = DefaultDataDir
	}

	configDir := os.Getenv(EnvConfigDir)
	if configDir == "" {
		configDir = DefaultConfigDir
	}

	uploadLimitBytes, err := loadUploadLimitBytes()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Listen:           listen,
		DatabaseURL:      databaseURL,
		DataDir:          dataDir,
		ConfigDir:        configDir,
		UploadLimitBytes: uploadLimitBytes,
	}, nil
}

// loadUploadLimitBytes parses EnvUploadLimitBytes. An unset or empty
// variable keeps DefaultUploadLimitBytes; negative values are rejected
// because a size limit cannot be negative.
func loadUploadLimitBytes() (int64, error) {
	raw := os.Getenv(EnvUploadLimitBytes)
	if raw == "" {
		return DefaultUploadLimitBytes, nil
	}
	limit, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: parse %q as an integer: %w", EnvUploadLimitBytes, raw, err)
	}
	if limit < 0 {
		return 0, fmt.Errorf("%s must not be negative, got %d", EnvUploadLimitBytes, limit)
	}
	return limit, nil
}
