<script lang="ts">
	// THE MONTH AXIS AS ONE CONTROL: a scrollbar, not a row of
	// buttons. The whole rail takes the pointer (the dots and the
	// years are decoration and never eat an event): a press jumps
	// where you point -- the pointer's height maps onto the wall's
	// scroll span and the wall is there before the release -- holding
	// drags the wall along, and the readout bubble follows the
	// pointer while it hovers or drags and rides the position
	// indicator while the wall scrolls alone. The keyboard is one
	// tab stop walking the months (the readout is the focus
	// indicator; a box ring on a 72px rail reads as a broken style).
	// The geometry is the wall's own: every dot sits at the scroll
	// position that lands its month header on the reading line, so
	// the dot you see, the place a press takes you and the month the
	// readout names are ONE mapping -- the wall feeds `tops` and
	// `maxScroll` and nothing here re-derives them. v0 maps the
	// loaded span only; deep jumps with date filters arrive later.
	import { _ } from 'svelte-i18n';

	type MonthEntry = {
		/** `YYYY-MM`, the jump target key of the month group. */
		key: string;
		/** Locale-aware month label (the wall header's form). */
		label: string;
	};

	type Props = {
		months: readonly MonthEntry[];
		/**
		 * Per month (wall order): the scroll position that lands its
		 * header on the reading line -- the landing the wall's own
		 * jumps use. The dots, the press and the readout all read
		 * this one array.
		 */
		tops?: readonly number[];
		/** The scroll root's scrollable span (the mapping's denominator). */
		maxScroll?: number;
		/** The month the wall currently reads (the highlighted dot). */
		activeKey?: string | null;
		/** The wall's scroll progress (0-1) -- the indicator's truth. */
		progress?: number;
		/**
		 * The id of the element this scrollbar controls (the ARIA
		 * contract's own requirement): the page rail drives the
		 * shell's <main>; the panel rail drives the wall's own root.
		 */
		controls?: string;
		/** `page` is the fixed right-edge rail (the wall pages' 72px
		 * slot); `panel` is the SAME control in the map panel's
		 * compact 24px slot (the axis-as-scrollbar discipline: the
		 * panel never grows a second organ). */
		variant?: 'page' | 'panel';
		/** Walk callback (the keyboard): the page scrolls the month into view. */
		onjump: (key: string) => void;
		/** Scrub callback (the press and the drag): the pointer's rail
		 * fraction, which the wall maps onto its scroll span. */
		onscrub?: (fraction: number) => void;
	};

	let {
		months,
		tops = [],
		maxScroll = 0,
		activeKey = null,
		progress = 0,
		controls = 'shell-main',
		variant = 'page',
		onjump,
		onscrub
	}: Props = $props();

	// The rim pad: the dots' half-height, kept clear of both rail
	// ends so a dot at the very top or bottom never clips.
	const PAD = 8;

	/** The page rail's pixel height: dense enough that N months never
	 * crowd, capped for tall loaded histories, floored so a single
	 * month still reads as a rail. The panel rail is the panel's own
	 * height (measured live); the same fallback covers a layout-less
	 * host there. */
	const fallbackHeight = $derived(Math.min(560, Math.max(140, months.length * 20)));

	/** The panel rail's measured height (its slot is the panel's
	 * height minus its insets -- not a number the component can
	 * compute). jsdom has no observer: the fallback stands in. */
	let measuredHeight = $state(0);

	const railHeight = $derived(variant === 'panel' && measuredHeight > 0 ? measuredHeight : fallbackHeight);

	let rail: HTMLElement | undefined = $state();

	$effect(() => {
		const element = rail;
		if (variant !== 'panel' || element === undefined || typeof ResizeObserver !== 'function') {
			return;
		}
		const observer = new ResizeObserver((entries) => {
			for (const entry of entries) {
				const block = entry.contentBoxSize[0]?.blockSize ?? entry.contentRect.height;
				if (typeof block === 'number' && block > 0) {
					measuredHeight = block;
				}
			}
		});
		observer.observe(element);
		return () => {
			observer.disconnect();
		};
	});

	// The timers outlive any single render: a fade schedule set in a
	// pointer handler must die with the component, not leak.
	let scrollTimer: ReturnType<typeof setTimeout> | undefined;
	let lingerTimer: ReturnType<typeof setTimeout> | undefined;

	$effect(() => {
		return () => {
			if (scrollTimer !== undefined) {
				clearTimeout(scrollTimer);
			}
			if (lingerTimer !== undefined) {
				clearTimeout(lingerTimer);
			}
		};
	});

	// The usable span between the rim pads: every y on this rail --
	// dot, indicator, tick, pointer -- crosses it through the ONE
	// linear mapping, so no two of them can ever disagree.
	const span = $derived(Math.max(0, railHeight - PAD * 2));

	const clamp01 = (value: number): number => Math.min(1, Math.max(0, value));

	/** The rail fraction of a scroll position (the drawing side). */
	function fractionOfTop(top: number): number {
		return maxScroll > 0 ? clamp01(top / maxScroll) : 0;
	}

	/** The scroll fraction of a pointer height (the press side). */
	function fractionOfY(y: number): number {
		return span > 0 ? clamp01((y - PAD) / span) : 0;
	}

	/** The positioned marks: one per month, at the fraction of the
	 * landing the wall reported -- never a static top. */
	const marks = $derived.by(() =>
		months.map((month, index) => {
			const year = month.key.split('-')[0] ?? '';
			const previous = months[index - 1];
			const yearChanges = previous === undefined || previous.key.split('-')[0] !== year;
			return {
				key: month.key,
				label: month.label,
				year: yearChanges ? year : null,
				y: PAD + fractionOfTop(tops[index] ?? 0) * span
			};
		})
	);

	/** The month at a scroll position: the last month whose landing
	 * sits at or above it, with the span's far end taking the last
	 * month (the same mapping the dots draw, read backwards). */
	function monthIndexAt(scrollTop: number): number {
		let index = 0;
		for (let i = 0; i < months.length; i += 1) {
			if ((tops[i] ?? 0) <= scrollTop) {
				index = i;
			}
		}
		if (maxScroll > 2 && scrollTop >= maxScroll - 2) {
			index = months.length - 1;
		}
		return index;
	}

	/** The keyboard's base month: the wall's own reading when it
	 * tracks one, the mapping's answer otherwise. */
	const activeIndex = $derived.by(() => {
		const at = months.findIndex((month) => month.key === activeKey);
		return at >= 0 ? at : monthIndexAt(clamp01(progress) * maxScroll);
	});

	const activeLabel = $derived(months[activeIndex]?.label ?? '');

	// The axis is AWAKE while the pointer owns it (hover or drag), a
	// released drag still lingers, the keyboard focus holds it (the
	// readout is the focus indicator), or the wall scrolled recently.
	let hovering = $state(false);
	let dragging = $state(false);
	let lingering = $state(false);
	let focused = $state(false);
	let scrollActive = $state(false);

	const railActive = $derived(hovering || dragging || lingering || focused || scrollActive);

	// The pointer's height on the rail: the bubble and the tick ride
	// it while the pointer owns the axis.
	let pointerY = $state(0);

	/** The readout's content and height: the month a press at the
	 * pointer would land while the pointer owns the axis; the wall's
	 * own reading riding the indicator otherwise (the scroll case).
	 * The element stays mounted and fades by opacity, so the leaves
	 * are fades, not pops. */
	const bubbleLabel = $derived(
		hovering || dragging ? (months[monthIndexAt(fractionOfY(pointerY) * maxScroll)]?.label ?? '') : activeLabel
	);
	const bubbleTop = $derived(hovering || dragging ? pointerY : PAD + clamp01(progress) * span);
	const bubbleVisible = $derived(railActive && months.length > 0);

	// The wall's scrolling wakes the axis: element scrolls do not
	// bubble, so the capture phase at the document reaches every
	// scroll root (the timeline page's own listener established the
	// pattern). Nine hundred quiet milliseconds settle it.
	$effect(() => {
		const onScroll = () => {
			scrollActive = true;
			if (scrollTimer !== undefined) {
				clearTimeout(scrollTimer);
			}
			scrollTimer = setTimeout(() => {
				scrollActive = false;
			}, 900);
		};
		document.addEventListener('scroll', onScroll, { capture: true, passive: true });
		return () => {
			document.removeEventListener('scroll', onScroll, { capture: true });
			if (scrollTimer !== undefined) {
				clearTimeout(scrollTimer);
			}
		};
	});

	/** The pointer's height clamped into the rail (0 when the host
	 * answers no layout -- the tests' stand-ins). */
	function railY(event: PointerEvent): number {
		if (rail === undefined) {
			return 0;
		}
		const rect = rail.getBoundingClientRect();
		if (rect.height <= 0) {
			return 0;
		}
		return Math.min(rect.height, Math.max(0, event.clientY - rect.top));
	}

	/** THE PRESS IS THE JUMP: the wall scrolls to the pointer's
	 * position on the down, not on the release; every move of a held
	 * press scrubs again (the rail IS a scrollbar). Capture keeps the
	 * moves arriving while the pointer travels anywhere; hosts
	 * without the method (jsdom) simply deliver moves where they
	 * land. */
	function handlePointerDown(event: PointerEvent): void {
		if (event.button !== 0) {
			return;
		}
		dragging = true;
		pointerY = railY(event);
		const target = event.currentTarget as HTMLElement | null;
		if (target !== null && typeof target.setPointerCapture === 'function') {
			try {
				target.setPointerCapture(event.pointerId);
			} catch {
				// A lost pointer id just means no capture.
			}
		}
		onscrub?.(fractionOfY(pointerY));
	}

	function handlePointerMove(event: PointerEvent): void {
		pointerY = railY(event);
		if (!dragging) {
			// The hover: the readout follows the pointer and names the
			// month a press there would land -- the wall stays put.
			hovering = true;
			return;
		}
		onscrub?.(fractionOfY(pointerY));
	}

	/** The release: the indicator bar comes back (the tick made it
	 * redundant mid-drag) and the readout lingers a beat before it
	 * fades -- unless the pointer still hovers, which owns it until
	 * it leaves. */
	function endDrag(): void {
		if (!dragging) {
			return;
		}
		dragging = false;
		lingering = !hovering;
		if (lingering) {
			if (lingerTimer !== undefined) {
				clearTimeout(lingerTimer);
			}
			lingerTimer = setTimeout(() => {
				lingering = false;
			}, 700);
		}
	}

	function handlePointerUp(): void {
		endDrag();
	}

	function handlePointerCancel(): void {
		endDrag();
	}

	function handlePointerLeave(): void {
		hovering = false;
	}

	// THE KEYBOARD: one tab stop, the arrows walk the months, Home
	// and End take the ends. The walk rides the page's own jump (the
	// same smooth landing the dots' months get).
	function handleKeydown(event: KeyboardEvent): void {
		const at = activeIndex;
		let target: number | null = null;
		if (event.key === 'ArrowUp') {
			target = at - 1;
		} else if (event.key === 'ArrowDown') {
			target = at + 1;
		} else if (event.key === 'Home') {
			target = 0;
		} else if (event.key === 'End') {
			target = months.length - 1;
		}
		if (target === null) {
			return;
		}
		event.preventDefault();
		const month = months[Math.min(months.length - 1, Math.max(0, target))];
		if (month !== undefined) {
			onjump(month.key);
		}
	}

	function handleFocus(): void {
		focused = true;
	}

	function handleBlur(): void {
		focused = false;
	}
