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
	<h1>{$_('albums_title')}</h1>
	<p>{$_('albums_coming_later')}</p>
</section>
