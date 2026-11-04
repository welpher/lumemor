<script lang="ts">
	import { _ } from 'svelte-i18n';
</script>

<svelte:head>
	<title>lumemor</title>
</svelte:head>

<section>
	<!-- The utilities preserve the previous user-agent rendering of
	     this unstyled page across Tailwind's preflight reset: the h1's
	     2em (32px) size and the p's 1em block margins. -->
	<h1 class="text-[32px]">{$_('welcome_title')}</h1>
	<p class="my-4">{$_('welcome_body')}</p>
</section>
