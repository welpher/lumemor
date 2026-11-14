// Package config loads the server configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

// EnvGeodataDir is the environment variable pointing the server at
// the baked GeoNames reference directory (cities500.txt,
// admin1CodesASCII.txt, alternateNames-zh.txt): the startup load
// imports it into places/place_names and the in-memory place grid
// builds from the imported rows. Unset or empty keeps the feature
// honestly absent -- no import, no GPS resolution, the places rerun
// refuses to start -- which is the local-development default (the
// Docker image sets it to the bake it carries).
const EnvGeodataDir = "LUMEMOR_GEODATA_DIR"

// EnvUpdateCheckDisabled is the environment variable that turns the
// release check off at the deployment level: a stack that must never
// reach the outside world (a test harness, an air-gapped evaluation)
// sets it to a true value and the feature idles, without touching the
// instance's own settings. Unset or empty leaves the check under the
// admin settings switch alone.
const EnvUpdateCheckDisabled = "LUMEMOR_UPDATE_CHECK_DISABLED"

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
	// GeodataDir is the baked GeoNames reference directory the startup
	// load imports (places/place_names) and the place grid builds
	// from; empty keeps the place resolution feature off (the
	// local-development default -- the Docker image sets it).
	GeodataDir string
	// UploadLimitBytes is the maximum accepted size in bytes of a
	// single uploaded file.
	UploadLimitBytes int64
	// UpdateCheckDisabled is the deployment-level off switch of the
	// release check (the environment's own, distinct from the
	// admin settings switch).
	UpdateCheckDisabled bool
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
		Listen:              listen,
		DatabaseURL:         databaseURL,
		DataDir:             dataDir,
		ConfigDir:           configDir,
		GeodataDir:          os.Getenv(EnvGeodataDir),
		UploadLimitBytes:    uploadLimitBytes,
		UpdateCheckDisabled: parseTruthyEnv(EnvUpdateCheckDisabled),
	}, nil
}

// parseTruthyEnv reports whether the named environment variable is set
// to a true value ("1", "true", "yes", "on", case-insensitive). An
// unset, empty or unrecognized value is false -- the feature's default
// is the documented one, and a typo must not silently arm a switch.
func parseTruthyEnv(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
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
