import { expect, test } from 'vitest';
import { ambiguousLabels, displayNameOf, initialOf } from './identity';

// The identity presentation rule: the display name
// leads, an empty one falls back to the email, and two accounts under
// one name get the email as their disambiguating title.

test('the display label is the name, and the email when the name is empty', () => {
	expect(displayNameOf({ name: 'Mum', email: 'mum@example.com' })).toBe('Mum');
	expect(displayNameOf({ name: '', email: 'mum@example.com' })).toBe('mum@example.com');
	expect(displayNameOf({ email: 'mum@example.com' })).toBe('mum@example.com');
	// A whitespace-only name is no name: a blank attribution never renders.
	expect(displayNameOf({ name: '   ', email: 'mum@example.com' })).toBe('mum@example.com');
});

test('the avatar initial comes from the display label', () => {
	expect(initialOf({ name: 'Grace Hopper', email: 'grace@example.com' })).toBe('G');
	// A multi-byte leading character is one rune, not one byte.
	expect(initialOf({ name: '🌷 Garden', email: 'flower@example.com' })).toBe('🌷');
	// The email fallback's local part is the initial when no name is set.
	expect(initialOf({ name: '', email: 'grace@example.com' })).toBe('G');
});

test('a name two different accounts carry is ambiguous; a repeated account is not', () => {
	const ambiguous = ambiguousLabels([
		{ id: 'u1', name: 'Mum', email: 'mum@example.com' },
		{ id: 'u2', name: 'Mum', email: 'mum2@example.com' },
		{ id: 'u3', name: 'Dad', email: 'dad@example.com' },
		// The same account twice (two cards of one owner): no ambiguity.
		{ id: 'u3', name: 'Dad', email: 'dad@example.com' },
		// An email-labelled identity is unique by definition.
		{ id: 'u4', name: '', email: 'solo@example.com' }
	]);
	expect([...ambiguous]).toEqual(['Mum']);
});
