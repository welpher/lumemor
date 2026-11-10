<script lang="ts">
	// The offline strip: a
	// thin yellow band pinned ABOVE the topbar content of both shells
	// (content and tool -- they share the events lib, so they share
	// its connection truth), telling the user the SSE stream is down
	// and the browser is reconnecting. It rises on the connection
	// store's 'down' state, falls on recovery ('open') and on the
	// intentional close ('closed' -- no stream means nothing is
	// lost), and counts the ELAPSED seconds since the drop, ticking
	// once a second: the browser's EventSource owns the retry
	// schedule and never exposes it to JavaScript, so a countdown to
	// the next attempt would be fabricated precision -- the elapsed
	// count is the honest surface for the reconnect countdown.
	//
	// THE SURFACE: the Sun notice fill with Navy ink (the brand's
	// yellow pair, AA by construction: dark-on-light 8:1), the
	// offline glyph leading the line (decorative, 16px -- a status
	// band, not an action affordance; the icon-first rule still
	// gives it its glyph), and role="status" so the drop and the
	// recovery announce themselves to assistive tech. The band rides
	// the frame's top edge: the shells mount it as the first child
	// of their h-dvh column, above the topbar (and above <main> on
	// the album detail, whose own chrome hides the shell bar -- the
	// connection is a shell-wide concern, not the topbar's).
	import { _ } from 'svelte-i18n';
	import { streamDownSince, streamStatus } from './connection.svelte';
	import WifiOff from '@lucide/svelte/icons/wifi-off';

	const down = $derived(streamStatus() === 'down');

	// The live clock: one tick per second while the strip stands, so
	// the elapsed count keeps moving; the effect tears the interval
	// down on recovery and never runs while the stream is healthy.
	let now = $state(Date.now());
	$effect(() => {
		if (!down) {
			return;
		}
		const timer = setInterval(() => {
			now = Date.now();
		}, 1000);
		return () => clearInterval(timer);
	});

	const seconds = $derived.by(() => {
		const since = streamDownSince();
		return since === null ? 0 : Math.max(0, Math.floor((now - since) / 1000));
	});
</script>

{#if down}
	<div
		class="offline-strip flex items-center justify-center gap-2 w-full px-4 py-1 bg-notice text-navy-900 border-b border-[color-mix(in_srgb,var(--color-navy-900)_18%,transparent)]"
		role="status"
	>
		<WifiOff size={16} class="shrink-0" aria-hidden="true" />
		<span class="offline-strip__text font-body font-semibold text-[13px] leading-[1.4]"
			>{$_('offline_strip', { values: { seconds } })}</span
		>
	</div>
{/if}
