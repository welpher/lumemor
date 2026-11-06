<script lang="ts">
	// The right-edge timeline scrubber: a proportional year axis that
	// IS a scrollbar. At rest the years (one per year run), the
	// month dots (positioned by each month's share of the loaded
	// photo mass), and the position indicator carry the rail. A drag
	// scrubs the wall CONTINUOUSLY -- the pointer's rail fraction
	// maps onto the scroll position, within months, not between
	// them -- with the bubble naming the month under the pointer; a
	// pure tap (or the keyboard) jumps to a month group. v0 scrubs
	// within the loaded data only; deep jumps with date filters
	// arrive with the M3 timeline.
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
		/** The month the wall currently shows (the highlighted dot). */
		activeKey?: string | null;
		/** The wall's overall scroll progress (0-1) -- the position
		 * indicator's truth. */
		progress?: number;
		/** Jump callback (a tap or the keyboard): the page scrolls the
		 * month group into view. */
		onjump: (key: string) => void;
		/** Scrub callback (the drag): the pointer's rail fraction,
		 * mapped by the page onto the wall's continuous scroll
		 * position -- the rail IS a scrollbar. */
		onscrub?: (fraction: number) => void;
	};

	let { months, activeKey = null, progress = 0, onjump, onscrub }: Props = $props();

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
		label: string;
		/** The compact year-month readout ("Nov 2026" / "2026年11月"). */
		compact: string;
		/** The faint year label, only where the year changes. */
		year: string | null;
		top: number;
		center: number;
	};

	/** The positioned stops: data-proportional first, then a forward
	 * relaxation pass keeps every dot legible, and a final
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
				label: month.label,
				compact: compactMonth(Number(monthNumber) - 1, year, $locale),
				year: yearChanges ? year : null,
				top: byData,
				center: byData + STOP_HEIGHT / 2
			};
		});
		// The relaxation pass: no dot may ride onto the one above.
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

	/** The locale's compact year-month readout ("Nov 2026" /
	 * "2026年11月") -- narrow enough to live wholly in the reserved
	 * gutter, left of the dots. */
	function compactMonth(monthIndex: number, year: string, activeLocale: string | null | undefined): string {
		const formatter = new Intl.DateTimeFormat(activeLocale ?? 'en', {
			month: 'short',
			year: 'numeric',
			timeZone: 'UTC'
		});
		return formatter.format(new Date(Date.UTC(Number(year), monthIndex, 1)));
	}

	let rail: HTMLElement | undefined = $state();
	let dragging = $state(false);
	// A rail pointerdown already jumped; the click that follows the
	// release lands on the PRESSED button (the pointer capture) whose
	// month may differ from the drag's destination -- it must not
	// jump back.
	let suppressClick = false;
	// The month whose label the rail reveals (hover or keyboard
	// focus); the drag keeps the last jumped month revealed as the
	// live readout.
	let revealedKey: string | null = $state(null);
	// The rail is ACTIVE (the readout bubble visible) while the
	// pointer is over it, a drag is in flight, or the wall is
	// scrolling. The faint years and the position indicator rest on
	// the rail permanently.
	let hovering = $state(false);
	let scrollActive = $state(false);
	let scrollQuietTimer: ReturnType<typeof setTimeout> | undefined;
	// The pointer's height inside the rail -- the bubble rides it,
	// not the stop's position (the reference behavior).
	let hoverY = $state(0);

	// The wall's scrolling wakes the rail: element scrolls do not
	// bubble, so the capture phase at the document reaches every
	// scroll root (the timeline page's own listener established the
	// pattern). Eight hundred milliseconds of quiet settles it.
	$effect(() => {
		const onScroll = () => {
			scrollActive = true;
			if (scrollQuietTimer !== undefined) {
				clearTimeout(scrollQuietTimer);
			}
			scrollQuietTimer = setTimeout(() => {
				scrollActive = false;
			}, 800);
		};
		document.addEventListener('scroll', onScroll, { capture: true, passive: true });
		return () => {
			document.removeEventListener('scroll', onScroll, { capture: true });
			if (scrollQuietTimer !== undefined) {
				clearTimeout(scrollQuietTimer);
			}
		};
	});

	/** The rail's active state: any presence wakes the readout
	 * bubble; the years and the indicator rest permanently. */
	const railActive = $derived(hovering || dragging || scrollActive);

	/** The position indicator's top: the wall's scroll progress
	 * mapped onto the rail's usable span. */
	const indicatorTop = $derived(Math.min(1, Math.max(0, progress)) * Math.max(0, railHeight - STOP_HEIGHT));

	/** The readout bubble's stop (which month it names) and top
	 * (where it rides): the pointer's height while present, the
	 * indicator's height while the wall alone scrolls. */
	const readout = $derived.by(() => {
		if (!railActive) {
			return null;
		}
		const revealedStop = stops.find((stop) => stop.key === revealedKey);
		const stop = revealedStop ?? stops.find((candidate) => candidate.key === activeKey) ?? null;
		if (stop === null) {
			return null;
		}
		const top = hovering || dragging ? Math.min(Math.max(hoverY, 0), Math.max(0, railHeight - STOP_HEIGHT)) : indicatorTop;
		return { stop, top };
	});

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
			revealedKey = key;
			onjump(key);
		}
	}

	/** The pointer's rail fraction (0-1, clamped); 0 when the host
	 * answers no layout (the tests' stand-ins scrub to the top). */
	function pointerFraction(event: PointerEvent): number {
		if (rail === undefined) {
			return 0;
		}
		const rect = rail.getBoundingClientRect();
		if (rect.height <= 0) {
			return 0;
		}
		return Math.min(1, Math.max(0, (event.clientY - rect.top) / rect.height));
	}

	/** THE TRUE DRAG: the wall scrolls continuously with the pointer
	 * -- the rail IS a scrollbar. The press scrubs from its own
	 * position; every move scrubs again; the bubble names the month
	 * under the pointer while the wall's own scroll tracking moves
	 * the active dot. */
	function scrub(event: PointerEvent): void {
		onscrub?.(pointerFraction(event));
		revealedKey = nearestKey(event.clientY);
	}

	function handlePointerDown(event: PointerEvent): void {
		dragging = true;
		// A press that never moves is a TAP: its trailing click jumps
		// to the month (the button semantics). A drag that moved
		// suppresses the trailing click.
		suppressClick = false;
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
		scrub(event);
	}

	function handlePointerMove(event: PointerEvent): void {
		// A hover (not pressed) reveals the nearest month's label;
		// a drag scrubs the wall to it. The bubble rides the
		// pointer's own height either way.
		hovering = true;
		if (rail !== undefined) {
			const rect = rail.getBoundingClientRect();
			if (rect.height > 0) {
				hoverY = event.clientY - rect.top - STOP_HEIGHT / 2;
			}
		}
		if (!dragging) {
			revealedKey = nearestKey(event.clientY);
			return;
		}
		suppressClick = true;
		scrub(event);
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

	function handlePointerLeave(): void {
		// Leaving the rail clears the hover readout; the active month
		// stays marked by its dot, not a label.
		hovering = false;
		if (!dragging) {
			revealedKey = null;
		}
	}

	function handleFocus(key: string): void {
		// The keyboard path sees the same reveal the pointer does.
		revealedKey = key;
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
		<!-- The drag owns the pointer: no touch scrolling fights it.
		     The rail is transparent: its gutter is its background. -->
		<!-- svelte-ignore a11y_no_static_element_interactions -- the rail is a pointer surface of the labeled nav above it; its buttons carry the semantics. -->
		<div
			class="scrubber__rail relative w-[72px] touch-none"
			bind:this={rail}
			style="height: {railHeight}px"
			onpointerdown={handlePointerDown}
			onpointermove={handlePointerMove}
			onpointerleave={handlePointerLeave}
			onpointerup={handlePointerUp}
			onpointercancel={handlePointerCancel}
		>
			<!-- The axis spine the years and dots ride on. -->
			<span class="scrubber__spine absolute top-1 bottom-1 right-[6px] w-px bg-border" aria-hidden="true"></span>
			<!-- The years rest on the rail permanently (faint, one per
			     year run); the year-MONTH readout is the presence-driven
			     bubble below. -->
			{#each stops as stop (stop.key)}
				{#if stop.year !== null}
					<span
						class="scrubber__year absolute right-[18px] font-body font-normal text-[10px] leading-[16px] text-text-faint tabular-nums"
						style="top: {stop.top}px"
						aria-hidden="true"
					>{stop.year}</span>
				{/if}
			{/each}
			{#each stops as stop (stop.key)}
				<button
					class="scrubber__stop absolute right-0 flex items-center justify-center w-[12px] h-[16px] cursor-pointer"
					class:scrubber__stop--active={activeKey === stop.key}
					style="top: {stop.top}px"
					type="button"
					aria-label={stop.label}
					aria-current={activeKey === stop.key ? 'true' : undefined}
					onfocus={() => handleFocus(stop.key)}
					onclick={() => handleClick(stop.key)}
				>
					<span
						class="scrubber__dot w-[5px] h-[5px] rounded-full {activeKey === stop.key ? 'bg-accent' : 'bg-border-strong'}"
						aria-hidden="true"
					></span>
				</button>
			{/each}
			{#if !dragging && months.length > 0}
				<!-- The position indicator: the wall's scroll progress as an
				     accent tick crossing the spine -- the rail's resting
				     answer to "where am I" (hidden mid-drag: the bubble is
				     the feedback then). -->
				<span
					class="scrubber__indicator absolute right-0 w-6 h-[2px] rounded-full bg-accent"
					style="top: {indicatorTop + 7}px"
					aria-hidden="true"
				></span>
			{/if}
			{#if readout !== null}
				<!-- The readout bubble: the nearest (or active-while-scrolling)
				     month's full label on a fully-rounded opaque card -- its
				     vertical CENTER rides the
				     pointer while present, the indicator tick while the
				     wall alone scrolls; left of the dots, wholly inside the
				     reserved gutter. -->
				<span
					class="scrubber__readout absolute right-[18px] inline-flex items-center h-[20px] max-w-[88px] truncate px-1.5 font-body font-semibold text-[11px] leading-none whitespace-nowrap text-text bg-card border border-border rounded-control shadow-card"
					style="top: {readout.top - 10}px"
					aria-hidden="true"
				>{readout.stop.label}</span>
			{/if}
		</div>
	</nav>
{/if}
