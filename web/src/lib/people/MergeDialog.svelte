<script lang="ts">
	// The batch merge dialog of the unnamed region: pick the target
	// the selection folds into -- one of the selected clusters (then
	// the sources are the OTHERS, so the contract's two-source
	// minimum needs a third selection, and the target may be NAMED
	// inline first: the PUT label runs before the merge) or any
	// existing person (then the sources are the whole selection).
	// The page owns the writes: `onsubmit` receives the target choice
	// and runs the PUT-then-POST pair; the dialog owns the target
	// list's lifecycle (the existing people fetch on open) and the
	// choice's shape.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import { listPeople, type PersonSummary, type UnnamedClusterSummary } from '$lib/api/people';
	import { resolveErrorText } from '$lib/i18n/errors';

	type MergeTarget = {
		clusterId: string;
		/** A name to PUT on the target before the merge (selected targets only). */
		inlineName?: string;
	};

	type Props = {
		/** Open state; bind it from the caller, the modal closes itself. */
		open?: boolean;
		/** The selected clusters the merge folds. */
		selected: readonly UnnamedClusterSummary[];
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Run the merge with the chosen target. */
		onsubmit: (target: MergeTarget) => void;
	};

	let { open = $bindable(false), selected, busy = false, errorKey = '', onsubmit }: Props = $props();

	// The existing people (the merge targets outside the selection),
	// fetched every open: the list is small and the freshest answer
	// beats a cached one.
	let people = $state<PersonSummary[]>([]);
	let loadingPeople = $state(false);
	let loadErrorKey = $state('');

	let targetId = $state('');
	let inlineName = $state('');

	const selectedIds = $derived(selected.map((cluster) => cluster.clusterId));
	// A selected cluster may host the merge only when the selection
	// holds at least three: the other selected ones are the sources,
	// and this dialog's batch shape keeps at least two of them (the
	// one-source pairwise fold is the cluster detail's "merge into"
	// path, not this batch bar).
	const offerSelected = $derived(selected.length >= 3);
	const optionIds = $derived([
		...(offerSelected ? selectedIds : []),
		...people.map((entry) => entry.clusterId)
	]);
	const targetAmongSelected = $derived(offerSelected && selectedIds.includes(targetId));

	$effect(() => {
		if (open) {
			targetId = '';
			inlineName = '';
			loadErrorKey = '';
			people = [];
			loadingPeople = true;
			void (async () => {
				const outcome = await listPeople();
				if (!open) {
					// Closed mid-flight: the next open refetches.
					return;
				}
				loadingPeople = false;
				if (!outcome.ok) {
					loadErrorKey = outcome.errorKey;
					return;
				}
				people = outcome.people;
			})();
		}
	});

	// Re-pick the default target when the option set changes: the
	// dialog re-opened (targetId reset) or the people list arrived.
	$effect(() => {
		void optionIds;
		if (!open) {
			return;
		}
		if (!optionIds.includes(targetId)) {
			targetId = optionIds[0] ?? '';
		}
	});

	function submit(): void {
		if (targetId === '') {
			return;
		}
		const trimmed = inlineName.trim();
		onsubmit({
			clusterId: targetId,
			inlineName: targetAmongSelected && trimmed !== '' ? trimmed : undefined
		});
	}
</script>

<Modal bind:open title={$_('people_merge_dialog_title')}>
	<div class="merge-dialog flex flex-col gap-3">
		<p class="merge-dialog__body m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('people_merge_body')}</p>
		{#if loadingPeople}
			<p class="merge-dialog__loading m-0 font-body font-normal text-[14px] leading-[1.4] text-text-soft animate-loading-in" role="status">{$_('people_loading')}</p>
		{:else if loadErrorKey !== ''}
			<p class="merge-dialog__load-error m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, loadErrorKey)}</p>
		{:else if optionIds.length === 0}
			<!-- No target exists for this selection: two clusters with no
			     named person can only merge once a third cluster (or an
			     existing person) gives the request its target. -->
			<p class="merge-dialog__empty m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">{$_('people_merge_no_target')}</p>
		{:else}
			<label class="merge-dialog__label font-body font-semibold text-[14px] leading-[1.4] text-text" for="merge-dialog-target">{$_('people_merge_target_label')}</label>
			<select
				class="merge-dialog__target w-full px-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-card border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
				id="merge-dialog-target"
				bind:value={targetId}
			>
				{#if offerSelected}
					<optgroup label={$_('people_merge_group_selected')}>
						{#each selected as cluster (cluster.clusterId)}
							<option value={cluster.clusterId}>{$_('people_merge_cluster_option', { values: { count: cluster.faceCount } })}</option>
						{/each}
					</optgroup>
				{/if}
				{#if people.length > 0}
					<optgroup label={$_('people_merge_group_people')}>
						{#each people as entry (entry.clusterId)}
							<option value={entry.clusterId}>{entry.name}</option>
						{/each}
					</optgroup>
				{/if}
			</select>
			{#if targetAmongSelected}
				<!-- The inline naming of a selected target: the page PUTs
				     this name before the merge, so the merged person keeps
				     a name (a source's name dies; the target's persists). -->
				<Input label={$_('people_merge_inline_name_label')} name="merged-name" bind:value={inlineName} maxlength={128} autocomplete="off" />
			{/if}
		{/if}
		{#if errorKey !== ''}
			<p class="merge-dialog__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text" role="alert">{resolveErrorText($_, errorKey)}</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>{$_('common_cancel')}</Button>
		<Button disabled={busy || optionIds.length === 0} onclick={submit}>{$_('people_merge_action')}</Button>
	{/snippet}
</Modal>
