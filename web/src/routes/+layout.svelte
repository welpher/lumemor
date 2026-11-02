<script lang="ts">
	// Skeleton of the app shell from the web IA design: a top bar with
	// the drawer toggle (hamburger), the instance name, the search
	// entry, the upload action and the account avatar. Static
	// placeholders only -- later tasks wire the drawer, search, upload
	// and account behaviors and split the tool and auth shells into
	// their own route-group layouts.
	let { children } = $props();
</script>

<div class="shell">
	<header class="shell-topbar">
		<button class="topbar-menu" type="button" aria-label="Open navigation (placeholder)">☰</button>
		<span class="topbar-brand">lumemor</span>
		<input
			class="topbar-search"
			type="search"
			placeholder="Search (placeholder)"
			aria-label="Search (placeholder)"
		/>
		<button class="topbar-upload" type="button" aria-label="Upload (placeholder)">Upload</button>
		<span class="topbar-avatar" aria-label="Account (placeholder)" title="Account (placeholder)"
			>●</span
		>
	</header>
	<main class="shell-main">
		{@render children()}
	</main>
</div>

<style>
	.shell {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
		font: 16px/1.6 system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif;
	}

	.shell-topbar {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 16px;
		border-bottom: 1px solid rgba(128, 128, 128, 0.35);
	}

	.topbar-menu,
	.topbar-upload,
	.topbar-avatar {
		background: none;
		border: none;
		font: inherit;
		cursor: pointer;
	}

	.topbar-brand {
		font-weight: 700;
	}

	.topbar-search {
		/* Push the search box away from the brand; the upload and
			account marks stay pinned to the right end. */
		flex: 1;
		max-width: 420px;
		margin-left: auto;
	}

	.topbar-avatar {
		width: 32px;
		height: 32px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border-radius: 50%;
		border: 1px solid rgba(128, 128, 128, 0.5);
	}

	.shell-main {
		flex: 1;
		padding: 24px;
	}
</style>
