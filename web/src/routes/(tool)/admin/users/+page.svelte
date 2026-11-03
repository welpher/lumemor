<script lang="ts">
	// The user management page of the admin surface (M2d Task 8):
	// the account listing (role and disabled badges), the create-user
	// form, and the disable/enable toggle. The create form mirrors
	// the login form's failure surface (one mapped error line); the
	// toggle adopts the PATCH response as the row's fresh state and
	// confirms before a disable, because the write signs the account
	// out of every session at once.
	//
	// THE 403 PAGE: a signed-in non-admin landing on this route sees
	// the mapped auth_forbidden message -- the tool shell has no
	// caller-role knowledge of its own (the session contract carries
	// none), so this page's listing is the probe and its mapped
	// error is the answer.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import Input from '$lib/components/Input.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import {
		createAdminUser,
		listAdminUsers,
		setAdminUserDisabled,
		type AdminUser
	} from '$lib/api/adminUsers';
	import { resolveErrorText } from '$lib/i18n/errors';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	let users = $state<AdminUser[]>([]);
	let status = $state<ListStatus>('idle');
	let errorKey = $state('');
	let actionErrorKey = $state('');
	let busy = $state(false);
	/** Bumping re-runs the listing (the retry button). */
	let retryTick = $state(0);
	/** The account whose disable the confirm Modal names. */
	let pendingDisable = $state<AdminUser | null>(null);
	let disableErrorKey = $state('');

	// The create form's fields; the role select defaults to 'user'.
	let formName = $state('');
	let formEmail = $state('');
	let formPassword = $state('');
	let formRole = $state<'user' | 'admin'>('user');
	let creating = $state(false);
	let createErrorKey = $state('');

	const activeLocale = $derived($locale ?? 'en');

	// The listing, keyed on the retry tick. A superseded run -- the
	// retry fired again, or the route left -- never applies.
	$effect(() => {
		void retryTick;
		status = 'loading';
		errorKey = '';
		let current = true;
		void (async () => {
			const outcome = await listAdminUsers();
			if (!current) {
				return;
			}
			if (!outcome.ok) {
				status = 'error';
				errorKey = outcome.errorKey;
				return;
			}
			users = outcome.users;
			status = 'ready';
		})();
		return () => {
			current = false;
		};
	});

	function retry(): void {
		retryTick += 1;
	}

	// The shared unknown-code fallback (see $lib/i18n/errors); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	/** Locale-aware creation date of one account. */
	function createdLabel(user: AdminUser): string {
		return new Intl.DateTimeFormat(activeLocale, { dateStyle: 'medium' }).format(
			new Date(user.createdAt)
		);
	}

	async function handleSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (creating) {
			return;
		}
		creating = true;
		createErrorKey = '';
		const outcome = await createAdminUser({
			name: formName,
			email: formEmail,
			password: formPassword,
			role: formRole
		});
		creating = false;
		if (outcome.ok) {
			// The listing is creation-ordered (oldest first), so the
			// new account joins the end; the response is the fresh
			// row, no refetch needed.
			users = [...users, outcome.user];
			formName = '';
			formEmail = '';
			formPassword = '';
			formRole = 'user';
			return;
		}
		createErrorKey = outcome.errorKey;
	}

	function askDisable(user: AdminUser): void {
		disableErrorKey = '';
		pendingDisable = user;
	}

	function cancelDisable(): void {
		if (busy) {
			return;
		}
		pendingDisable = null;
		disableErrorKey = '';
	}

	async function confirmDisable(): Promise<void> {
		const user = pendingDisable;
		if (user === null || busy) {
			return;
		}
		pendingDisable = null;
		await toggle(user, true);
	}

	async function enable(user: AdminUser): Promise<void> {
		await toggle(user, false);
	}

	async function toggle(user: AdminUser, isDisabled: boolean): Promise<void> {
		if (busy) {
			return;
		}
		busy = true;
		actionErrorKey = '';
		const outcome = await setAdminUserDisabled(user.id, isDisabled);
		busy = false;
		if (outcome.ok) {
			// The response is the account after the write: adopt it.
			users = users.map((entry) => (entry.id === user.id ? outcome.user : entry));
			return;
		}
		if (outcome.errorKey === 'user_not_found') {
			// The row is stale (the account left the instance through
			// another session): reconcile the whole listing.
			retry();
			return;
		}
		// The self-disable guard's 400 lands here too: the web client
		// cannot know its own user id, so the server's refusal and
		// its mapped message are the guard's visible half.
		actionErrorKey = outcome.errorKey;
	}
