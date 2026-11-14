// The release check: the instance's one scheduled outbound probe. It
// reads the newest tag of a configured release feed, compares it with
// the running version and keeps the answer in the instance settings,
// where the admin face reads it.
//
// THE DISCIPLINE (what the probe is and is never):
//
//   - The request is one plain GET for a public JSON document. No
//     cookie, no instance identifier, no photo, no EXIF and no
//     embedding rides along; the feed sees the instance's address.
//   - The target is refused when it resolves to an internal range
//     and redirects are never followed (see client.go).
//   - Failure is silence: a probe that cannot reach its feed, that
//     answers a non-200, that names nothing readable or that names a
//     version the running build cannot compare leaves the status at
//     `unknown`. Nothing red, nothing loud -- an offline instance is
//     the ordinary case of a self-hosted library, not a fault.
//   - The answer is cached until the next interval. The scheduled
//     loop probes at most once per interval; the manual run probes
//     on the administrator's request. Nothing is ever downloaded or
//     installed: the surface only reports that a newer tag exists.
//
// The state -- when the last probe completed and the tag it named --
// is persisted beside the settings, so a restart keeps both the
// answer and the honest "last checked" line.
package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// Status is the verdict the face renders. The values are the
// contract's enum values verbatim.
type Status string

// The four states of the check.
const (
	// StatusDisabled means no check runs: the settings switch is off
	// or the deployment disabled the feature.
	StatusDisabled Status = "disabled"
	// StatusUnknown is the honest silence of a check that could not
	// conclude.
	StatusUnknown Status = "unknown"
	// StatusUpToDate means the running version is the feed's newest.
	StatusUpToDate Status = "up_to_date"
	// StatusUpdateAvailable means the feed names a newer release.
	StatusUpdateAvailable Status = "update_available"
)

// Snapshot is the check's state as the admin face reads it.
type Snapshot struct {
	// Enabled reports whether a check runs at all.
	Enabled bool
	// Status is the verdict.
	Status Status
	// CurrentVersion is the running build's version, verbatim.
	CurrentVersion string
	// LatestVersion is the newest tag the feed named, empty when no
	// probe has succeeded.
	LatestVersion string
	// CheckedAt is when the last probe completed, nil before the
	// first one.
	CheckedAt *time.Time
}

// probeBodyLimit bounds the release document the probe reads: a
// release feed's newest-release document is a few kilobytes, and the
// bound keeps a hostile or broken target from streaming at the
// instance.
const probeBodyLimit = 64 << 10

// Options carries the checker's dependencies.
type Options struct {
	// Queries read and write the settings rows.
	Queries *dbgen.Queries
	// Logger receives the check's one line per attempt; nil falls
	// back to slog.Default().
	Logger *slog.Logger
	// Client carries the outbound discipline; nil builds the
	// production client (proxy environment, internal targets
	// refused, no redirects, the bounded timeout).
	Client *http.Client
	// Version is the running build's version string (the build
	// info's value verbatim).
	Version string
	// Disabled is the deployment's own off switch: a stack that
	// must never reach the outside world (a test harness) turns the
	// feature off without touching the instance's settings.
	Disabled bool
	// Now drives the freshness decision and the stamps; nil falls
	// back to time.Now.
	Now func() time.Time
}

// Checker runs the scheduled check and answers its state.
type Checker struct {
	queries  *dbgen.Queries
	logger   *slog.Logger
	client   *http.Client
	version  string
	disabled bool
	now      func() time.Time
}

