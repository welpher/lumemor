// Package config loads the server configuration from the environment.
package config

import (
	"fmt"
	"os"
)

// EnvListen is the environment variable overriding the HTTP listen address.
const EnvListen = "LUMEMOR_LISTEN"

// EnvDatabaseURL is the required environment variable holding the
// PostgreSQL connection string.
const EnvDatabaseURL = "LUMEMOR_DATABASE_URL"

// DefaultListen is the listen address used when EnvListen is unset or empty.
const DefaultListen = ":8244"

// Config holds the runtime configuration of the lumemor server.
type Config struct {
	// Listen is the address the HTTP server binds to, for example ":8244".
	Listen string
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string
}

// Load reads the configuration from the environment. It returns an error
// when EnvDatabaseURL is missing. Listen falls back to DefaultListen when
// EnvListen is unset or empty.
func Load() (Config, error) {
	databaseURL := os.Getenv(EnvDatabaseURL)
	if databaseURL == "" {
		return Config{}, fmt.Errorf("%s is required", EnvDatabaseURL)
	}

	listen := os.Getenv(EnvListen)
	if listen == "" {
		listen = DefaultListen
	}

	return Config{Listen: listen, DatabaseURL: databaseURL}, nil
}
