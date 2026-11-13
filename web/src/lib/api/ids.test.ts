import { expect, test } from 'vitest';
import { isUuidShaped } from './ids';

// The malformed-id edge: a route id that is not UUID-shaped names no
// resource, so the detail clients answer it with the unreachable face
// before any request (a 400's retryable voice would invite the reader
// to retry a typo forever).

test('the contract UUID shape is the only accepted path id', () => {
	expect(isUuidShaped('0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60')).toBe(true);
	expect(isUuidShaped('0192F2A1-3B4C-7D8E-9F0A-1B2C3D4E5F60')).toBe(true);
});

test('anything else is not a resource id', () => {
	for (const id of [
		'',
		'not-a-uuid',
		'0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f6',
		'0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f600',
		'0192f2a13b4c7d8e9f0a1b2c3d4e5f60',
		'0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f6g',
		' 0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60'
	]) {
		expect(isUuidShaped(id), id).toBe(false);
	}
});
