// The app's transient-confirmation organ: one module-level queue
// every surface pushes into (the face-correction echoes, the batch
// counts), one host rendering it (the root layout's Toasts).
//
// THE DIVISION OF LABOR (the unified-feedback decision): SUCCESS
// confirmations are transient toasts -- the write landed, the
// surfaces converge through their own refetches, and the user needs
// a beat of acknowledgment, not a standing notice. ERRORS stay
// inline near their source (field slots, dialog error lines, page
// failure cards): a failure needs a place to act from, so it never
// rides this organ. The danger tone exists for the echoes that
// should outlast a success's blink; the info tone for neutral
// announcements.
//
// THE SHAPE: at most three live toasts (the oldest drops when a
// fourth arrives -- a burst confirms its latest state, not its
// history); every toast self-dismisses after its tone's duration
// (success and info 3.2s, danger 6s) and a click dismisses early.
// The timers live HERE, not in the renderer: the durations are the
// organ's semantics, and a hand-dismiss must clear the pending
// expiry instead of racing it.
//
// The queue holds ALREADY-TRANSLATED text: the caller resolved its
// i18n key at the push (the organ never translates, so it stays a
// plain store with no locale dependency).

export type ToastTone = 'success' | 'danger' | 'info';

/** One live toast of the queue. */
export type Toast = {
	/** The queue's monotonic id (the dismissal and timer key). */
	id: number;
	/** The already-resolved message (the caller translated it). */
	text: string;
	tone: ToastTone;
	/** The dismissal fade is in flight; the entry leaves the queue after it. */
	leaving: boolean;
};

/** A toast's self-dismiss delay in ms, by tone. */
const TONE_TTL: Record<ToastTone, number> = {
	success: 3200,
	danger: 6000,
	info: 3200
};

/** The dismissal fade's length (the motion-fast beat). */
const LEAVE_MS = 150;

/** The live queue, oldest first; module state so every surface shares one organ. */
const queue = $state<Toast[]>([]);

let nextId = 1;
/** The pending timers (expiries and leave fades), keyed by toast id. */
const timers = new Map<number, ReturnType<typeof setTimeout>>();

/** The live toasts, oldest first; the host renders exactly this. */
export function toasts(): readonly Toast[] {
	return queue;
}

/**
 * Push one transient confirmation. Past the third live toast the
 * OLDEST drops outright (the cap's drop is immediate, never a fade
 * -- the newest three are what the burst means).
 */
export function toast(text: string, tone: ToastTone = 'success'): void {
	while (queue.length >= 3) {
		const oldest = queue[0];
		if (oldest === undefined) {
			break;
		}
		removeToast(oldest.id);
	}
	const entry: Toast = { id: nextId++, text, tone, leaving: false };
	queue.push(entry);
	timers.set(
		entry.id,
		setTimeout(() => dismissToast(entry.id), TONE_TTL[tone])
	);
}

/**
 * Begin one toast's dismissal (the expiry's path and the click's):
 * mark the fade, clear the pending expiry, remove after the fade.
 * Idempotent -- the second call on a leaving toast is a no-op.
 */
export function dismissToast(id: number): void {
	const entry = queue.find((candidate) => candidate.id === id);
	if (entry === undefined || entry.leaving) {
		return;
	}
	const expiry = timers.get(id);
	if (expiry !== undefined) {
		clearTimeout(expiry);
	}
	entry.leaving = true;
	timers.set(
		id,
		setTimeout(() => removeToast(id), LEAVE_MS)
	);
}

/** Drop one toast outright: the cap's drop and the fade's end. */
function removeToast(id: number): void {
	const timer = timers.get(id);
	if (timer !== undefined) {
		clearTimeout(timer);
	}
	timers.delete(id);
	const index = queue.findIndex((candidate) => candidate.id === id);
	if (index !== -1) {
		queue.splice(index, 1);
	}
}

/** Drop every toast with its timers (tests reset the shared organ between cases). */
export function clearToasts(): void {
	for (const timer of timers.values()) {
		clearTimeout(timer);
	}
	timers.clear();
	queue.length = 0;
}