// New builds a checker under the options.
func New(options Options) *Checker {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	client := options.Client
	if client == nil {
		client = NewClient(ClientPolicy{RefuseInternal: true})
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Checker{
		queries:  options.Queries,
		logger:   logger,
		client:   client,
		version:  options.Version,
		disabled: options.Disabled,
		now:      now,
	}
}

// Snapshot answers the current state without any network access.
func (c *Checker) Snapshot(ctx context.Context) Snapshot {
	settings, err := db.UpdateCheckSettingsOf(ctx, c.queries)
	if err != nil {
		// The accessor already answered the documented defaults; the
		// line points at the row to fix.
		c.logger.Warn("read the release check settings; answering the defaults", "error", err)
	}
	state, err := db.UpdateCheckStateOf(ctx, c.queries)
	if err != nil {
		c.logger.Warn("read the release check state; answering never-checked", "error", err)
	}
	snapshot := Snapshot{
		Enabled:        settings.Enabled && !c.disabled,
		CurrentVersion: c.version,
		LatestVersion:  state.LatestTag,
		CheckedAt:      state.CheckedAt,
	}
	snapshot.Status = c.verdict(snapshot.Enabled, state)
	return snapshot
}

// verdict derives the status from the switch and the last answer.
func (c *Checker) verdict(enabled bool, state db.UpdateCheckState) Status {
	if !enabled {
		return StatusDisabled
	}
	if state.CheckedAt == nil || state.LatestTag == "" {
		// Never checked, or the last probe named nothing usable.
		return StatusUnknown
	}
	current, okCurrent := ParseVersion(c.version)
	latest, okLatest := ParseVersion(state.LatestTag)
	if !okCurrent || !okLatest {
		// A development build has no comparable version; the face
		// must not invent an order.
		return StatusUnknown
	}
	if latest.Compare(current) > 0 {
		return StatusUpdateAvailable
	}
	return StatusUpToDate
}

// CheckNow runs one probe on the caller's request (the manual check)
// and answers the fresh state. A disabled check probes nothing and
// answers `disabled`; a failed probe answers `unknown` -- 200 in both
// cases, because neither is a request error.
func (c *Checker) CheckNow(ctx context.Context) Snapshot {
	settings, err := db.UpdateCheckSettingsOf(ctx, c.queries)
	if err != nil {
		c.logger.Warn("read the release check settings; answering the defaults", "error", err)
	}
	if c.disabled || !settings.Enabled {
		return c.Snapshot(ctx)
	}
	c.probe(ctx, settings.Endpoint)
	return c.Snapshot(ctx)
}

// probe fetches the feed once and persists the answer: the tag on
// success, an empty tag on failure -- the same row either way, so the
// "last checked" line is always the last attempt's.
func (c *Checker) probe(ctx context.Context, endpoint string) {
	checkedAt := c.now().UTC()
	tag, err := c.fetchLatestTag(ctx, endpoint)
	state := db.UpdateCheckState{CheckedAt: &checkedAt}
	if err != nil {
		// Silence is the product behavior; this line is the
		// operator's own record of why the face stays quiet.
		c.logger.Info("release check did not conclude; the state stays quiet", "error", err)
	} else {
		state.LatestTag = tag
		c.logger.Info("release check completed", "latest_tag", tag)
	}
	if err := db.SaveUpdateCheckState(ctx, c.queries, state); err != nil {
		// Losing the row costs the next restart the answer, never
		// the running instance's.
		c.logger.Warn("save the release check state", "error", err)
	}
}

// fetchLatestTag performs the bounded request and reads the tag. The
// document is a JSON object; the tag rides `tag`, or `tag_name` for
// release feeds that spell it that way.
func (c *Checker) fetchLatestTag(ctx context.Context, endpoint string) (string, error) {
	parsed, err := validEndpoint(endpoint)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("updatecheck: build the probe request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "lumemor/"+c.version)
	response, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("updatecheck: reach the release feed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("updatecheck: the release feed answered %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, probeBodyLimit))
	if err != nil {
		return "", fmt.Errorf("updatecheck: read the release feed: %w", err)
	}
	var document struct {
		Tag     string `json:"tag"`
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		return "", fmt.Errorf("updatecheck: parse the release feed: %w", err)
	}
	tag := strings.TrimSpace(document.Tag)
	if tag == "" {
		tag = strings.TrimSpace(document.TagName)
	}
	if tag == "" {
		return "", fmt.Errorf("updatecheck: the release feed names no tag")
	}
	return tag, nil
}

// Run keeps the scheduled check alive until ctx ends: one pass at
// start (a fresh or stale instance checks immediately) and then a
// pass every minute, each probing only when the interval has elapsed
// since the last attempt. A failing probe waits the interval like a
// successful one -- a feed that is not there is not called more often
// for it.
func (c *Checker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		c.checkIfDue(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// checkIfDue runs one scheduled pass: due when no probe has completed
// yet, or when the last one is older than the configured interval.
func (c *Checker) checkIfDue(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	settings, err := db.UpdateCheckSettingsOf(ctx, c.queries)
	if err != nil {
		c.logger.Warn("read the release check settings; skipping this pass", "error", err)
	}
	if c.disabled || !settings.Enabled {
		return
	}
	state, err := db.UpdateCheckStateOf(ctx, c.queries)
	if err != nil {
		c.logger.Warn("read the release check state; checking now", "error", err)
	}
	if state.CheckedAt != nil {
		interval := time.Duration(settings.IntervalHours) * time.Hour
		if c.now().Before(state.CheckedAt.Add(interval)) {
			return
		}
	}
	c.probe(ctx, settings.Endpoint)
}
