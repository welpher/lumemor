<script lang="ts">
	// The toast organ's host: the root layout mounts it once and it
	// renders the store's live queue -- one button per toast (a click
	// dismisses early), the tone's 16px decorative glyph and the
	// 13px/600 message on the card face, bottom-center above the
	// batch bar. The container is the polite live region and it
	// EXISTS before any toast lands (a live region present at
	// announcement time reads reliably); each success or info toast
	// carries role=status, each danger role=alert. The entrance and
	// the leave fade ride the global prefers-reduced-motion collapse.
	import { dismissToast, toasts, type Toast } from './toast.svelte';
	import Check from '@lucide/svelte/icons/check';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import Info from '@lucide/svelte/icons/info';

	const live = $derived(toasts());

	/** The tone's decorative glyph: the teal check, the ember warning, the accent info mark. */
	function glyphOf(tone: Toast['tone']): typeof Check {
		if (tone === 'danger') {
			return TriangleAlert;
		}
		return tone === 'info' ? Info : Check;
	}
</script>

<div
	class="toasts fixed left-1/2 bottom-[84px] -translate-x-1/2 z-(--z-toast) flex flex-col items-center gap-2 pointer-events-none"
	aria-live="polite"
>
	{#each live as entry (entry.id)}
		{@const Glyph = glyphOf(entry.tone)}
		<button
			class="toast pointer-events-auto flex items-center gap-2 max-w-[min(420px,calc(100vw-32px))] px-3.5 py-2.5 font-body font-semibold text-[13px] leading-[1.4] text-text bg-card border border-border rounded-control shadow-overlay cursor-pointer transition-[opacity] duration-(--motion-fast) ease-standard"
			class:toast--success={entry.tone === 'success'}
			class:toast--danger={entry.tone === 'danger'}
			class:toast--info={entry.tone === 'info'}
			class:opacity-0={entry.leaving}
			type="button"
			role={entry.tone === 'danger' ? 'alert' : 'status'}
			data-toast-id={entry.id}
			onclick={() => dismissToast(entry.id)}
		>
			<Glyph size={16} class="toast__icon shrink-0" aria-hidden="true" />
			<span class="toast__text">{entry.text}</span>
		</button>
	{/each}
</div>

<style>
	/* The tone's icon color: the success check teal, the danger
	 * warning ember, the info glyph accent. */
	.toast--success .toast__icon {
		color: var(--color-success);
	}

	.toast--danger .toast__icon {
		color: var(--color-danger-text);
	}

	.toast--info .toast__icon {
		color: var(--color-accent);
	}

	/* Residual styles (the specimen's own): the entrance keyframe;
	 * the leaving fade rides the opacity transition utility above. */
	.toast {
		animation: toast-in var(--motion-base) var(--ease-standard);
	}

	@keyframes toast-in {
		from {
			opacity: 0;
			transform: translateY(8px);
		}
	}
</style>
