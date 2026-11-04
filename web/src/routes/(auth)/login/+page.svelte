<script lang="ts">
	// The sign-in page of the (auth) shell: a labeled email/password
	// form whose failures map onto the shared i18n error table (the
	// error code lowercased), a rate-limit countdown hint driven by
	// the 429's Retry-After, a disabled submit while a request is
	// in flight, and the redirectTo round-trip on success. The bottom
	// notes are placeholders for the mobile pairing entry (a later
	// milestone's scope, per the web IA).
	import { goto } from '$app/navigation';
	import { _ } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import Input from '$lib/components/Input.svelte';
	import { login } from '$lib/auth/session.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let email = $state('');
	let password = $state('');
	let submitting = $state(false);
	let errorKey = $state('');
	let retryAfterSeconds = $state<number | null>(null);
	let countdownRemaining = $state(0);

	// A fresh instance has no account to sign in with: hand the
	// browser over to the first-run wizard. /setup is a
	// server-rendered page outside the SPA (the SPA has no /setup
	// route on purpose), so this must be a full document navigation
	// rather than the client-side router. The form stays hidden
	// while the navigation is in flight.
	$effect(() => {
		if (data.setupRequired) {
			window.location.replace('/setup');
		}
	});

	// The rate-limit countdown hint: one-second ticks from the
	// response's Retry-After down to zero, then the hint clears.
	$effect(() => {
		const seconds = retryAfterSeconds;
		if (seconds === null) {
			return;
		}
		countdownRemaining = seconds;
		const timer = setInterval(() => {
			countdownRemaining -= 1;
			if (countdownRemaining <= 0) {
				clearInterval(timer);
				retryAfterSeconds = null;
			}
		}, 1000);
		return () => clearInterval(timer);
	});

	// The shared unknown-code fallback (see $lib/i18n/errors): an
	// unknown error code has no i18n key, and svelte-i18n would echo
	// the key itself -- such codes render the generic message. The
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	async function handleSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (submitting) {
			return;
		}
		submitting = true;
		errorKey = '';
		retryAfterSeconds = null;
		const outcome = await login(email, password);
		submitting = false;
		if (outcome.ok) {
			goto(data.redirectTo);
			return;
		}
		errorKey = outcome.errorKey;
		retryAfterSeconds = outcome.retryAfterSeconds;
	}
</script>

<svelte:head>
	<title>{$_('login_title')} · lumemor</title>
</svelte:head>

{#if !data.setupRequired}
	<main class="login flex flex-col items-center gap-3 pt-[12vh] px-4 pb-12">
		<!-- The wrapper owns the width: the Card's own element belongs to
		     another component, so styling it from here is scoped out. -->
		<div class="login__card w-[min(400px,100%)]">
			<Card>
				<h1 class="login__title mb-1 font-display font-bold text-[26px] tracking-[0.01em] text-text">{$_('login_title')}</h1>
				<p class="login__description mb-5 font-body font-normal text-[15px] leading-[1.5] text-text-soft">{$_('login_description')}</p>
				<form class="login__form flex flex-col gap-3.5" onsubmit={handleSubmit}>
					<Input
						id="login-email"
						label={$_('login_email')}
						type="email"
						autocomplete="username"
						bind:value={email}
					/>
					<Input
						id="login-password"
						label={$_('login_password')}
						type="password"
						autocomplete="current-password"
						bind:value={password}
					/>
					{#if errorKey !== ''}
						<p class="login__error font-body font-medium text-[14px] leading-[1.4] text-danger-text" role="alert">{errorText(errorKey)}</p>
					{/if}
					{#if retryAfterSeconds !== null && countdownRemaining > 0}
						<!-- The hint's lifetime is retryAfterSeconds (the response's
						     authoritative rate-limit state): a resubmit nulls it, so a
						     credential-failure outcome can never leave a frozen
						     countdown next to its own message. -->
						<p class="login__hint font-body font-normal text-[14px] leading-[1.4] text-text-soft tabular-nums">{$_('login_retry_in', { values: { seconds: countdownRemaining } })}</p>
					{/if}
					<div class="login__actions flex justify-end mt-1">
						<Button type="submit" disabled={submitting}>{$_('login_submit')}</Button>
					</div>
				</form>
			</Card>
		</div>
		<!-- Placeholder notes for the mobile pairing entry (the web IA's
		     bottom note); the QR flow is a later milestone's scope. -->
		<p class="login__note max-w-[min(400px,100%)] font-body font-normal text-[13px] leading-[1.5] text-text-faint text-center">{$_('login_pairing_qr_note')}</p>
		<p class="login__note max-w-[min(400px,100%)] font-body font-normal text-[13px] leading-[1.5] text-text-faint text-center">{$_('login_pairing_url_note')}</p>
	</main>
{/if}
