// Package config loads the server configuration from the environment.
package config

import (
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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

// EnvFolderSyncInterval is the environment variable overriding the
// folder live sync's cadence: how often the periodic directory-snapshot
// diff re-reads every directory album's source folder so its wall
// follows the folder's adds, removes and changes. A Go duration
// ("90s", "5m"); "0" or "off" disables the periodic sweep at the
// deployment level (a family instance that references no folder pays
// nothing), and unset or empty keeps the queue's own 60s default. The
// e2e harness shortens it so a scene observes the diff inside the
// test's bounds.
const EnvFolderSyncInterval = "LUMEMOR_FOLDER_SYNC_INTERVAL"

// EnvFolderWatch is the environment variable that turns the folder
// live sync's filesystem-watcher accelerator off: the watcher only
// marks an album dirty when its source folder changes, and the
// periodic diff remains the decider either way. On by default -- the
// accelerator is what makes an active house see a dropped photo in
// seconds instead of one cadence window -- and "0" or "off" leaves a
// deployment with the gated diff alone (the shape a platform whose
// mounts carry no events behaves like anyway).
const EnvFolderWatch = "LUMEMOR_FOLDER_WATCH"

// EnvPublicURL is the deployment's canonical public origin -- the
// address browsers actually use, scheme and host only, for example
// "https://photos.example.com". It is the deployment's own input
// because a reverse proxy can terminate TLS and rewrite the Host
// header, which the request alone cannot reveal. Setting it to an
// https origin makes the session cookie Secure and turns on HSTS;
// unset or empty keeps the request's own TLS state (the LAN HTTP
// deployment is unchanged).
const EnvPublicURL = "LUMEMOR_PUBLIC_URL"

// EnvTrustedProxies is the deployment's list of reverse-proxy peers:
// a comma-separated CIDR list (an address without a prefix names that
// single host). A request whose socket peer matches may assert its
// client address through X-Forwarded-For and its scheme through
// X-Forwarded-Proto; a peer outside the list is trusted for neither,
// so a direct client cannot forge either header. Unset or empty
// trusts the socket peer only (the LAN HTTP deployment is unchanged).
const EnvTrustedProxies = "LUMEMOR_TRUSTED_PROXIES"

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
	// FolderSyncInterval is the folder live sync's cadence. Zero keeps
	// the job queue's default window (60s); a negative value means
	// disabled, the shape the queue's periodic bundle reads.
	FolderSyncInterval time.Duration
	// FolderWatch turns the folder live sync's filesystem watcher on:
	// the accelerator that marks a changed album dirty so the sweep
	// runs within seconds instead of one cadence window. It never
	// ingests anything itself; the diff's stamp comparison stays the
	// only decider. False leaves the gated diff alone.
	FolderWatch bool
	// PublicURL is the deployment's canonical public origin
	// ("https://photos.example.com"), or empty when the instance is
	// reached directly. It is already normalized: scheme and host
	// only, no trailing slash.
	PublicURL string
	// TrustedProxies are the reverse-proxy peers allowed to assert a
	// forwarded client address and scheme. Empty trusts the socket
	// peer only.
	TrustedProxies []netip.Prefix
}

// Load reads the configuration from the environment. It returns an
// error when EnvDatabaseURL is missing or when EnvUploadLimitBytes is
// not a non-negative integer. Listen, DataDir and ConfigDir fall back
// to their defaults when the corresponding environment variables are
// unset or empty; EnvPublicURL and EnvTrustedProxies are validated
// where set and stay empty otherwise.
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

	folderSyncInterval, err := loadFolderSyncInterval()
	if err != nil {
		return Config{}, err
	}

	folderWatch, err := loadFolderWatch()
	if err != nil {
		return Config{}, err
	}

	publicURL, err := loadPublicURL()
	if err != nil {
		return Config{}, err
	}

	trustedProxies, err := loadTrustedProxies()
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
		FolderSyncInterval:  folderSyncInterval,
		FolderWatch:         folderWatch,
		PublicURL:           publicURL,
		TrustedProxies:      trustedProxies,
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

// loadFolderSyncInterval parses EnvFolderSyncInterval. An unset or
// empty variable keeps the queue's default (zero, "use the default");
// "0" and "off" disable the sweep (the negative marker); anything else
// must be a positive Go duration, so a typo fails the boot loudly
// instead of silently disabling a feature or walking every folder in a
// tight loop.
func loadFolderSyncInterval() (time.Duration, error) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(EnvFolderSyncInterval)))
	switch raw {
	case "":
		return 0, nil
	case "0", "off":
		return -1, nil
	}
	interval, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: parse %q as a duration: %w", EnvFolderSyncInterval, raw, err)
	}
	if interval <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration, got %q", EnvFolderSyncInterval, raw)
	}
	return interval, nil
}

// loadFolderWatch parses EnvFolderWatch. The accelerator is on when the
// variable is unset or empty (the shipped default), "0"/"off" (and the
// common falsy spellings) turn it off, and anything else fails the boot
// loudly: an on-by-default switch must not be disarmed by a typo any
// more than an off-by-default one may be armed by it.
func loadFolderWatch() (bool, error) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(EnvFolderWatch)))
	switch raw {
	case "", "1", "true", "yes", "on":
		return true, nil
	case "0", "off", "false", "no":
		return false, nil
	}
	return false, fmt.Errorf("%s: parse %q: want 0/off to disable the folder watcher, or 1/on to enable it", EnvFolderWatch, raw)
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

// loadPublicURL parses EnvPublicURL. An unset or empty variable stays
// empty (the direct/LAN deployment, where the request's own TLS state
// decides). A set value must be a bare absolute http(s) origin --
// scheme, host, optional port, and nothing else: a path, query,
// fragment or embedded credentials cannot name an origin, and
// silently dropping one would make the cookie/HSTS answer disagree
// with the address the browser actually uses.
func loadPublicURL() (string, error) {
	raw := strings.TrimSpace(os.Getenv(EnvPublicURL))
	if raw == "" {
		return "", nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%s: parse %q: %w", EnvPublicURL, raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%s must be an absolute http(s) origin, got %q", EnvPublicURL, raw)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("%s must name a host, got %q", EnvPublicURL, raw)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%s must not embed credentials, got %q", EnvPublicURL, raw)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", fmt.Errorf("%s must be a bare origin (scheme://host), got %q", EnvPublicURL, raw)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

// loadTrustedProxies parses EnvTrustedProxies. An unset or empty
// variable trusts the socket peer only. Otherwise every comma-separated
// entry must be a CIDR prefix (v4 or v6), or a bare address naming a
// single host; a malformed entry fails the boot instead of silently
// shrinking or widening the trust set -- trusting a proxy that is not
// one lets a direct client forge its address, and the wrong prefix
// would collapse every client into one bucket.
func loadTrustedProxies() ([]netip.Prefix, error) {
	raw := strings.TrimSpace(os.Getenv(EnvTrustedProxies))
	if raw == "" {
		return nil, nil
	}
	entries := strings.Split(raw, ",")
	prefixes := make([]netip.Prefix, 0, len(entries))
	for i, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			return nil, fmt.Errorf("%s: entry %d is empty", EnvTrustedProxies, i+1)
		}
		if prefix, err := netip.ParsePrefix(entry); err == nil {
			prefixes = append(prefixes, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, fmt.Errorf("%s: parse %q as a CIDR prefix or address: %w", EnvTrustedProxies, entry, err)
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return prefixes, nil
}