</script>

{#if months.length > 0}
	{#if variant === 'page'}
		<nav
			class="scrubber fixed top-1/2 right-0 -translate-y-1/2 z-(--z-page)"
			aria-label={$_('timeline_scrubber_label')}
		>
			{@render railBody()}
		</nav>
	{:else}
		<nav
			class="wall__axis absolute inset-y-2 end-1 z-(--z-page) w-6"
			data-testid="panel-axis"
			aria-label={$_('timeline_scrubber_label')}
		>
			{@render railBody()}
		</nav>
	{/if}
{/if}

{#snippet railBody()}
	<!-- The rail is ONE control (role=scrollbar, one tab stop): the
	     press, the drag, the hover and the keyboard all land here;
	     the marks inside are decoration and never eat an event. -->
	<div
		bind:this={rail}
		class="scrubber__rail relative touch-none select-none cursor-row-resize focus-visible:outline-none {variant ===
		'page'
			? 'w-[72px]'
			: 'h-full w-6'}"
		style:height={variant === 'page' ? `${railHeight}px` : undefined}
		role="scrollbar"
		aria-orientation="vertical"
		aria-label={$_('timeline_scrubber_label')}
		aria-controls={controls}
		tabindex="0"
		aria-valuemin="0"
		aria-valuemax="100"
		aria-valuenow={Math.round(clamp01(progress) * 100)}
		aria-valuetext={activeLabel}
		onpointerdown={handlePointerDown}
		onpointermove={handlePointerMove}
		onpointerup={handlePointerUp}
		onpointercancel={handlePointerCancel}
		onpointerleave={handlePointerLeave}
		onkeydown={handleKeydown}
		onfocus={handleFocus}
		onblur={handleBlur}
	>
		<!-- The axis spine the years and dots ride on. -->
		<span
			class="absolute top-1 bottom-1 {variant === 'page' ? 'right-[11.5px]' : 'right-[10.5px]'} w-px bg-border"
			aria-hidden="true"
		></span>
		<!-- The years rest on the rail permanently (faint, one per
		     year run). The compact panel slot has no room for them --
		     the dots carry the months there. -->
		{#if variant === 'page'}
			{#each marks as mark (mark.key)}
				{#if mark.year !== null}
					<span
						class="scrubber__year pointer-events-none absolute right-[24px] font-body font-normal text-[10px] leading-[16px] text-text-faint tabular-nums"
						style="top: {mark.y - 8}px"
						aria-hidden="true">{mark.year}</span
					>
				{/if}
			{/each}
		{/if}
		<!-- The month dots: one per month, at the fraction of the
		     landing the wall reported (the dot you see IS the place a
		     press takes you). -->
		{#each marks as mark (mark.key)}
			<span
				class="scrubber__stop pointer-events-none absolute {variant === 'page'
					? 'right-[6px] w-[12px]'
					: 'right-[6px] w-[10px]'} flex h-[16px] items-center justify-center"
				style="top: {mark.y - 8}px"
			>
				<span
					class="scrubber__dot h-[5px] w-[5px] rounded-full transition-transform duration-(--motion-fast) ease-standard {mark.key ===
					activeKey
						? 'scale-125 bg-accent'
						: 'bg-border-strong'}"
					aria-hidden="true"
				></span>
			</span>
		{/each}
		<!-- The position indicator: the wall's scroll progress as the
		     solid accent bar crossing the spine -- the rail's resting
		     answer to "where am I" (it stands down mid-drag: the tick
		     is the position marker then). -->
		<span
			class="scrubber__indicator absolute right-0 w-6 h-[2px] rounded-full bg-accent transition-opacity duration-(--motion-fast) ease-standard"
			style="top: {PAD + clamp01(progress) * span - 1}px; opacity: {dragging ? 0 : 1}"
			aria-hidden="true"
		></span>
		<!-- The pointer's tick: the same length, the same edge, the
		     same thickness as the indicator, at 70% strength -- the bar
		     says where the reading is, the tick says where a press
		     would land. It shows for the hover and the drag only (the
		     keyboard has no pointer; a plain scroll has no aim). -->
		<span
			class="scrubber__tick pointer-events-none absolute right-0 w-6 h-[2px] rounded-full bg-accent/70 transition-opacity duration-(--motion-fast) ease-standard"
			style="top: {pointerY - 1}px; opacity: {hovering || dragging ? 1 : 0}"
			aria-hidden="true"
		></span>
		<!-- The readout bubble: the month's label with its right edge
		     meeting the tick/indicator's left end, concentric with it
		     (the label is this mark's name, not a floating card). -->
		<span
			class="scrubber__readout pointer-events-none absolute right-[24px] -translate-y-1/2 max-w-[120px] truncate whitespace-nowrap rounded-control border border-border bg-card px-2 py-1 font-body font-semibold text-[12px] leading-[1.4] text-text shadow-card transition-opacity duration-(--motion-fast) ease-standard"
			style="top: {bubbleTop}px; opacity: {bubbleVisible ? 1 : 0}"
			aria-hidden="true">{bubbleLabel}</span
		>
	</div>
{/snippet}
