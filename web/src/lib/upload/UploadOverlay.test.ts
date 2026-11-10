import { flushSync, mount, tick, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import UploadOverlay from './UploadOverlay.svelte';
import { initI18n, translate } from '$lib/i18n';

// The overlay is a pure view over the queue singleton: the mocked
// module keeps the queue's world static for each mounted instance
// (the snapshot is set before mount), lets the tests assert the
// routing of every affordance (dismiss, retry, decide, the dialog's
// two resolution buttons) and the drop-zone edge counting. Every
// mount unmounts in a finally so the window-level listeners of an
// aborted case cannot leak into its successors.

const box = vi.hoisted(() => ({
	items: [] as Array<Record<string, unknown>>,
	conflictId: null as string | null,
	conflictErrorKey: '',
	enqueues: [] as Array<unknown>,
	actions: {
		closeConflict: vi.fn(),
		openConflict: vi.fn(),
		restore: vi.fn(),
		saveAs: vi.fn(),
		retry: vi.fn(),
		dismiss: vi.fn()
	}
}));

vi.mock('$lib/upload/queue.svelte', () => ({
	uploadQueueItems: () => box.items,
	uploadQueueConflictId: () => box.conflictId,
	uploadQueueConflictErrorKey: () => box.conflictErrorKey,
	enqueueUploadFiles: (files: Array<File>) => {
		box.enqueues.push(files);
	},
	closeUploadConflict: box.actions.closeConflict,
	openUploadConflict: box.actions.openConflict,
	resolveUploadByRestore: box.actions.restore,
	resolveUploadBySaveAs: box.actions.saveAs,
	retryUpload: box.actions.retry,
	dismissUpload: box.actions.dismiss,
	uploadExtensionOf: (fileName: string) => {
		const dot = fileName.lastIndexOf('.');
		return dot <= 0 || dot === fileName.length - 1 ? '' : fileName.slice(dot + 1).toLowerCase();
	},
	uploadFormatSupported: (fileName: string) => {
		const dot = fileName.lastIndexOf('.');
		if (dot <= 0 || dot === fileName.length - 1) {
			return false;
		}
		return [
			'jpg',
			'jpeg',
			'png',
			'gif',
			'webp',
			'tif',
			'tiff',
			'heic',
			'avif',
			'mp4',
			'mov',
			'webm',
			'mkv'
		].includes(fileName.slice(dot + 1).toLowerCase());
	}
}));

function item(overrides: Record<string, unknown>): Record<string, unknown> {
	return {
		id: 'row-1',
		fileName: 'photo.jpg',
		size: 10,
		state: 'uploading',
		progress: 0.5,
		errorKey: '',
		asset: { id: 'asset-1', state: 'created' },
		...overrides
	};
}

/** A draggable event with the payload the handlers read (jsdom
 * carries no DragEvent constructor). */
function drag(type: string, payload: { types?: string[]; files?: File[] } = {}): DragEvent {
	const event = new MouseEvent(type, {
		bubbles: true,
		cancelable: true
	}) as unknown as DragEvent;
	Object.defineProperty(event, 'dataTransfer', {
		value: { types: payload.types ?? [], files: payload.files ?? [] }
	});
	return event;
}

/** Mount, run, and always unmount (window listeners leak otherwise). */
async function withOverlay(run: () => void | Promise<void>): Promise<void> {
	const instance = mount(UploadOverlay, { target: document.body });
	try {
		await run();
	} finally {
		unmount(instance);
		flushSync();
		document.body.textContent = '';
	}
}

const footerButtons = (): HTMLButtonElement[] => [
	...document.body.querySelectorAll<HTMLButtonElement>('footer button')
];

/** A click followed by a full drain: the resolution handlers flip
 * their resolving flag back on microtasks; flushSync forces the
 * render that carries it into the DOM. */
async function commit(button: HTMLButtonElement): Promise<void> {
	button.click();
	await tick();
	flushSync();
}

beforeEach(async () => {
	document.body.textContent = '';
	box.items = [];
	box.conflictId = null;
	box.conflictErrorKey = '';
	box.enqueues = [];
	for (const mock of Object.values(box.actions)) {
		mock.mockClear();
	}
	await initI18n();
});

test('an empty queue renders nothing', async () => {
	await withOverlay(() => {
		flushSync();
		expect(document.body.querySelector('.overlay')).toBeNull();
	});
});

test('rows render their state label, bar, failure text and affordances', async () => {
	box.items = [
		item({ id: 'r-up', progress: 0.4 }),
		item({ id: 'r-processing', state: 'processing', progress: 1 }),
		item({ id: 'r-done', state: 'done', progress: 1 }),
		item({ id: 'r-fail', state: 'failed', errorKey: 'asset_too_large', asset: null }),
		item({ id: 'r-conflict', state: 'conflict' })
	];
	await withOverlay(() => {
		flushSync();

		const rows = [...document.body.querySelectorAll<HTMLLIElement>('ul.overlay__rows > li')];
		expect(rows).toHaveLength(5);
		expect(rows[0]?.textContent).toContain(translate('upload_state_uploading'));
		expect(rows[1]?.textContent).toContain(translate('upload_state_processing'));
		expect(rows[2]?.textContent).toContain(translate('upload_state_done'));
		expect(rows[3]?.textContent).toContain(translate('upload_state_failed'));
		// The failure surfaces the mapped error message and its retry.
		expect(rows[3]?.textContent).toContain(translate('asset_too_large'));
		expect(rows[3]?.textContent).toContain(translate('action_retry'));
		// The parked row carries the decide affordance.
		expect(rows[4]?.textContent).toContain(translate('upload_resolve_conflict'));
		// The in-flight row carries a determinate bar at 40%.
		const fill = rows[0]?.querySelector('.row__bar-fill') as HTMLElement;
		expect(fill?.style.width).toBe('40%');
		// The determinate bar names its own state and value; the
		// indeterminate one names the processing state and carries
		// no value (an unknown-duration wait has no percentage).
		const bar = rows[0]?.querySelector('.row__bar') as HTMLElement;
		expect(bar?.getAttribute('aria-label')).toBe(translate('upload_state_uploading'));
		expect(bar?.getAttribute('aria-valuenow')).toBe('40');
		const indeterminate = rows[1]?.querySelector('.row__bar') as HTMLElement;
		expect(indeterminate?.className).toContain('row__bar--indeterminate');
		expect(indeterminate?.getAttribute('aria-label')).toBe(translate('upload_state_processing'));
		expect(indeterminate?.hasAttribute('aria-valuenow')).toBe(false);
		// Every row dismisses by its close affordance.
		const closes = [...document.body.querySelectorAll<HTMLButtonElement>('button.row__close')];
		expect(closes).toHaveLength(5);
		expect(closes[4]?.getAttribute('aria-label')).toBe(translate('upload_dismiss'));
		closes[4]?.click();
		expect(box.actions.dismiss).toHaveBeenCalledWith('r-conflict');
	});
});

test('retry routes a supported failed row; a hint-rejected row gets no retry affordance', async () => {
	box.items = [
		item({ id: 'r-fail', state: 'failed', errorKey: 'asset_too_large', asset: null }),
		item({
			id: 'r-bad',
			fileName: 'notes.txt',
			state: 'failed',
			errorKey: 'validation_failed',
			asset: null
		})
	];
	await withOverlay(() => {
		flushSync();
		const rows = [...document.body.querySelectorAll<HTMLLIElement>('ul.overlay__rows > li')];
		expect(rows[0]?.textContent).toContain(translate('action_retry'));
		expect(rows[1]?.textContent.indexOf(translate('action_retry'))).toBe(-1);
		const retry = rows[0]?.querySelector<HTMLButtonElement>('button.row__action');
		retry?.click();
		expect(box.actions.retry).toHaveBeenCalledWith('r-fail');
	});
});

test('a decided row reopens the dialog through its affordance', async () => {
	box.items = [item({ id: 'r-conflict', state: 'conflict' })];
	box.conflictId = null; // The dialog was decided or dismissed earlier.
	await withOverlay(() => {
		flushSync();
		const decide = document.body.querySelector<HTMLButtonElement>('ul.overlay__rows > li button.row__action');
		decide?.click();
		expect(box.actions.openConflict).toHaveBeenCalledWith('r-conflict');
	});
});

test('the conflict dialog renders the tracked file with the pinned button order and routes', async () => {
	box.items = [item({ id: 'r-conflict', state: 'conflict' })];
	box.conflictId = 'r-conflict';
	await withOverlay(() => {
		flushSync();

		// The panel is open and names the tracked file.
		const panel = document.body.querySelector<HTMLElement>('[role="dialog"]');
		expect(panel).not.toBeNull();
		expect(panel?.textContent).toContain(translate('upload_conflict_title'));
		expect(panel?.textContent).toContain('photo.jpg');

		// The footer order: cancel, save-as, restore.
		const buttons = footerButtons();
		expect(buttons.map((button) => button.textContent)).toEqual([
			translate('common_cancel'),
			translate('upload_conflict_save_as'),
			translate('upload_conflict_restore')
		]);
	});

	await withOverlay(async () => {
		flushSync();
		await commit(footerButtons()[0] as HTMLButtonElement);
		expect(box.actions.closeConflict).toHaveBeenCalledOnce();
		await commit(footerButtons()[1] as HTMLButtonElement);
		expect(box.actions.saveAs).toHaveBeenCalledWith('r-conflict');
		await commit(footerButtons()[2] as HTMLButtonElement);
		expect(box.actions.restore).toHaveBeenCalledWith('r-conflict');
	});
});

test('resolution in flight disables every footer button', async () => {
	box.items = [item({ id: 'r-conflict', state: 'conflict' })];
	box.conflictId = 'r-conflict';
	let release: (() => void) | null = null;
	box.actions.restore.mockImplementation(
		() =>
			new Promise<void>((resolve) => {
				release = resolve;
			})
	);
	await withOverlay(async () => {
		flushSync();
		const button = footerButtons()[2] as HTMLButtonElement;
		button.click();
		// One tick: the resolving flag reached the DOM.
		await tick();
		flushSync();
		const inFlight = footerButtons();
		expect(inFlight).toHaveLength(3);
		for (const pending of inFlight) {
			expect(pending.disabled, pending.textContent ?? '').toBe(true);
		}
		release?.();
		await tick();
		flushSync();
		for (const pending of footerButtons()) {
			expect(pending.disabled, pending.textContent ?? '').toBe(false);
		}
	});
});

test("a failed conflict decision surfaces the queue's error key", async () => {
	box.items = [item({ id: 'r-conflict', state: 'conflict' })];
	box.conflictId = 'r-conflict';
	box.conflictErrorKey = 'internal';
	await withOverlay(() => {
		flushSync();
		const panel = document.body.querySelector<HTMLElement>('[role="dialog"]');
		expect(panel?.textContent).toContain(translate('internal'));
	});
});

test('the drop zone counts edges, hints, enqueues the dropped files and consumes a Files drag only', async () => {
	const file = new File(['x'], 'photo.jpg', { type: 'image/jpeg' });
	await withOverlay(() => {
		flushSync();
		expect(document.body.querySelector('.drophint')).toBeNull();

		// Entering with a non-Files drag (text selection) never engages.
		window.dispatchEvent(drag('dragenter', { types: ['text/plain'] }));
		expect(document.body.querySelector('.drophint')).toBeNull();

		// Entering with files shows the hint; crossing child-element
		// edges during the drag leaves it count-stable.
		window.dispatchEvent(drag('dragenter', { types: ['Files'] }));
		flushSync();
		expect(document.body.querySelector('.drophint')).not.toBeNull();
		window.dispatchEvent(drag('dragenter', { types: ['Files'] }));
		window.dispatchEvent(drag('dragleave', { types: ['Files'] }));
		flushSync();
		expect(document.body.querySelector('.drophint')).not.toBeNull();

		// Dropping enqueues and closes the zone.
		window.dispatchEvent(drag('drop', { types: ['Files'], files: [file] }));
		expect(box.enqueues).toHaveLength(1);
		expect(box.enqueues[0] as Array<File>, 'one file per drop').toHaveLength(1);

		// The drop cleared the depth; one leave closes the zone.
		window.dispatchEvent(drag('dragleave', { types: ['Files'] }));
		flushSync();
		expect(document.body.querySelector('.drophint')).toBeNull();

		// A Files drop with no files enqueues nothing.
		window.dispatchEvent(drag('dragenter', { types: ['Files'] }));
		window.dispatchEvent(drag('drop', { types: ['Files'], files: [] }));
		expect(box.enqueues).toHaveLength(1);
	});
});
