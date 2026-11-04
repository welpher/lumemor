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

	function handlePointerCancel(): void {
		dragging = false;
		// A cancelled press (the host took the gesture over) is
		// followed by no click, so the next genuine tap must jump.
		suppressClick = false;
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
	<nav class="scrubber fixed top-1/2 right-1 -translate-y-1/2 z-[40]" aria-label={$_('timeline_scrubber_label')}>
		<!-- The drag owns the pointer: no touch scrolling fights it. -->
		<div
			class="scrubber__rail flex flex-col gap-0.5 px-1 py-1.5 rounded-control bg-[color-mix(in_srgb,var(--color-card)_85%,transparent)] shadow-card touch-none"
			bind:this={rail}
		>
			{#each months as month (month.key)}
				<button
					class="scrubber__stop px-2 py-[3px] font-body font-semibold text-[11px] leading-[1.4] whitespace-nowrap rounded-control bg-transparent cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
					class:scrubber__stop--active={activeKey === month.key}
					class:bg-accent-soft={activeKey === month.key}
					class:bg-transparent={activeKey !== month.key}
					class:text-accent-soft-text={activeKey === month.key}
					class:text-text-soft={activeKey !== month.key}
					type="button"
					aria-current={activeKey === month.key ? 'true' : undefined}
					onpointerdown={(event) => handlePointerDown(event, month.key)}
					onpointermove={handlePointerMove}
					onpointerup={handlePointerUp}
					onpointercancel={handlePointerCancel}
					onclick={() => handleClick(month.key)}
				>
					{month.label}
				</button>
			{/each}
		</div>
	</nav>
{/if}
