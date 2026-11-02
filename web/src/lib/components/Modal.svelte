<script lang="ts">
	import type { Snippet } from 'svelte';
	import { _ } from 'svelte-i18n';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** Accessible name of the dialog. */
		title: string;
		/** Optional footer row, e.g. a cancel and a confirm button. */
		footer?: Snippet;
		children?: Snippet;
	};

	let { open = $bindable(false), title, footer, children }: Props = $props();

	function close(): void {
		open = false;
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (open && event.key === 'Escape') {
			close();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<!-- mousedown (not click) so text selections started inside the
	     panel never count as backdrop dismissals. The Escape key on
	     svelte:window is the keyboard equivalent of this dismissal. -->
	<div
		class="modal__backdrop"
		role="presentation"
		onmousedown={close}
	>
		<div
			class="modal__panel"
			role="dialog"
			aria-modal="true"
			aria-label={title}
			tabindex="-1"
			onmousedown={(event) => event.stopPropagation()}
		>
			<header class="modal__header">
				<h2 class="modal__title">{title}</h2>
				<button class="modal__close" type="button" aria-label={$_('common_close')} onclick={close}>
					<svg viewBox="0 0 24 24" width="20" height="20" fill="none" aria-hidden="true">
						<path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
					</svg>
				</button>
			</header>
			<div class="modal__body">
				{@render children?.()}
			</div>
			{#if footer}
				<footer class="modal__footer">
					{@render footer()}
				</footer>
			{/if}
		</div>
	</div>
{/if}

<style>
	.modal__backdrop {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 24px;
		background-color: rgb(20 26 46 / 0.55);
		animation: modal-fade var(--motion-fast) var(--ease-standard);
	}

	.modal__panel {
		width: 100%;
		max-width: 480px;
		max-height: calc(100vh - 48px);
		display: flex;
		flex-direction: column;
		background-color: var(--color-card);
		border-radius: var(--radius-overlay);
		box-shadow: var(--shadow-overlay);
		animation: modal-in var(--motion-base) var(--ease-standard);
	}

	.modal__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 16px 20px 0;
	}

	.modal__title {
		font-size: 18px;
	}

	.modal__close {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border: none;
		border-radius: var(--radius-control);
		background: none;
		color: var(--color-text-soft);
		cursor: pointer;
		transition:
			background-color var(--motion-fast) var(--ease-standard),
			color var(--motion-fast) var(--ease-standard);
	}

	.modal__close:hover {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	.modal__body {
		padding: 16px 20px;
		overflow-y: auto;
	}

	.modal__footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		padding: 0 20px 20px;
	}

	@keyframes modal-fade {
		from {
			opacity: 0;
		}
	}

	@keyframes modal-in {
		from {
			opacity: 0;
			transform: scale(0.98);
		}
	}
</style>
