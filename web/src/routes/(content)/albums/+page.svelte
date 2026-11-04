<script lang="ts">
	// The albums placeholder shell: the album domain (curated and
	// shared albums, virtual albums) is M3 scope; this route pins the
	// IA slot with an honest empty state so the drawer's link never
	// dead-ends.
	import { _ } from 'svelte-i18n';
</script>

<svelte:head>
	<title>{$_('albums_title')} · lumemor</title>
</svelte:head>

<section>
	<!-- The utilities preserve the previous user-agent rendering of
	     this placeholder page across Tailwind's preflight reset: the
	     h1's 2em (32px) size and the p's 1em block margins. -->
	<h1 class="text-[32px]">{$_('albums_title')}</h1>
	<p class="my-4">{$_('albums_coming_later')}</p>
</section>
