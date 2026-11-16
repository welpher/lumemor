<script lang="ts">
	// The user management page of the admin surface (the deprovision
	// update): the account listing (role and disabled badges), the
	// create-user flow, the disable/enable toggle, and the two-step
	// account disposal. CREATION IS A MODAL now:
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
	// THE DISPOSAL is two steps, never one red dialog: the first names
	// where the departing account's photos go (a transfer to another
	// member, with the member picker, or the trash, with the instance's
	// retention window spelled out -- read from the trash settings when
	// the dialog opens, so the copy never promises a window the
	// instance does not keep), and the second asks for the irreversible
	// confirmation. The delete affordance itself is a TEXT button in
	// the danger tone: an everyday row is one misclick away from a
	// destructive one, and a solid red fill next to Disable would make
	// the two indistinguishable at a glance. Every refusal stays in the
	// second dialog with the mapped message -- the last administrator,
	// the caller's own account and a stale row are the server's
	// answers, never a disabled button pretending they are impossible.
	//
	// THE GEOMETRY is the admin content column's: the 760 column, the
	// page-header row, and the list inside its own card shell, so a
	// narrow viewport compresses the rows instead of pushing the page
	// sideways. THE ROWS BREATHE (E41): each account is one list row --
	// the avatar, the stacked identity (the name bold with the role
	// and status chips inline, the email soft under it), the joined
	// date hidden on narrow (its WRAPPER span carries the hidden, the
	// E37 review's N7 lesson), and the actions right. Every confirm
	// the table carried survives the reshape unchanged.
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
	import { toast } from '$lib/components/toast.svelte';
	import {
		createAdminUser,
		deleteAdminUser,
		listAdminUsers,
		setAdminUserDisabled,
		type AdminUser,
		type AdminUserDisposition
	} from '$lib/api/adminUsers';
	import { getTrashSettings } from '$lib/api/adminSettings';
	import { resolveErrorText } from '$lib/i18n/errors';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import UserX from '@lucide/svelte/icons/user-x';
	import UserMinus from '@lucide/svelte/icons/user-minus';
	import Check from '@lucide/svelte/icons/check';
	import X from '@lucide/svelte/icons/x';
	import UserPlus from '@lucide/svelte/icons/user-plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';
	/** The disposal's two steps: where the photos go, then the confirmation. */
	type DeleteStage = 'disposition' | 'confirm';

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

	// The disposal's state: the account, the step, the chosen arm and
	// its receiver, the in-flight flag and the dialog's error line.
	let pendingDelete = $state<AdminUser | null>(null);
	let deleteStage = $state<DeleteStage>('disposition');
	let deleteDisposition = $state<'transfer' | 'trash'>('transfer');
	let deleteTransferId = $state('');
	let deleting = $state(false);
	let deleteErrorKey = $state('');
	/**
	 * The instance's trash retention, read when the disposal dialog
	 * first opens: the trash arm's copy names the window the settings
	 * actually keep. Null until the read lands (or when it fails --
	 * the neutral sentence stands in, and the next open retries).
	 */
	let retentionDays = $state<number | null>(null);

	/** The accounts the disposal may hand the content to (never the departing one). */
	const otherUsers = $derived(users.filter((entry) => entry.id !== pendingDelete?.id));

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

	/** The avatar's initial: the name's first grapheme. */
	function initialOf(user: AdminUser): string {
		return Array.from(user.name)[0] ?? '?';
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
			// row, no refetch needed. Success closes the modal; a failure keeps it open with the
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

	/**
	 * Open the disposal's first step for one account, seeding the arm
	 * and its receiver. The retention read is lazy: the copy needs the
	 * instance's window, and a users page nobody deletes from pays
	 * nothing for it.
	 */
	function openDelete(user: AdminUser): void {
		if (deleting || busy) {
			return;
		}
		deleteErrorKey = '';
		deleteStage = 'disposition';
		deleteDisposition = 'transfer';
		pendingDelete = user;
		const first = users.find((entry) => entry.id !== user.id);
		deleteTransferId = first?.id ?? '';
		void loadRetention();
	}

	/** Read the trash retention once (the next open retries a failure). */
	async function loadRetention(): Promise<void> {
		if (retentionDays !== null) {
			return;
		}
		const outcome = await getTrashSettings();
		if (!outcome.ok) {
			return;
		}
		retentionDays = outcome.settings.retentionDays;
	}

	/** Leave the disposal entirely (both steps). */
	function cancelDelete(): void {
		if (deleting) {
			return;
		}
		pendingDelete = null;
		deleteStage = 'disposition';
		deleteErrorKey = '';
	}

	/** The first step's forward move: to the irreversible confirmation. */
	function continueDelete(): void {
		if (deleteDisposition === 'transfer' && deleteTransferId === '') {
			return;
		}
		deleteErrorKey = '';
		deleteStage = 'confirm';
	}

	/** The disposal's write: the chosen arm lands, the row leaves. */
	async function confirmDelete(): Promise<void> {
		const user = pendingDelete;
		if (user === null || deleting) {
			return;
		}
		deleteErrorKey = '';
		deleting = true;
		const disposition: AdminUserDisposition =
			deleteDisposition === 'trash' ? { dispose: 'trash' } : { transferTo: deleteTransferId };
		const outcome = await deleteAdminUser(user.id, disposition);
		deleting = false;
		if (outcome.ok) {
			// The account is gone: the listing converges without a
			// refetch (the response carries no body), and the danger
			// toast carries the acknowledgment past the dialog's close.
			users = users.filter((entry) => entry.id !== user.id);
			pendingDelete = null;
			deleteStage = 'disposition';
			toast($_('user_delete_done'), 'danger');
			return;
		}
		if (outcome.errorKey === 'user_not_found') {
			// The row is stale (another administrator removed the
			// account meanwhile): the listing's refetch is the
			// already-gone reconciliation, not a failure to show.
			pendingDelete = null;
			deleteStage = 'disposition';
			retry();
			return;
		}
		// Every other refusal stays where the decision was made: the
		// dialog keeps the account's name and the mapped reason (the
		// last administrator, the caller's own account, a payload
		// the server rejected), and the row is untouched.
		deleteErrorKey = outcome.errorKey;
	}

	/** The trash arm's copy: the instance's window, or the neutral sentence. */
	function trashNote(): string {
		if (retentionDays === null) {
			return $_('user_delete_trash_note_unknown');
		}
		return $_('user_delete_trash_note', { values: { days: retentionDays } });
	}
</script>

<svelte:head>
	<title>Users · lumemor</title>
</svelte:head>

<div class="users flex flex-col gap-6 max-w-[760px] mx-auto">
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
		<!-- The page-header row: the title
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
		<section class="users__list-section" aria-label={$_('admin_users_title')}>
			<!-- The list's card shell: the rounded card the rows live in.
			     The rows breathe instead of a six-column shell squeezing
			     (E41): the identity is stacked (the name bold with the
			     role/status chips inline, the email soft under it), the
			     joined date hides on narrow, and the actions sit right. -->
			<div class="users__list-shell relative rounded-card border border-border shadow-card">
				<ul
					class="users__list flex flex-col bg-card overflow-hidden rounded-card border border-border m-0 p-0 list-none"
				>
					{#each users as user (user.id)}
						<li
							class="users__row flex items-center gap-4 px-4 py-3 border-b border-border last:border-b-0"
							class:users__row--disabled={user.isDisabled}
							data-user-email={user.email}
						>
							<!-- The avatar: the name's initial on the soft accent
							     ground. Decorative (the name speaks beside it). -->
							<span
								class="users__avatar w-10 h-10 shrink-0 rounded-full bg-accent-soft text-accent-soft-text flex items-center justify-center font-bold text-[14px]"
								aria-hidden="true"
							>
								{initialOf(user)}
							</span>
							<div class="users__identity min-w-0 flex-1">
								<div class="users__identity-top flex flex-wrap items-center gap-2">
									<span
										class="users__name font-body font-semibold text-[15px] leading-[1.4] text-text"
										>{user.name}</span
									>
									<span
										class="users__badge users__badge--role inline-block px-2 py-0.5 font-body font-semibold text-[12px] leading-[1.6] rounded-control {user.role ===
										'admin'
											? 'bg-accent-soft text-accent-soft-text'
											: 'border border-border text-text-soft'}"
									>
										{user.role === 'admin' ? $_('admin_role_admin') : $_('admin_role_user')}
									</span>
									<span
										class="users__badge users__badge--status inline-block px-2 py-0.5 font-body font-semibold text-[12px] leading-[1.6] rounded-control {user.isDisabled
											? 'bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)] text-danger-text'
											: 'border border-border text-text-soft'}"
									>
										{user.isDisabled ? $_('admin_user_disabled') : $_('admin_user_active')}
									</span>
								</div>
								<div
									class="users__email truncate font-body font-normal text-[13px] leading-[1.5] text-text-soft"
								>
									{user.email}
								</div>
							</div>
							<!-- The joined date hides on narrow. The WRAPPER carries
							     the hidden (the E37 review's N7 lesson): a component's
							     base display class must never be in a position to eat
							     a caller's `hidden`, so the wrapper is a plain span
							     with no display rules of its own. -->
							<span
								class="users__joined hidden sm:inline font-body font-normal text-[13px] leading-[1.4] text-text-faint whitespace-nowrap tabular-nums"
							>
								{$_('admin_users_joined', { values: { date: createdLabel(user) } })}
							</span>
							<div class="users__actions flex items-center gap-1.5">
								{#if user.isDisabled}
									<Button
										variant="ghost"
										disabled={busy || deleting}
										onclick={() => void enable(user)}
									>
										<Check size={20} aria-hidden="true" />
										{$_('admin_users_enable')}
									</Button>
								{:else}
									<Button
										variant="danger"
										disabled={busy || deleting}
										onclick={() => askDisable(user)}
									>
										<UserX size={20} aria-hidden="true" />
										{$_('admin_users_disable')}
									</Button>
								{/if}
								<!-- The disposal entry: a text button in the danger
							     tone beside an everyday Disable -- a solid fill
							     would make the two read alike at a glance. -->
								<Button
									variant="danger-text"
									disabled={busy || deleting}
									aria-label={$_('user_delete_action', { values: { name: user.name } })}
									onclick={() => openDelete(user)}
								>
									<UserMinus size={20} aria-hidden="true" />
									{$_('user_delete')}
								</Button>
							</div>
						</li>
					{/each}
				</ul>
			</div>
			{#if actionErrorKey !== ''}
				<p
					class="users__error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
					role="alert"
				>
					{errorText(actionErrorKey)}
				</p>
			{/if}
		</section>
	{/if}
</div>

<!-- The creation modal: the same fields the
     inline form held, the same failure surface (one mapped error
     line), success closing it and the listing converging. The busy
     write keeps it open; the cancel is inert while one runs. -->
<Modal bind:open={createOpen} title={$_('admin_users_create_heading')} onclose={cancelCreate}>
	<p class="users__create-note mb-4 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
		{$_('admin_users_create_note')}
	</p>
	<form id="admin-user-create" class="users__form flex flex-col gap-3.5" onsubmit={handleSubmit}>
		<!-- The native pre-flight mirrors the creation body: the email and
		     password are required, the password spans the contract's 8-128
		     runes (the wizard's own numbers), and the optional display name
		     stops where the account rules stop at 100. There is no edit arm
		     to bound here -- the only account write this page carries is the
		     create and the disable switch. -->
		<Input
			id="admin-user-name"
			label={$_('admin_users_field_name')}
			type="text"
			autocomplete="name"
			maxlength={100}
			bind:value={formName}
		/>
		<Input
			id="admin-user-email"
			label={$_('admin_users_field_email')}
			type="email"
			autocomplete="off"
			required
			bind:value={formEmail}
		/>
		<Input
			id="admin-user-password"
			label={$_('admin_users_field_password')}
			type="password"
			autocomplete="new-password"
			required
			minlength={8}
			maxlength={128}
			bind:value={formPassword}
		/>
		<div class="users__field flex flex-col gap-1.5">
			<label
				class="users__label font-body font-semibold text-[14px] leading-[1.4] text-text"
				for="admin-user-role">{$_('admin_users_field_role')}</label
			>
			<select
				class="users__select px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
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
		<!-- A real submit control: the footer sits outside the form
		     (the modal's own structure), so the button binds to it by
		     id -- the browser's validation runs on the click path and
		     Enter in a field submits the form. -->
		<Button type="submit" form="admin-user-create" disabled={creating}>
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

<!-- The disposal, step one: where the departing account's photos go.
     The title names the account, and the two arms are the whole
     decision -- a transfer to another member (the picker names who),
     or the trash (whose subline spells out the instance's own
     retention window, because "the trash" means nothing without it). -->
<Modal
	open={pendingDelete !== null && deleteStage === 'disposition'}
	title={$_('user_delete_title', { values: { name: pendingDelete?.name ?? '' } })}
	onclose={cancelDelete}
>
	<p class="users__delete-body m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
		{$_('user_delete_body')}
	</p>
	<div class="users__delete-arms mt-3 flex flex-col gap-2">
		<label
			class="users__delete-arm flex items-start gap-2 px-3 py-2.5 bg-page border border-border rounded-control cursor-pointer"
		>
			<input
				class="mt-0.5 w-[16px] h-[16px] accent-accent"
				type="radio"
				name="user-delete-disposition"
				value="transfer"
				bind:group={deleteDisposition}
			/>
			<span class="flex flex-col">
				<span class="font-body font-semibold text-[14px] leading-[1.4] text-text"
					>{$_('user_delete_transfer')}</span
				>
				<span class="font-body font-normal text-[12px] leading-[1.5] text-text-faint"
					>{$_('user_delete_transfer_note')}</span
				>
			</span>
		</label>
		<label
			class="users__delete-arm flex items-start gap-2 px-3 py-2.5 bg-page border border-border rounded-control cursor-pointer"
		>
			<input
				class="mt-0.5 w-[16px] h-[16px] accent-accent"
				type="radio"
				name="user-delete-disposition"
				value="trash"
				bind:group={deleteDisposition}
			/>
			<span class="flex flex-col">
				<span class="font-body font-semibold text-[14px] leading-[1.4] text-text"
					>{$_('user_delete_trash')}</span
				>
				<span class="users__delete-retention font-body font-normal text-[12px] leading-[1.5] text-text-faint"
					>{trashNote()}</span
				>
			</span>
		</label>
	</div>
	{#if deleteDisposition === 'transfer'}
		<div class="users__delete-target mt-3 flex flex-col gap-1.5">
			<label class="font-body font-semibold text-[13px] leading-[1.4] text-text-soft" for="user-delete-transfer"
				>{$_('user_delete_transfer_to')}</label
			>
			<select
				class="users__select px-3 py-[9px] pe-9 font-body font-normal text-[14px] leading-[1.4] text-text bg-card border border-border-strong rounded-control"
				id="user-delete-transfer"
				bind:value={deleteTransferId}
			>
				{#each otherUsers as entry (entry.id)}
					<option value={entry.id}>{entry.name}</option>
				{/each}
			</select>
		</div>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" onclick={cancelDelete}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button
			variant="danger-text"
			disabled={deleteDisposition === 'transfer' && deleteTransferId === ''}
			onclick={continueDelete}
		>
			{$_('user_delete_continue')}
		</Button>
	{/snippet}
</Modal>

<!-- The disposal, step two: the irreversible confirmation. The name
     and the chosen disposition ride along, the bold line says what the
     click means, and every refusal stays HERE with its mapped message
     -- the dialog closes only when the account is gone. -->
<Modal
	open={pendingDelete !== null && deleteStage === 'confirm'}
	title={$_('user_delete_confirm_title')}
	maxWidth="440px"
	onclose={cancelDelete}
>
	<p class="users__delete-confirm m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
		{$_('user_delete_confirm_body', { values: { name: pendingDelete?.name ?? '' } })}
		<b class="font-semibold text-text">{$_('user_delete_irreversible')}</b>
	</p>
	{#if deleteErrorKey !== ''}
		<p
			class="users__confirm-error mt-3 font-body font-medium text-[14px] leading-[1.4] text-danger-text"
			role="alert"
		>
			{errorText(deleteErrorKey)}
		</p>
	{/if}
	{#snippet footer()}
		<Button variant="ghost" disabled={deleting} onclick={cancelDelete}>
			<X size={20} aria-hidden="true" />
			{$_('common_cancel')}
		</Button>
		<Button variant="danger-text" disabled={deleting} onclick={() => void confirmDelete()}>
			<Trash2 size={20} aria-hidden="true" />
			{$_('user_delete_confirm')}
		</Button>
	{/snippet}
</Modal>

<style>
	/* Residual style (verbatim in spirit): a disabled account's row
	 * reads quieter -- the identity and the joined date dim, the
	 * actions keep their colors (they still work). */
	.users__row--disabled .users__identity,
	.users__row--disabled .users__joined {
		opacity: 0.6;
	}
</style>
