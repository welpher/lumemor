<script lang="ts">
	// The user management page of the admin surface (M2d Task 8,
	// reshaped by the 2026-11-09 modal pin): the account listing
	// (role and disabled badges), the create-user flow, and the
	// disable/enable toggle. CREATION IS A MODAL now (web IA §7.8:
	// the page-header row's right slot carries the "New user" button
	// -- UserPlus + text, the primary-flow shape -- and the Modal
	// holds the same fields the inline form held, the same failure
	// surface, success closing it with the listing converging; the
	// grammar joins the album and tag dialogs and this page's own
	// disable confirm). The toggle adopts the PATCH response as the
	// row's fresh state and confirms before a disable, because the
	// write signs the account out of every session at once. A failed
	// disable keeps the confirm dialog open with the mapped error
	// (the trash page's confirm pattern); a 404 is the already-gone
	// reading -- the listing's refetch reconciles the row and the
	// dialog closes.
	//
	// THE 403 PAGE: a signed-in non-admin landing on this route sees
	// the mapped auth_forbidden message -- the tool shell has no
	// caller-role knowledge of its own (the session contract carries
	// none), so this page's listing is the probe and its mapped
	// error is the answer.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import AdminPageHeader from '$lib/components/AdminPageHeader.svelte';
	import { createAdminUser, listAdminUsers, setAdminUserDisabled, type AdminUser } from '$lib/api/adminUsers';
	import { resolveErrorText } from '$lib/i18n/errors';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import UserX from '@lucide/svelte/icons/user-x';
	import Check from '@lucide/svelte/icons/check';
	import X from '@lucide/svelte/icons/x';
	import UserPlus from '@lucide/svelte/icons/user-plus';

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

	// The create modal's state: the open flag plus the form's
	// fields; the role select defaults to 'user'. Opening the modal
	// seeds the fields clean (a close mid-draft leaves the stale
	// text behind, but the next open overwrites it).
	let createOpen = $state(false);
	let formName = $state('');
	let formEmail = $state('');
	let formPassword = $state('');
	let formRole = $state<'user' | 'admin'>('user');
	let creating = $state(false);
	let createErrorKey = $state('');

	/** Open the creation modal with clean fields. */
	function openCreate(): void {
		if (creating) {
			return;
		}
		formName = '';
		formEmail = '';
		formPassword = '';
		formRole = 'user';
		createErrorKey = '';
		createOpen = true;
	}

	/** Close the creation modal (the busy write guards it). */
	function cancelCreate(): void {
		if (creating) {
			return;
		}
		createOpen = false;
		createErrorKey = '';
	}

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
		return new Intl.DateTimeFormat(activeLocale, { dateStyle: 'medium' }).format(new Date(user.createdAt));
	}

	async function handleSubmit(event?: SubmitEvent): Promise<void> {
		event?.preventDefault();
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
			// row, no refetch needed. Success closes the modal (the
			// 2026-11-09 pin); a failure keeps it open with the
			// mapped error inside -- the disable confirm's pattern.
			users = [...users, outcome.user];
			createOpen = false;
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
		// A retry starts clean: the modal's error from the previous
		// attempt clears before the request runs (the trash page's
		// confirm pattern), so a stale message cannot sit beside the
		// in-flight state of the next one.
		disableErrorKey = '';
		// The modal stays open through the request (the trash page's
		// confirm pattern): a failed disable recovers in place, with
		// the mapped error inside the dialog -- closing first would
		// park the failure under the table, away from the account it
		// names. Success (and the already-gone 404 reconciliation)
		// closes it below.
		const failureKey = await toggle(user, true);
		if (failureKey !== '') {
			disableErrorKey = failureKey;
			return;
		}
		pendingDisable = null;
	}

	async function enable(user: AdminUser): Promise<void> {
		// The enable is a direct row action (no confirm dialog), so
		// its mapped failure lands under the table.
		const failureKey = await toggle(user, false);
		if (failureKey !== '') {
			actionErrorKey = failureKey;
		}
	}

	/** Run one toggle; answer the failure's error key ('' when the write landed). */
	async function toggle(user: AdminUser, isDisabled: boolean): Promise<string> {
		if (busy) {
			return '';
		}
		busy = true;
		actionErrorKey = '';
		const outcome = await setAdminUserDisabled(user.id, isDisabled);
		busy = false;
		if (outcome.ok) {
			// The response is the account after the write: adopt it.
			users = users.map((entry) => (entry.id === user.id ? outcome.user : entry));
			return '';
		}
		if (outcome.errorKey === 'user_not_found') {
			// The row is stale (the account left the instance through
			// another session): reconcile the whole listing. The row
			// is already gone from the listing's answer, so this is
			// the already-gone success path, not a failure to show.
			retry();
			return '';
		}
		// The self-disable guard's 400 lands here too: the web client
		// cannot know its own user id, so the server's refusal and
		// its mapped message are the guard's visible half.
		return outcome.errorKey;
	}
