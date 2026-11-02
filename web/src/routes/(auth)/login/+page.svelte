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

	// An unknown error code (a proxy answer, a drifted contract) has
	// no i18n key; svelte-i18n echoes the key itself, which would leak
	// code jargon into the UI -- such codes fall back to the generic
	// message.
	function errorText(key: string): string {
		const text = $_(key);
		return text === key ? $_('internal') : text;
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
	<main class="login">
		<!-- The wrapper owns the width: the Card's own element belongs to
		     another component, so styling it from here is scoped out. -->
		<div class="login__card">
			<Card>
				<h1 class="login__title">{$_('login_title')}</h1>
				<p class="login__description">{$_('login_description')}</p>
				<form class="login__form" onsubmit={handleSubmit}>
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
						<p class="login__error" role="alert">{errorText(errorKey)}</p>
					{/if}
					{#if retryAfterSeconds !== null && countdownRemaining > 0}
						<!-- The hint's lifetime is retryAfterSeconds (the response's
						     authoritative rate-limit state): a resubmit nulls it, so a
						     credential-failure outcome can never leave a frozen
						     countdown next to its own message. -->
						<p class="login__hint">{$_('login_retry_in', { values: { seconds: countdownRemaining } })}</p>
					{/if}
					<div class="login__actions">
						<Button type="submit" disabled={submitting}>{$_('login_submit')}</Button>
					</div>
				</form>
			</Card>
		</div>
		<!-- Placeholder notes for the mobile pairing entry (the web IA's
		     bottom note); the QR flow is a later milestone's scope. -->
		<p class="login__note">{$_('login_pairing_qr_note')}</p>
		<p class="login__note">{$_('login_pairing_url_note')}</p>
	</main>
{/if}

<style>
	.login {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		padding: 12vh 16px 48px;
	}

	.login__card {
		width: min(400px, 100%);
	}

	.login__title {
		margin: 0 0 4px;
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 26px;
		letter-spacing: 0.01em;
		color: var(--color-text);
	}

	.login__description {
		margin: 0 0 20px;
		font: 400 15px/1.5 var(--font-body);
		color: var(--color-text-soft);
	}

	.login__form {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.login__error {
		margin: 0;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-danger-text);
	}

	.login__hint {
		margin: 0;
		font: 400 14px/1.4 var(--font-body);
		color: var(--color-text-soft);
		font-variant-numeric: tabular-nums;
	}

	.login__actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 4px;
	}

	.login__note {
		max-width: min(400px, 100%);
		margin: 0;
		font: 400 13px/1.5 var(--font-body);
		color: var(--color-text-faint);
		text-align: center;
	}
</style>
