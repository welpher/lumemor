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

<div class="tool min-h-screen flex flex-col">
	<header class="tool-topbar flex items-center gap-3 px-4 py-2 border-b border-border bg-card">
		<button
			class="tool-topbar__back inline-flex items-center justify-center w-9 h-9 rounded-control bg-transparent text-text cursor-pointer transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft"
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
		<span class="tool-topbar__brand font-display font-bold text-[20px] text-text">{$_('app_name')}</span>
	</header>
	<main class="tool-main flex-1 p-6">
		{@render children()}
	</main>
</div>
