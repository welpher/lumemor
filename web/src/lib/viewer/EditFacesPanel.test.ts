import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test } from 'vitest';
import { initI18n } from '$lib/i18n';
import type { AssetDetail, AssetFaceSummary } from '$lib/api/assets';
import EditFacesPanel from './EditFacesPanel.svelte';

// The edit-people panel's contract: one row per face of the photo
// (the named, the unnamed and the not-yet-grouped alike, in the
// detail's own order), the trash affordance on NAMED rows alone (an
// unnamed or clusterless face is nobody's member -- the write would
// answer the honest 404, so the affordance never lies), the staged
// label with its pending dot while an edit waits for the commit, the
// apply button's gate (no stages, or a commit in flight, disables
// it), and the callbacks carrying the row's own face.

function face(id: string, overrides: Record<string, unknown> = {}): AssetFaceSummary {
	return {
		id,
		score: 0.9,
		thumbnail: `/api/v1/faces/${id}/thumbnail`,
		clusterId: `c-${id}`,
		label: null,
		...overrides
	} as AssetFaceSummary;
}

const faces = [face('f1', { label: { name: 'Grace' } }), face('f2'), face('f3', { clusterId: null })];

const detail = {
	id: 'a1',
	ownerId: 'u1',
	kind: 'image',
	originalFilename: 'IMG_1.jpg',
	fileSize: 1024,
	mimeType: 'image/jpeg',
	checksum: 'abc',
	createdAt: '2026-11-01T09:00:00Z',
	takenAt: '2026-11-02T10:00:00Z',
	takenAtSource: 'import',
	description: '',
	isTrashed: false,
	isHidden: false,
	isReady: true,
	width: 100,
	height: 100,
	durationMs: null,
	blurhash: null,
	livePhotoVideoId: null,
	exif: {},
	faces,
	tags: [],
	albums: [],
	place: null
} as unknown as AssetDetail;

const reassignes: AssetFaceSummary[] = [];
const detaches: AssetFaceSummary[] = [];
const applies: number[] = [];
const closes: number[] = [];

const mounted: Array<ReturnType<typeof mount>> = [];

function mountPanel(overrides: Record<string, unknown> = {}): ReturnType<typeof mount> {
	const panel = mount(EditFacesPanel, {
		target: document.body,
		props: {
			detail,
			stageLabelOf: () => undefined,
			stagedCount: 0,
			busy: false,
			errorKey: '',
			onclose: () => closes.push(closes.length),
			onreassign: (face: AssetFaceSummary) => reassignes.push(face),
			ondetach: (face: AssetFaceSummary) => detaches.push(face),
			onapply: () => applies.push(applies.length),
			...overrides
		}
	});
	mounted.push(panel);
	flushSync();
	return panel;
}

function rows(): HTMLLIElement[] {
	return Array.from(document.body.querySelectorAll<HTMLLIElement>('.faces-panel__row'));
}

function rowByName(text: string): HTMLLIElement {
	const found = rows().find((row) => row.textContent?.includes(text));
	if (found === undefined) {
		throw new Error(`missing row ${text}`);
	}
	return found;
}

function rowButton(row: HTMLLIElement, label: string): HTMLButtonElement {
	const found = Array.from(row.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.getAttribute('aria-label') === label
	);
	if (found === undefined) {
		throw new Error(`missing button ${label}`);
	}
	return found;
}

function clickButton(button: HTMLButtonElement): void {
	button.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

function clickByLabel(label: string): void {
	const found = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.getAttribute('aria-label') === label
	);
	if (found === undefined) {
		throw new Error(`missing button ${label}`);
	}
	clickButton(found);
}

beforeEach(async () => {
	document.body.textContent = '';
	reassignes.length = 0;
	detaches.length = 0;
	applies.length = 0;
	closes.length = 0;
	await initI18n();
});

afterEach(() => {
	while (mounted.length > 0) {
		unmount(mounted.pop()!);
		flushSync();
	}
});

test("one row per face, the three label states in the detail's order", () => {
	mountPanel();

	const rendered = rows();
	expect(rendered).toHaveLength(3);
	expect(rendered.map((row) => row.dataset.faceId)).toEqual(['f1', 'f2', 'f3']);
	expect(rowByName('Grace').textContent?.trim()).toBe('Grace');
	// The unnamed and not-grouped states use the chips' vocabulary.
	expect(rowByName('Unnamed').textContent?.trim()).toBe('Unnamed');
	expect(rowByName('Not grouped yet').textContent?.trim()).toBe('Not grouped yet');
});

test('the trash sits on NAMED rows alone; every row keeps the reassign pencil', () => {
	mountPanel();

	// The named row: both affordances, icon-only with their labels.
	const named = rowByName('Grace');
	expect(named.querySelectorAll('button')).toHaveLength(2);
	expect(named.querySelector('[aria-label="Remove this face"]')).not.toBeNull();
	expect(named.querySelector('[aria-label="Reassign this face"]')).not.toBeNull();

	// The unnamed and the not-yet-grouped rows: reassign alone (no
	// person to detach from -- the affordance is honest).
	for (const text of ['Unnamed', 'Not grouped yet']) {
		const row = rowByName(text);
		expect(row.querySelectorAll('button')).toHaveLength(1);
		expect(row.querySelector('[aria-label="Reassign this face"]')).not.toBeNull();
		expect(row.querySelector('[aria-label="Remove this face"]')).toBeNull();
	}
});

test("the staged label replaces the row's name and carries the pending dot", () => {
	mountPanel({
		stageLabelOf: (face: AssetFaceSummary) => (face.id === 'f2' ? 'Mom' : undefined),
		stagedCount: 1
	});

	const staged = rowByName('Mom');
	expect(staged.dataset.staged).toBe('true');
	expect(staged.querySelector('.faces-panel__pending')).not.toBeNull();
	// The unstaged rows carry no dot and no marker.
	const plain = rowByName('Grace');
	expect(plain.dataset.staged).toBe('false');
	expect(plain.querySelector('.faces-panel__pending')).toBeNull();
});

test('the apply button gates on the staged count and the busy flag', () => {
	mountPanel();
	const apply = () => document.body.querySelector<HTMLButtonElement>('.faces-panel__apply')!;
	expect(apply().disabled).toBe(true);

	while (mounted.length > 0) {
		unmount(mounted.pop()!);
	}
	flushSync();
	mountPanel({ stagedCount: 2 });
	expect(apply().disabled).toBe(false);

	while (mounted.length > 0) {
		unmount(mounted.pop()!);
	}
	flushSync();
	mountPanel({ stagedCount: 2, busy: true });
	expect(apply().disabled).toBe(true);
});

test("the callbacks carry the row's own face; the header returns to the information panel", () => {
	mountPanel();

	clickButton(rowButton(rowByName('Grace'), 'Reassign this face'));
	expect(reassignes.map((face) => face.id)).toEqual(['f1']);

	clickButton(rowButton(rowByName('Grace'), 'Remove this face'));
	expect(detaches.map((face) => face.id)).toEqual(['f1']);

	clickByLabel('Finish editing');
	expect(applies).toHaveLength(1);

	clickByLabel('Photo information');
	expect(closes).toHaveLength(1);
});

test("the commit's failure renders the honest error line inside the panel", () => {
	mountPanel({ errorKey: 'person_not_found' });

	const error = document.body.querySelector('.faces-panel__error');
	expect(error?.getAttribute('role')).toBe('alert');
	expect(error?.textContent?.trim()).toBe('This person does not exist or is not visible to you.');
});
