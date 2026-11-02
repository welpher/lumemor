<script lang="ts">
	// The right-edge month scrubber of the timeline wall: one button
	// per loaded month (newest at the top, matching the wall's
	// order). Clicking a button jumps to that month's group; dragging
	// anywhere on the rail scrubs continuously -- the pointer's
	// vertical fraction of the rail maps onto the month index. v0
	// scrubs within the loaded data only; deep jumps with date
	// filters arrive with the M3 timeline.
	import { _ } from 'svelte-i18n';

	type MonthEntry = {
		/** `YYYY-MM`, the jump target key of the month group. */
		key: string;
		/** Locale-aware month label. */
		label: string;
	};

	type Props = {
		months: readonly MonthEntry[];
		/** The month the wall currently shows (the highlighted stop). */
		activeKey?: string | null;
		/** Jump callback: the page scrolls the month group into view. */
		onjump: (key: string) => void;
	};

	let { months, activeKey = null, onjump }: Props = $props();

	let rail: HTMLElement | undefined = $state();
	let dragging = false;
	// A rail pointerdown already jumped; the click that follows it on
	// the button underneath must not jump again.
	let suppressClick = false;

	/** The month nearest the pointer's height on the rail. */
	function nearestKey(clientY: number): string | null {
		if (rail === undefined || months.length === 0) {
			return null;
		}
		const rect = rail.getBoundingClientRect();
		// Layout-less hosts (jsdom) answer a zero-height rail: the top
		// month is the only sane answer.
		if (rect.height <= 0) {
			return months[0]?.key ?? null;
		}
		const fraction = Math.min(1, Math.max(0, (clientY - rect.top) / rect.height));
		const index = Math.min(months.length - 1, Math.floor(fraction * months.length));
		return months[index]?.key ?? null;
	}

	function jumpTo(key: string | null): void {
		if (key !== null) {
			onjump(key);
		}
	}

	function handlePointerDown(event: PointerEvent, key: string): void {
		dragging = true;
		suppressClick = true;
		// Capture keeps the moves arriving on the pressed stop even
		// while the pointer travels past its neighbors; hosts without
		// the method (jsdom) simply deliver moves where they land.
		const target = event.currentTarget as HTMLElement | null;
		if (target !== null && typeof target.setPointerCapture === 'function') {
			try {
				target.setPointerCapture(event.pointerId);
			} catch {
				// A lost pointer id just means no capture.
			}
		}
		jumpTo(key);
	}

	function handlePointerMove(event: PointerEvent): void {
		if (!dragging) {
			return;
		}
		jumpTo(nearestKey(event.clientY));
	}

	function handlePointerUp(): void {
		dragging = false;
	}

	function handleClick(key: string): void {
		if (suppressClick) {
			suppressClick = false;
			return;
		}
		onjump(key);
	}
</script>

{#if months.length > 0}
	<nav class="scrubber" aria-label={$_('timeline_scrubber_label')}>
		<div class="scrubber__rail" bind:this={rail}>
			{#each months as month (month.key)}
				<button
					class="scrubber__stop"
					class:scrubber__stop--active={activeKey === month.key}
					type="button"
					aria-current={activeKey === month.key ? 'true' : undefined}
					onpointerdown={(event) => handlePointerDown(event, month.key)}
					onpointermove={handlePointerMove}
					onpointerup={handlePointerUp}
					onpointercancel={handlePointerUp}
					onclick={() => handleClick(month.key)}
				>
					{month.label}
				</button>
			{/each}
		</div>
	</nav>
{/if}

<style>
	.scrubber {
		position: fixed;
		top: 50%;
		right: 4px;
		transform: translateY(-50%);
		z-index: 40;
	}

	.scrubber__rail {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 6px 4px;
		border-radius: var(--radius-control);
		background-color: color-mix(in srgb, var(--color-card) 85%, transparent);
		box-shadow: var(--shadow-card);
		/* The drag owns the pointer: no touch scrolling fights it. */
		touch-action: none;
	}

	.scrubber__stop {
		padding: 3px 8px;
		font: 600 11px/1.4 var(--font-body);
		white-space: nowrap;
		color: var(--color-text-soft);
		background: none;
		border: none;
		border-radius: var(--radius-control);
		cursor: pointer;
		transition:
			background-color var(--motion-fast) var(--ease-standard),
			color var(--motion-fast) var(--ease-standard);
	}

	.scrubber__stop:hover {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}

	.scrubber__stop--active {
		background-color: var(--color-accent-soft);
		color: var(--color-accent-soft-text);
	}
</style>
