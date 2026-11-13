import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test } from 'vitest';
import { initI18n } from '$lib/i18n';
import type { FaceDetail } from '$lib/api/people';
import PersonAvatarDialog from './PersonAvatarDialog.svelte';

// The avatar picker's seeding contract: the page resolves the face
// the profile row actually RENDERS (the stored pick when the caller
// can see it, the best visible member otherwise) and passes it as
// the current face -- the dialog itself never reaches for the raw
// stored value, so an invisible stored pick can neither lose the
// badge nor arm a write the server would refuse.

function face(id: string): FaceDetail {
	return {
		id,
		assetId: `a-${id}`,
		score: 0.9,
		thumbnail: `/api/v1/faces/${id}/thumbnail`,
		asset: { takenAt: '2026-11-01T10:00:00Z', originalFilename: `photo-${id}.jpg` }
	} as unknown as FaceDetail;
}

const faces = [face('f-1'), face('f-2'), face('f-3')];

const submits: string[] = [];

const mounted: Array<ReturnType<typeof mount>> = [];

function mountDialog(currentFaceId: string | null): void {
	mounted.push(
		mount(PersonAvatarDialog, {
			target: document.body,
			props: {
				open: true,
				faces,
				currentFaceId,
				busy: false,
				errorKey: '',
				onsubmit: (faceId: string) => submits.push(faceId)
			}
		})
	);
	flushSync();
}

function pressedFaceId(): string | null {
	const pressed = document.querySelector<HTMLButtonElement>(
		'.person-avatar-dialog__grid button[aria-pressed="true"]'
	);
	if (pressed === null) {
		return null;
	}
	const src = pressed.querySelector('img')?.getAttribute('src') ?? '';
	return src.match(/faces\/([^/]+)\//)?.[1] ?? null;
}

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

afterEach(() => {
	while (mounted.length > 0) {
		unmount(mounted.pop()!);
		flushSync();
	}
});

function submitButton(): HTMLButtonElement | undefined {
	return Array.from(document.body.querySelectorAll<HTMLButtonElement>('button')).find((button) =>
		button.textContent?.includes('Set avatar')
	);
}

test('the current face opens pressed and the submit is armed', () => {
	mountDialog('f-2');
	expect(pressedFaceId()).toBe('f-2');
	expect(submitButton()?.disabled).toBe(false);
});

test('a stored pick outside the visible set presses nothing and disarms the submit', () => {
	// The page passes the STORED pick verbatim; a pick the caller
	// cannot see renders no row (no badge) and the visibility gate
	// keeps the submit disarmed -- the write the server would refuse
	// is never offered.
	mountDialog('f-invisible');
	expect(pressedFaceId()).toBeNull();
	expect(submitButton()?.disabled).toBe(true);
});
