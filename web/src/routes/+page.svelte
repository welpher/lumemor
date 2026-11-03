<script lang="ts">
	import { _ } from 'svelte-i18n';
</script>

<svelte:head>
	<title>lumemor</title>
</svelte:head>

<section>
	<h1>{$_('welcome_title')}</h1>
	<p>{$_('welcome_body')}</p>
</section>
