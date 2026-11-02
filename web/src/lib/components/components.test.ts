import { createRawSnippet, flushSync, mount, tick, unmount } from 'svelte';
import { beforeEach, expect, test } from 'vitest';
import Button from './Button.svelte';
import Card from './Card.svelte';
import Input from './Input.svelte';
import Modal from './Modal.svelte';
import { initI18n } from '../i18n';

// createRawSnippet is the supported way to hand a snippet to mount()
// from plain TS test code. Its render output must be a single
// element, so the plain-text helper wraps in a span.
const text = (value: string) =>
	createRawSnippet(() => ({ render: () => `<span>${value}</span>` }));

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

test('Button renders its label with each variant class', () => {
	const cases = [
		{ variant: 'primary', className: 'btn--primary' },
		{ variant: 'ghost', className: 'btn--ghost' },
		{ variant: 'danger', className: 'btn--danger' }
	] as const;
	for (const { variant, className } of cases) {
		const button = mount(Button, {
			target: document.body,
			props: { variant, children: text('Press') }
		});
		flushSync();
		const element = document.body.querySelector<HTMLButtonElement>('button.btn');
		expect(element, `variant ${variant}`).not.toBeNull();
		expect(element?.className).toContain(className);
		expect(element?.textContent).toBe('Press');
		expect(element?.type).toBe('button');
		unmount(button);
		flushSync();
	}
});

test('Input ties its label to the field and passes attributes through', () => {
	const input = mount(Input, {
		target: document.body,
		props: { id: 'email-field', label: 'Email', type: 'email', value: 'first@example.test' }
	});
	flushSync();
	const label = document.body.querySelector<HTMLLabelElement>('label');
	const field = document.body.querySelector<HTMLInputElement>('input');
	expect(label?.htmlFor).toBe('email-field');
	expect(field?.id).toBe('email-field');
	expect(field?.type).toBe('email');
	expect(field?.value).toBe('first@example.test');
	unmount(input);
	flushSync();
});

test('Input wires its error message to the field for assistive tech', () => {
	const input = mount(Input, {
		target: document.body,
		props: { id: 'pw-field', label: 'Password', error: 'Wrong password' }
	});
	flushSync();
	const field = document.body.querySelector<HTMLInputElement>('input');
	const message = document.body.querySelector<HTMLParagraphElement>('.field__error');
	expect(field?.getAttribute('aria-invalid')).toBe('true');
	expect(field?.getAttribute('aria-describedby')).toBe('pw-field-error');
	expect(message?.id).toBe('pw-field-error');
	expect(message?.textContent).toBe('Wrong password');
	unmount(input);
	flushSync();
});

test('Card renders its content on the card surface', () => {
	const card = mount(Card, {
		target: document.body,
		props: { children: text('Card body') }
	});
	flushSync();
	const element = document.body.querySelector<HTMLDivElement>('div.card');
	expect(element).not.toBeNull();
	expect(element?.className).toContain('card--padded');
	expect(element?.textContent).toBe('Card body');
	unmount(card);
	flushSync();
});

test('Modal opens as a dialog, and Escape and the backdrop close it', async () => {
	const modal = mount(Modal, {
		target: document.body,
		props: {
			open: true,
			title: 'Confirm',
			children: text('Delete this photo?'),
			footer: text('Cancel')
		}
	});
	flushSync();
	const dialog = document.body.querySelector<HTMLDivElement>('[role="dialog"]');
	expect(dialog).not.toBeNull();
	expect(dialog?.getAttribute('aria-modal')).toBe('true');
	expect(dialog?.getAttribute('aria-label')).toBe('Confirm');
	expect(dialog?.textContent).toContain('Delete this photo?');
	// The close affordances carry translated labels.
	const closeButton = dialog?.querySelector<HTMLButtonElement>('.modal__close');
	expect(closeButton?.getAttribute('aria-label')).toBe('Close');

	// Escape anywhere closes the dialog.
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
	await tick();
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();

	// The backdrop click closes it too; clicks inside the panel must
	// not (mousedown on the panel is stopped).
	const modal2 = mount(Modal, {
		target: document.body,
		props: { open: true, title: 'Confirm', children: text('Again?') }
	});
	flushSync();
	const backdrop = document.body.querySelector<HTMLDivElement>('.modal__backdrop');
	backdrop?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
	await tick();
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	unmount(modal2);
	flushSync();
	unmount(modal);
	flushSync();
});