</script>

<svelte:head>
	<title>Users · lumemor</title>
</svelte:head>

<div class="users flex flex-col gap-6">
	{#if status === 'loading'}
		<p class="users__status my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('admin_users_loading')}
		</p>
	{:else if status === 'error'}
		<div
			class="users__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={retry}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{:else if status === 'ready'}
		<!-- The page-header row (the 2026-11-09 specimen): the title
		     on the left, this page's own contextual action on the
		     right -- the create-user button, UserPlus + text (the
		     primary-flow shape). The listing is a full-width page:
		     the rows eat the width the rail leaves. -->
		<AdminPageHeader title={$_('admin_users_title')}>
			{#snippet actions()}
				<Button onclick={openCreate}>
					<UserPlus size={20} aria-hidden="true" />
					{$_('admin_users_new')}
				</Button>
			{/snippet}
		</AdminPageHeader>
		<section class="users__list" aria-label={$_('admin_users_title')}>
			<table
				class="users__table w-full border-collapse bg-card border border-border rounded-card shadow-card overflow-hidden"
			>
				<thead>
					<tr>
						<th
							scope="col"
							class="px-3.5 py-2.5 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-left text-text-faint border-b border-border"
							>{$_('admin_users_name')}</th
						>
						<th
							scope="col"
							class="px-3.5 py-2.5 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-left text-text-faint border-b border-border"
							>{$_('admin_users_email')}</th
						>
						<th
							scope="col"
							class="px-3.5 py-2.5 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-left text-text-faint border-b border-border"
							>{$_('admin_users_role')}</th
						>
						<th
							scope="col"
							class="px-3.5 py-2.5 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-left text-text-faint border-b border-border"
							>{$_('admin_users_status')}</th
						>
						<th
							scope="col"
							class="px-3.5 py-2.5 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-left text-text-faint border-b border-border"
							>{$_('admin_users_created')}</th
						>
						<th
							scope="col"
							class="px-3.5 py-2.5 font-body font-semibold text-[12px] leading-[1.4] tracking-[0.04em] uppercase text-left text-text-faint border-b border-border"
						>
							<!-- The actions column's heading is for screen readers; the
							     buttons' own labels name the actions. -->
							<span
								class="users__actions-heading absolute w-px h-px overflow-hidden [clip-path:inset(50%)] whitespace-nowrap"
								>{$_('admin_users_actions')}</span
							>
						</th>
					</tr>
				</thead>
				<tbody>
					{#each users as user (user.id)}
						<!-- The group class feeds the last-row border removal
						     (group-last) on the cells below. -->
						<tr class="group" class:users__row--disabled={user.isDisabled}>
							<td
								class="px-3.5 py-2.5 font-body font-semibold text-[14px] leading-[1.5] text-text border-b border-border group-last:border-b-0"
								>{user.name}</td
							>
							<td
								class="px-3.5 py-2.5 font-body font-normal text-[14px] leading-[1.5] text-text-soft border-b border-border group-last:border-b-0"
								>{user.email}</td
							>
							<td
								class="px-3.5 py-2.5 font-body font-normal text-[14px] leading-[1.5] text-text border-b border-border group-last:border-b-0"
							>
								<span
									class="users__badge inline-block px-2.5 py-0.5 font-body font-semibold text-[12px] leading-[1.6] bg-accent-soft rounded-control"
									class:users__badge--admin={user.role === 'admin'}
									class:text-accent-soft-text={user.role === 'admin'}
									class:text-text-soft={user.role !== 'admin'}
								>
									{user.role === 'admin' ? $_('admin_role_admin') : $_('admin_role_user')}
								</span>
							</td>
							<td
								class="px-3.5 py-2.5 font-body font-normal text-[14px] leading-[1.5] text-text border-b border-border group-last:border-b-0"
							>
								<span
									class="users__badge inline-block px-2.5 py-0.5 font-body font-semibold text-[12px] leading-[1.6] rounded-control {user.isDisabled
										? 'text-danger-text bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)]'
										: 'text-text-soft bg-accent-soft'}"
									class:users__badge--disabled={user.isDisabled}
								>
									{user.isDisabled ? $_('admin_user_disabled') : $_('admin_user_active')}
								</span>
							</td>
							<td
								class="px-3.5 py-2.5 font-body font-normal text-[14px] leading-[1.5] text-text-soft border-b border-border group-last:border-b-0"
								>{createdLabel(user)}</td
							>
							<td
								class="users__actions px-3.5 py-2.5 text-right font-body font-normal text-[14px] leading-[1.5] text-text border-b border-border group-last:border-b-0"
							>
								{#if user.isDisabled}
									<Button variant="ghost" disabled={busy} onclick={() => void enable(user)}>
										<Check size={20} aria-hidden="true" />
										{$_('admin_users_enable')}
									</Button>
								{:else}
									<Button variant="danger" disabled={busy} onclick={() => askDisable(user)}>
										<UserX size={20} aria-hidden="true" />
										{$_('admin_users_disable')}
									</Button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
			{#if actionErrorKey !== ''}
				<p class="users__error font-body font-medium text-[14px] leading-[1.4] text-danger-text" role="alert">
					{errorText(actionErrorKey)}
				</p>
			{/if}
		</section>
	{/if}
</div>

<!-- The creation modal (the 2026-11-09 pin): the same fields the
     inline form held, the same failure surface (one mapped error
     line), success closing it and the listing converging. The busy
     write keeps it open; the cancel is inert while one runs. -->
<Modal bind:open={createOpen} title={$_('admin_users_create_heading')} onclose={cancelCreate}>
	<p class="users__create-note mb-4 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
		{$_('admin_users_create_note')}
	</p>
	<form id="admin-user-create" class="users__form flex flex-col gap-3.5" onsubmit={handleSubmit}>
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
		<div class="users__field flex flex-col gap-1.5">
			<label
				class="users__label font-body font-semibold text-[14px] leading-[1.4] text-text"
				for="admin-user-role">{$_('admin_users_field_role')}</label
			>
			<select
				class="users__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
				id="admin-user-role"
				bind:value={formRole}
			>
				<option value="user">{$_('admin_role_user')}</option>
				<option value="admin">{$_('admin_role_admin')}</option>
			</select>
		</div>
		{#if createErrorKey !== ''}
			<p class="users__error m-0 font-body font-medium text-[14px] leading-[1.4] text-danger-text" role="alert">
				{errorText(createErrorKey)}
			</p>
		{/if}
	</form>
	{#snippet footer()}
		<Button variant="ghost" disabled={creating} onclick={cancelCreate}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button disabled={creating} onclick={() => void handleSubmit()}>
			<UserPlus size={20} aria-hidden="true" />
			{$_('admin_users_create_submit')}
		</Button>
	{/snippet}
</Modal>

<Modal open={pendingDisable !== null} title={$_('admin_users_disable_title')} onclose={cancelDisable}>
	<p class="users__confirm-body font-body font-normal text-[15px] leading-[1.5] text-text">
		{$_('admin_users_disable_body', { values: { name: pendingDisable?.name ?? '' } })}
	</p>
	{#if disableErrorKey !== ''}
		<p
			class="users__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(disableErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={cancelDisable}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="danger" disabled={busy} onclick={() => void confirmDisable()}>
			<UserX size={20} aria-hidden="true" />
			{$_('admin_users_disable_confirm')}
		</Button>
	{/snippet}
</Modal>

<style>
	/* Residual style (verbatim): a disabled row's state class reaches
	 * its table cells. */
	.users__row--disabled td {
		color: var(--color-text-faint);
	}
</style>
