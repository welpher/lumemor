<script lang="ts">
	// The right-edge timeline scrubber: a proportional month axis.
	// One stop per LOADED month, positioned by the month's share of
	// the loaded photo mass -- a month with many photos occupies more
	// rail than a sparse one, so the rail reads like the timeline it
	// navigates instead of an even list. Labels are short month names
	// (the year appears once, at each year boundary, as a faint
	// prefix). Clicking a stop jumps to its group; dragging anywhere
	// on the rail scrubs continuously -- the pointer's height maps
	// onto the nearest stop's center. v0 scrubs within the loaded
	// data only; deep jumps with date filters arrive with the M3
	// timeline.
	import { _, locale } from 'svelte-i18n';

	type MonthEntry = {
		/** `YYYY-MM`, the jump target key of the month group. */
		key: string;
		/** Locale-aware month label (the wall header's form). */
		label: string;
		/** The month's photo count (its share of the rail). */
		count?: number;
	};

	type Props = {
		months: readonly MonthEntry[];
		/** The month the wall currently shows (the highlighted stop). */
		activeKey?: string | null;
		/** Jump callback: the page scrolls the month group into view. */
		onjump: (key: string) => void;
	};

	let { months, activeKey = null, onjump }: Props = $props();

	// One stop row's height on the rail (the anti-overlap unit).
	const STOP_HEIGHT = 16;

	/** The rail's pixel height: dense enough that N stops never
	 * overlap, capped for tall loaded histories, floored so a single
	 * month still reads as a rail. */
	const railHeight = $derived(
		Math.min(560, Math.max(140, months.length * 20))
	);

	type Stop = {
		key: string;
		short: string;
		/** The faint year prefix, only where the year changes. */
		year: string | null;
		top: number;
		center: number;
	};

	/** The positioned stops: data-proportional first, then a forward
	 * relaxation pass keeps every label legible, and a final
	 * proportional squeeze fits a dense history into the capped
	 * rail. Deterministic -- the same months always pack the same. */
	const stops = $derived.by(() => {
		if (months.length === 0) {
			return [] as Stop[];
		}
		const counts = months.map((month) => Math.max(1, month.count ?? 1));
		const total = counts.reduce((sum, count) => sum + count, 0);
		const usable = Math.max(STOP_HEIGHT, railHeight - STOP_HEIGHT);
		let cumulative = 0;
		const laid: Stop[] = months.map((month, index) => {
			const [year, monthNumber] = month.key.split('-');
			const previous = months[index - 1];
			const yearChanges = previous === undefined || previous.key.split('-')[0] !== year;
			const byData = (cumulative / total) * usable;
			cumulative += counts[index];
			return {
				key: month.key,
				short: shortMonth(Number(monthNumber) - 1, $locale),
				year: yearChanges ? year : null,
				top: byData,
				center: byData + STOP_HEIGHT / 2
			};
		});
		// The relaxation pass: no label may ride onto the one above.
		for (let i = 1; i < laid.length; i += 1) {
			laid[i].top = Math.max(laid[i].top, laid[i - 1].top + STOP_HEIGHT);
			laid[i].center = laid[i].top + STOP_HEIGHT / 2;
		}
		// The squeeze pass: a capped rail with many months scales
		// everything back into the usable span.
		const last = laid[laid.length - 1];
		if (last !== undefined && last.top > usable) {
			const scale = usable / last.top;
			for (const stop of laid) {
				stop.top *= scale;
				stop.center = stop.top + STOP_HEIGHT / 2;
			}
		}
		return laid;
	});

	/** The locale's short month name ("Nov" / "11月"). */
	function shortMonth(monthIndex: number, activeLocale: string | null | undefined): string {
		const formatter = new Intl.DateTimeFormat(activeLocale ?? 'en', {
			month: 'short',
			timeZone: 'UTC'
		});
		return formatter.format(new Date(Date.UTC(2026, monthIndex, 1)));
	}

	let rail: HTMLElement | undefined = $state();
	let dragging = false;
	// A rail pointerdown already jumped; the click that follows it on
	// the button underneath must not jump again.
	let suppressClick = false;

	/** The stop whose center is nearest the pointer's height. The
	 * comparison runs in rail fractions (the pointer's fraction of
	 * the measured rect against the stop centers' fractions of the
	 * layout height), so any measured height resolves the same. */
	function nearestKey(clientY: number): string | null {
		if (rail === undefined || stops.length === 0) {
			return null;
		}
		const rect = rail.getBoundingClientRect();
		// Layout-less hosts (jsdom) answer a zero-height rail: the top
		// month is the only sane answer.
		if (rect.height <= 0) {
			return stops[0]?.key ?? null;
		}
		const fraction = Math.min(1, Math.max(0, (clientY - rect.top) / rect.height));
		let nearest = stops[0];
		for (const stop of stops) {
			if (Math.abs(stop.center / railHeight - fraction) < Math.abs(nearest.center / railHeight - fraction)) {
				nearest = stop;
			}
		}
		return nearest?.key ?? null;
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
	<nav class="scrubber fixed top-1/2 right-3 -translate-y-1/2 z-(--z-page)" aria-label={$_('timeline_scrubber_label')}>
		<!-- The drag owns the pointer: no touch scrolling fights it. -->
		<div
			class="scrubber__rail relative rounded-full bg-[color-mix(in_srgb,var(--color-card)_80%,transparent)] px-1.5 touch-none"
			bind:this={rail}
			style="height: {railHeight}px"
		>
			<!-- The axis spine the stops ride on. -->
			<span class="scrubber__spine absolute top-1 bottom-1 right-[13px] w-px bg-border" aria-hidden="true"></span>
			{#each stops as stop (stop.key)}
				<button
					class="scrubber__stop absolute right-1 flex items-center gap-1.5 h-[16px] pr-1.5 font-body font-semibold text-[10px] leading-[1] whitespace-nowrap cursor-pointer transition-[color] duration-(--motion-fast) ease-standard hover:text-accent-soft-text"
					class:scrubber__stop--active={activeKey === stop.key}
					class:text-accent-soft-text={activeKey === stop.key}
					class:text-text-soft={activeKey !== stop.key}
					style="top: {stop.top}px"
					type="button"
					aria-current={activeKey === stop.key ? 'true' : undefined}
					onpointerdown={(event) => handlePointerDown(event, stop.key)}
					onpointermove={handlePointerMove}
					onpointerup={handlePointerUp}
					onpointercancel={handlePointerCancel}
					onclick={() => handleClick(stop.key)}
				>
					{#if stop.year !== null}
						<span class="scrubber__year text-[9px] font-normal text-text-faint tabular-nums">{stop.year}</span>
					{/if}
					<span class="scrubber__label">{stop.short}</span>
					<span
						class="scrubber__dot w-[5px] h-[5px] rounded-full {activeKey === stop.key ? 'bg-accent' : 'bg-border-strong'}"
						aria-hidden="true"
					></span>
				</button>
			{/each}
		</div>
	</nav>
{/if}
