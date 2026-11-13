<script lang="ts">
	// The content-state card: the ONE shape a page's empty and failed
	// faces render. A page's states replace the page BODY (the bars,
	// the context bars and the profile rows stay); the card centers in
	// the content area at 420px, and it carries at most ONE action --
	// the no-data card may offer the way in (upload, create), the
	// no-results card offers none, and the failed card offers the
	// retry.
	//
	// TWO TONES, ONE SHAPE:
	// - `empty`: the quiet card -- a 16px display title (never an h1:
	//   the page's own title lives in its bar), a 14px line, the
	//   optional action.
	// - `error`: the same box in the danger border with the failure
	//   text and its retry. A load failure is a STATE, never a toast
	//   (the toast belongs to action feedback).
	import Button from './Button.svelte';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';

	type Props = {
		/** The card's tone (see the header). */
		tone?: 'empty' | 'error';
		/** The empty card's title line. */
		title?: string;
		/** The explanatory line (both tones). */
		body: string;
		/** The one action's label; absent = no action. */
		actionLabel?: string;
		/** The action's handler. */
		onaction?: () => void;
	};

	let { tone = 'empty', title = '', body, actionLabel = '', onaction }: Props = $props();
</script>

{#if tone === 'error'}
	<div
		class="state-card state-card--error flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
		role="alert"
	>
		<p class="state-card__body m-0 font-body font-normal text-[14px] leading-[1.5]">{body}</p>
		{#if actionLabel !== '' && onaction !== undefined}
			<Button variant="primary" onclick={onaction}>
				<RotateCcw size={20} aria-hidden="true" />
				{actionLabel}
			</Button>
		{/if}
	</div>
{:else}
	<div
		class="state-card max-w-[420px] box-content mx-auto my-12 p-6 text-center bg-card border border-border rounded-card shadow-card"
	>
		{#if title !== ''}
			<p class="state-card__title m-0 font-display font-bold text-[16px] leading-[1.3] text-text">{title}</p>
		{/if}
		<p
			class="state-card__body m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft"
			class:mt-2={title !== ''}
		>
			{body}
		</p>
		{#if actionLabel !== '' && onaction !== undefined}
			<div class="state-card__action mt-4">
				<Button variant="primary" onclick={onaction}>{actionLabel}</Button>
			</div>
		{/if}
	</div>
{/if}
