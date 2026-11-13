<script lang="ts">
	// The count idiom: anywhere the interface says how many photos or
	// faces there are, it renders a glyph and the number -- never the
	// words "12 photos" -- while the full wording survives on the
	// element's accessible name. The wrapper is one labeled image
	// (role="img" + aria-label/title), so a screen reader reads
	// "12 photos" once; the glyph inside is decorative and must never
	// be announced as an image of its own.
	//
	// Two glyphs, never mixed: photos carry the image glyph, faces
	// (the unnamed clusters) the scan-face glyph. The visible digits
	// keep the locale's grouping exactly as the wording's own number
	// formatting renders it; the caller passes the translated
	// wording, so the i18n keys stay the single source of both.
	import { locale } from 'svelte-i18n';
	import ImageIcon from '@lucide/svelte/icons/image';
	import ScanFace from '@lucide/svelte/icons/scan-face';

	type Props = {
		/** The number the glyph sits beside (grouped for the locale). */
		count: number;
		/** The full translated wording, e.g. "12 photos" — the accessible name. */
		label: string;
		/** Photos use the image glyph; faces the scan-face glyph. */
		kind?: 'photos' | 'faces';
		class?: string;
	};

	let { count, label, kind = 'photos', class: className = '' }: Props = $props();

	const digits = $derived(new Intl.NumberFormat($locale ?? 'en').format(count));
</script>

<span
	class="count inline-flex items-center gap-1 whitespace-nowrap tabular-nums {className}"
	role="img"
	aria-label={label}
	title={label}
>
	{#if kind === 'faces'}
		<ScanFace size={14} class="shrink-0" aria-hidden="true" />
	{:else}
		<ImageIcon size={14} class="shrink-0" aria-hidden="true" />
	{/if}
	{digits}
</span>
