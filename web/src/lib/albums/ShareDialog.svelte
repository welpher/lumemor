<script lang="ts">
	// The share management dialog of the album detail (web IA §5's
	// share flow): the owner's grantee list -- one row per current
	// share with its revoke action -- over the member pick from the
	// instance's minimal user list. The page owns the writes (the
	// grant POST and the revoke DELETE, each followed by the detail
	// refetch whose fresh `shares` array is this list) and the
	// candidate list (the users minus the current grantees minus the
	// caller's own account -- the identity seam's self filter, so the
	// grant's own target rule never has to refuse a self-pick); the
	// dialog owns the choice's lifecycle (reseeded on open, defaulted
	// to the first candidate).
	//
	// Disabled accounts render as unselectable options carrying the
	// honest note -- the entry's own switch, server-provided state.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import type { AlbumShareEntry, UserSummary } from '$lib/api/albums';
	import X from '@lucide/svelte/icons/x';
	import Share2 from '@lucide/svelte/icons/share-2';

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The owner's grantee list, the detail's `shares` array. */
		shares: readonly AlbumShareEntry[];
		/** The grant candidates: every account minus the current grantees. */
		candidates: readonly UserSummary[];
		/** The page is still fetching the member list. */
		loading?: boolean;
		/** Writes run one at a time: the actions disable while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Grant the view ACL to the picked account. */
		ongrant: (userId: string) => void;
		/** Revoke one grantee's view ACL. */
		onrevoke: (userId: string) => void;
	};

	let { open = $bindable(false), shares, candidates, loading = false, busy = false, errorKey = '', ongrant, onrevoke }: Props = $props();

	let pickedId = $state('');

	$effect(() => {
		if (open) {
			pickedId = '';
		}
	});

	// Re-pick the default candidate when the list changes: the dialog
	// re-opened (pickedId reset) or the page's list arrived.
	$effect(() => {
		void candidates;
		if (!open) {
			return;
		}
		if (!candidates.some((candidate) => candidate.id === pickedId)) {
			const first = candidates.find((candidate) => !candidate.isDisabled);
			pickedId = first?.id ?? candidates[0]?.id ?? '';
		}
	});

	/** Locale-aware grant date of one grantee row. */
	function dateLabel(grantedAt: string): string {
		return new Intl.DateTimeFormat($locale ?? 'en', { dateStyle: 'medium' }).format(
			new Date(grantedAt)
		);
	}

	function grant(): void {
		if (pickedId === '') {
			return;
		}
		ongrant(pickedId);
	}
</script>

<Modal bind:open title={$_('albums_share_title')}>
	<div class="share-dialog flex flex-col gap-3">
		<p class="share-dialog__body m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('albums_share_body')}</p>

		<section class="share-dialog__members flex flex-col gap-1.5">
			{#if shares.length === 0}
				<p class="share-dialog__empty m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('albums_share_empty')}</p>
			{:else}
				<ul class="share-dialog__list flex flex-col gap-1.5 m-0 p-0 list-none">
					{#each shares as share (share.userId)}
						<li class="share-dialog__row flex flex-wrap items-center gap-2 py-1.5 px-3 bg-card border border-border rounded-control">
							<span class="share-dialog__email font-body font-semibold text-[14px] leading-[1.4] text-text truncate">{share.email}</span>
							<span class="share-dialog__since font-body font-normal text-[13px] leading-[1.4] text-text-soft">{$_('albums_share_member_since', { values: { date: dateLabel(share.grantedAt) } })}</span>
							<!-- The row's revoke action is icon-only (the icon-first
							     rule), named by its aria-label. -->
							<IconButton
								class="share-dialog__revoke ml-auto"
								variant="ghost"
								disabled={busy}
								aria-label={$_('albums_share_revoke')}
								title={$_('albums_share_revoke')}
								onclick={() => onrevoke(share.userId)}
							>
								<X size={20} aria-hidden="true" />
							</IconButton>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="share-dialog__pick flex flex-col gap-1.5">
			{#if loading}
				<p class="share-dialog__loading m-0 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in" role="status">{$_('albums_share_pick_label')}…</p>
			{:else if candidates.length === 0}
				<p class="share-dialog__no-candidates m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('albums_share_no_candidates')}</p>
			{:else}
				<label class="share-dialog__label font-body font-semibold text-[14px] leading-[1.4] text-text" for="share-dialog-target">{$_('albums_share_pick_label')}</label>
				<select
					class="share-dialog__target w-full px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
					id="share-dialog-target"
					bind:value={pickedId}
				>
					{#each candidates as candidate (candidate.id)}
						<option value={candidate.id} disabled={candidate.isDisabled}>
							{candidate.isDisabled
								? `${candidate.email} -- ${$_('albums_share_disabled_note')}`
								: candidate.email}
						</option>
					{/each}
				</select>
			{/if}
		</section>

		{#if errorKey !== ''}
			<p class="share-dialog__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>
			<X size={20} aria-hidden="true" />
			{$_('common_close')}
		</Button>
		<Button disabled={busy || loading || pickedId === ''} onclick={grant}>
			<Share2 size={20} aria-hidden="true" />
			{$_('albums_share_submit')}
		</Button>
	{/snippet}
</Modal>
