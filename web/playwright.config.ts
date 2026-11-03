import { defineConfig, devices } from '@playwright/test';

// The end-to-end suite of the web app: one browser driving the real
// compose stack (docker-compose.yml, the server at
// http://localhost:8244). The stack's lifecycle is owned by the
// caller -- scripts/acceptance_m2d.sh builds it, waits for /healthz
// and tears it down -- so this config only pins how the suite talks
// to it. See web/e2e/ for the M2 full-chain story the suite walks.
export default defineConfig({
	testDir: 'e2e',
	// One stack, one story: the scenes build on one another (the
	// session, the uploads, the wall's navigation snapshot), so the
	// specs run serially in a single worker and share one browser
	// context (see the story fixture in the spec).
	fullyParallel: false,
	workers: 1,
	// The story's slow steps are pipeline waits (an upload's SSE
	// asset.ready, the Live Photo component's transcode), not page
	// loads -- generous test bounds, default expect bounds.
	timeout: 180_000,
	expect: {
		timeout: 15_000
	},
	// No retries locally: a flaky scene must fail loudly here. CI
	// allows one retry for the runner-level jitter only.
	retries: process.env.CI ? 1 : 0,
	reporter: [['list']],
	use: {
		baseURL: 'http://localhost:8244',
		// The visible strings the scenes assert (month labels, plural
		// forms) are locale-rendered; pin the locale and zone so the
		// i18n'd text is deterministic. A fresh context renders en.
		locale: 'en-US',
		timezoneId: 'UTC',
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure',
		video: 'retain-on-failure'
	},
	outputDir: 'test-results',
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'] }
		}
	]
});