</script>

<svelte:head>
	<title>Users · lumemor</title>
</svelte:head>

<div class="users">
	{#if status === 'loading'}
		<p class="users__status" role="status">{$_('admin_users_loading')}</p>
	{:else if status === 'error'}
		<div class="users__failure" role="alert">
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}>{$_('action_retry')}</Button>
		</div>
	{:else if status === 'ready'}
		<section class="users__list" aria-label={$_('admin_users_title')}>
			<h1 class="users__title">{$_('admin_users_title')}</h1>
			<table class="users__table">
				<thead>
					<tr>
						<th scope="col">{$_('admin_users_name')}</th>
						<th scope="col">{$_('admin_users_email')}</th>
						<th scope="col">{$_('admin_users_role')}</th>
						<th scope="col">{$_('admin_users_status')}</th>
						<th scope="col">{$_('admin_users_created')}</th>
						<th scope="col"><span class="users__actions-heading">{$_('admin_users_actions')}</span></th>
					</tr>
				</thead>
				<tbody>
					{#each users as user (user.id)}
						<tr class:users__row--disabled={user.isDisabled}>
							<td class="users__name">{user.name}</td>
							<td class="users__email">{user.email}</td>
							<td>
								<span class="users__badge" class:users__badge--admin={user.role === 'admin'}>
									{user.role === 'admin' ? $_('admin_role_admin') : $_('admin_role_user')}
								</span>
							</td>
							<td>
								<span class="users__badge" class:users__badge--disabled={user.isDisabled}>
									{user.isDisabled ? $_('admin_user_disabled') : $_('admin_user_active')}
								</span>
							</td>
							<td class="users__created">{createdLabel(user)}</td>
							<td class="users__actions">
								{#if user.isDisabled}
									<Button variant="ghost" disabled={busy} onclick={() => void enable(user)}>
										{$_('admin_users_enable')}
									</Button>
								{:else}
									<Button variant="danger" disabled={busy} onclick={() => askDisable(user)}>
										{$_('admin_users_disable')}
									</Button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
			{#if actionErrorKey !== ''}
				<p class="users__error" role="alert">{errorText(actionErrorKey)}</p>
			{/if}
		</section>

		<section class="users__create">
			<div class="users__create-card">
				<Card>
					<h2 class="users__create-title">{$_('admin_users_create_heading')}</h2>
					<p class="users__create-note">{$_('admin_users_create_note')}</p>
					<form class="users__form" onsubmit={handleSubmit}>
						<Input
							id="admin-user-name"
							label={$_('admin_users_field_name')}
							type="text"
							autocomplete="name"
							bind:value={formName}
						/>
						<Input
							id="admin-user-email"
							label={$_('admin_users_field_email')}
							type="email"
							autocomplete="off"
							bind:value={formEmail}
						/>
						<Input
							id="admin-user-password"
							label={$_('admin_users_field_password')}
							type="password"
							autocomplete="new-password"
							bind:value={formPassword}
						/>
						<div class="users__field">
							<label class="users__label" for="admin-user-role">{$_('admin_users_field_role')}</label>
							<select class="users__select" id="admin-user-role" bind:value={formRole}>
								<option value="user">{$_('admin_role_user')}</option>
								<option value="admin">{$_('admin_role_admin')}</option>
							</select>
						</div>
						{#if createErrorKey !== ''}
							<p class="users__error" role="alert">{errorText(createErrorKey)}</p>
						{/if}
						<div class="users__form-actions">
							<Button type="submit" disabled={creating}>{$_('admin_users_create_submit')}</Button>
						</div>
					</form>
				</Card>
			</div>
		</section>
	{/if}
</div>

<Modal open={pendingDisable !== null} title={$_('admin_users_disable_title')} onclose={cancelDisable}>
	{#snippet children()}
		<p class="users__confirm-body">
			{$_('admin_users_disable_body', { values: { name: pendingDisable?.name ?? '' } })}
		</p>
		{#if disableErrorKey !== ''}
			<p class="users__confirm-error" role="alert">{errorText(disableErrorKey)}</p>
		{/if}
	{/snippet}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelDisable}>{$_('common_cancel')}</Button>
		<Button variant="danger" disabled={busy} onclick={() => void confirmDisable()}>
			{$_('admin_users_disable_confirm')}
		</Button>
	{/snippet}
</Modal>

<style>
	.users {
		max-width: 920px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.users__status {
		margin: 48px 0;
		text-align: center;
		color: var(--color-text-soft);
	}

	.users__failure {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		max-width: 420px;
		margin: 48px auto;
		padding: 20px;
		text-align: center;
		border: 1px solid var(--color-danger);
		border-radius: var(--radius-card);
		color: var(--color-danger-text);
	}

	.users__failure p {
		margin: 0;
	}

	.users__title {
		margin: 0 0 12px;
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 26px;
		letter-spacing: 0.01em;
		color: var(--color-text);
	}

	.users__table {
		width: 100%;
		border-collapse: collapse;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
		overflow: hidden;
	}

	.users__table th {
		padding: 10px 14px;
		font: 600 12px/1.4 var(--font-body);
		letter-spacing: 0.04em;
		text-transform: uppercase;
		text-align: left;
		color: var(--color-text-faint);
		border-bottom: 1px solid var(--color-border);
	}

	.users__table td {
		padding: 10px 14px;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text);
		border-bottom: 1px solid var(--color-border);
	}

	.users__table tbody tr:last-child td {
		border-bottom: none;
	}

	.users__row--disabled td {
		color: var(--color-text-faint);
	}

	.users__name {
		font-weight: 600;
	}

	.users__email,
	.users__created {
		color: var(--color-text-soft);
	}

	.users__actions-heading {
		/* The actions column's heading is for screen readers; the
		 * buttons' own labels name the actions. */
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}

	.users__badge {
		display: inline-block;
		padding: 2px 10px;
		font: 600 12px/1.6 var(--font-body);
		color: var(--color-text-soft);
		background-color: var(--color-accent-soft);
		border-radius: var(--radius-control);
	}

	.users__badge--admin {
		color: var(--color-accent-soft-text);
	}

	.users__badge--disabled {
		color: var(--color-danger-text);
		background-color: color-mix(in srgb, var(--color-danger) 12%, transparent);
	}

	.users__actions {
		text-align: right;
	}

	.users__create-card {
		width: min(400px, 100%);
	}

	.users__create-title {
		margin: 0 0 4px;
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 22px;
		letter-spacing: 0.01em;
		color: var(--color-text);
	}

	.users__create-note {
		margin: 0 0 20px;
		font: 400 14px/1.5 var(--font-body);
		color: var(--color-text-soft);
	}

	.users__form {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.users__field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.users__label {
		font: 600 14px/1.4 var(--font-body);
		color: var(--color-text);
	}

	.users__select {
		padding: 9px 12px;
		font: 400 15px/1.4 var(--font-body);
		color: var(--color-text);
		background-color: var(--color-card);
		border: 1px solid var(--color-border-strong);
		border-radius: var(--radius-control);
		transition: border-color var(--motion-fast) var(--ease-standard);
	}

	.users__select:focus {
		border-color: var(--color-accent);
	}

	.users__error {
		margin: 0;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-danger-text);
	}

	.users__form-actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 4px;
	}

	.users__confirm-body {
		margin: 0;
		font: 400 15px/1.5 var(--font-body);
		color: var(--color-text);
	}

	.users__confirm-error {
		margin: 12px 0 0;
		font: 500 14px/1.4 var(--font-body);
		color: var(--color-danger-text);
	}
</style>
