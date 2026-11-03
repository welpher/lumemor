<script lang="ts">
	// The tool shell of the web IA: the persistent-navigation shell
	// behind /admin/* (and, later, /settings/*). Its top bar is the
	// simplified one the IA pins -- a back-to-content mark with the
	// wordmark instead of the content shell's full bar -- and the
	// section navigation itself lives in the section layouts (the
	// admin sidebar under (tool)/admin).
	import { _ } from 'svelte-i18n';
	import { goto } from '$app/navigation';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	function backToContent(): void {
		goto('/timeline');
	}
</script>

<div class="tool">
	<header class="tool-topbar">
		<button
			class="tool-topbar__back"
			type="button"
			aria-label={$_('tool_back_to_content')}
			onclick={backToContent}
		>
			<svg viewBox="0 0 24 24" width="22" height="22" fill="none" aria-hidden="true">
				<path
					d="M14.5 6.5L8 12l6.5 5.5"
					stroke="currentColor"
					stroke-width="1.75"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
		</button>
		<span class="tool-topbar__brand">{$_('app_name')}</span>
	</header>
	<main class="tool-main">
		{@render children()}
	</main>
</div>

<style>
	.tool {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}

	.tool-topbar {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 16px;
		border-bottom: 1px solid var(--color-border);
		background-color: var(--color-card);
	}

	.tool-topbar__back {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border: none;
		border-radius: var(--radius-control);
		background: none;
		color: var(--color-text);
		cursor: pointer;
		transition: background-color var(--motion-fast) var(--ease-standard);
	}

	.tool-topbar__back:hover {
		background-color: var(--color-accent-soft);
	}

	.tool-topbar__brand {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 20px;
		letter-spacing: 0.01em;
		color: var(--color-text);
	}

	.tool-main {
		flex: 1;
		padding: 24px;
	}
</style>
