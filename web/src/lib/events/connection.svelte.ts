// The SSE connection state of the shared events stream, as the
// offline strip consumes it (the offline state: a top-bar yellow
// strip plus the SSE reconnect countdown). The events module
// (./index.ts) owns the stream's lifecycle and reports every
// transition into this store -- the stream is CONSTRUCTED
// ('connecting'), the browser's connection lands ('open'), the
// browser reports a lost connection ('down'), the last subscriber
// leaves and the stream closes on purpose ('closed').
//
// THE RECONNECT CLOCK IS THE BROWSER'S, NOT OURS: the pinned
// architecture leaves reconnection to the EventSource (see the
// events module's header), and the browser never exposes its retry
// schedule to JavaScript -- so the store records WHEN the connection
// dropped (downSince, epoch ms) and the strip counts the elapsed
// reconnecting seconds from it. A true countdown to the next retry
// would claim a schedule no API can read; the elapsed count never
// lies. On recovery (the browser's own retry landed, onopen fired)
// the state returns to 'open' and the strip falls; on the
// INTENTIONAL close (the last subscriber left) it returns to
// 'closed' -- nothing is lost when nothing is live, so the strip
// falls there too.
//
// Svelte 5 rune module state, like the session and theme stores:
// plain modules and load functions read it through the functions
// below, components react to it directly.

/** The shared stream's connection state, from the events module's reports. */
export type StreamStatus = 'closed' | 'connecting' | 'open' | 'down';

let status = $state<StreamStatus>('closed');
let downSince = $state<number | null>(null);

/** The stream's connection state; 'closed' until the first subscriber opens it. */
export function streamStatus(): StreamStatus {
	return status;
}

/**
 * When the connection dropped (epoch ms), while the state is 'down';
 * null otherwise. Each failed retry re-reports 'down' -- the clock
 * keeps the FIRST drop's timestamp, so the elapsed count spans the
 * whole reconnecting window rather than restarting at every attempt.
 */
export function streamDownSince(): number | null {
	return downSince;
}

/** The stream was constructed; the browser is establishing the connection. */
export function markStreamConnecting(): void {
	status = 'connecting';
}

/** The browser's connection landed (onopen) -- the strip falls. */
export function markStreamOpen(): void {
	status = 'open';
	downSince = null;
}

/**
 * The browser lost the connection (onerror): it owns the retry, so
 * the state stays 'down' until its own reconnect fires onopen. A
 * repeated error (another failed retry) never resets the clock.
 */
export function markStreamDown(): void {
	if (status !== 'down') {
		status = 'down';
		downSince = Date.now();
	}
}

/** The stream closed on purpose (the last subscriber left) -- nothing is lost. */
export function markStreamClosed(): void {
	status = 'closed';
	downSince = null;
}
