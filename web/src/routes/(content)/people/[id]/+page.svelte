<script lang="ts">
	// The cluster detail of the people surface (web IA §3): the
	// caller's label state over the cluster's caller-visible faces,
	// strongest first. The unnamed state (label null) renders the
	// unnamed header and the naming CTA; a label renders the name
	// with the rename, hidden-toggle and merge-into actions.
	//
	// THE PINNED CHOICES:
	//
	// - The label and hidden writes adopt their RESPONSES (the PUT
	//   body's answer IS the fresh label state -- a rename keeps the
	//   hidden switch), no refetch; the SSE self-events converge
	//   anyway.
	// - Hiding rides on the label row: an unlabeled cluster answers
	//   400 with the message key `person_hide_requires_label`, which
	//   surfaces as the honest message line right beside the naming
	//   CTA -- the nudge to name first, never a silent failure.
	// - The merge-into fold posts THIS cluster as the single source
	//   and leaves for /people (this cluster -- and its name -- dies
	//   with the merge; the target's label persists).
	// - The single-face move posts /faces/{id}/move and converges
	//   through a refetch: the moved face left, the counts and the
	//   ordering are the server's to recompute.
	// - SSE: the person events that touch THIS cluster (a label or
	//   hidden change on it, a merge that dissolves or survives as
	//   it, a face move onto or out of it) refetch the detail; the
	//   refetched answer owns the truth -- a foreign label event
	//   changes nothing here (B4: labels are private per user).
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import {
		fetchPersonCluster,
		labelPerson,
		listUnnamedPeople,
		listPeople,
		mergePeople,
		moveFace,
		setPersonHidden,
		type ClusterDetail,
		type PersonSummary,
		type UnnamedClusterSummary
	} from '$lib/api/people';
	import { subscribePersonEvents, type PersonEvent } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import Button from '$lib/components/Button.svelte';
	import PersonNameDialog from '$lib/people/PersonNameDialog.svelte';
	import ClusterPickerDialog from '$lib/people/ClusterPickerDialog.svelte';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import EyeOff from '@lucide/svelte/icons/eye-off';
	import Eye from '@lucide/svelte/icons/eye';
	import GitMerge from '@lucide/svelte/icons/git-merge';

	const clusterId = $derived(page.params.id ?? '');

	type DetailStatus = 'idle' | 'loading' | 'ready' | 'error';

	let detail = $state<ClusterDetail | null>(null);
	let status = $state<DetailStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	// One write at a time across every action (the dialogs and the
	// header toggles share the flag).
	let writeBusy = $state(false);

	// The naming / rename dialog.
	let nameDialogOpen = $state(false);
	let nameErrorKey = $state('');

	// The hidden toggle's honest failure (the requires-label message
	// and its kin) -- the header owns the line.
	let hideErrorKey = $state('');

	// The merge-into dialog: the existing people (minus this cluster)
	// as the target list, fetched on open.
	let mergeIntoOpen = $state(false);
	let mergeIntoErrorKey = $state('');
	let mergeIntoPeople = $state<PersonSummary[]>([]);
	let mergeIntoLoading = $state(false);

	// The single-face move dialog: the face being moved (null while
	// closed), with the named people and the unnamed region's first
	// page (minus this cluster) as the target list, fetched on open.
	let moveFaceId = $state<string | null>(null);
	let moveErrorKey = $state('');
	let movePeople = $state<PersonSummary[]>([]);
	let moveUnnamed = $state<UnnamedClusterSummary[]>([]);
	let moveLoading = $state(false);

	const firstLoad = $derived(detail === null && status === 'loading');
	const hidden = $derived(detail?.label?.isHidden ?? false);

	// The fetch and the SSE subscription are keyed on the route id;
	// a superseded run (the id changed, or the route left, while the
	// fetch was in flight) never applies.
	let fetchCurrent = true;

	/** Apply one detail outcome; the superseded runs never call this. */
	function apply(outcome: Awaited<ReturnType<typeof fetchPersonCluster>>): void {
		if (outcome.ok) {
			detail = outcome.detail;
			errorKey = '';
			status = 'ready';
			return;
		}
		detail = null;
		status = 'error';
		errorKey = outcome.errorKey;
	}

	/**
	 * The convergence refetch: keep the rendered detail while the
	 * fresh answer is in flight (a flash of the loading line would
	 * flicker every SSE event), replace it when it lands.
	 */
	async function convergeDetail(): Promise<void> {
		const id = untrack(() => clusterId);
		if (id === '' || !fetchCurrent) {
			return;
		}
		const outcome = await fetchPersonCluster(id);
		if (fetchCurrent) {
			apply(outcome);
		}
	}

	// Whether one person event concerns this cluster: the label and
	// hidden writes on it, a merge that dissolves it (this cluster
	// was a source) or survives it (it absorbed the sources), a face
	// move that joins it or leaves it.
	function touchesThisCluster(event: PersonEvent): boolean {
		const id = untrack(() => clusterId);
		switch (event.name) {
			case 'person.labeled':
			case 'person.unlabeled':
			case 'person.hidden':
				return event.clusterId === id;
			case 'person.merged':
				return event.survivingClusterId === id || event.mergedClusterIds.includes(id);
			case 'person.face_moved':
				return event.clusterId === id || (detail?.faces.some((face) => face.id === event.faceId) ?? false);
		}
	}

	$effect(() => {
		const id = clusterId;
		if (id === '') {
			return;
		}
		fetchCurrent = true;
		status = 'loading';
		detail = null;
		errorKey = '';
		void (async () => {
			const outcome = await fetchPersonCluster(id);
			if (fetchCurrent) {
				apply(outcome);
			}
		})();

		const unsubscribe = subscribePersonEvents((event) => {
			if (touchesThisCluster(event)) {
				void convergeDetail();
			}
		});
		return () => {
			fetchCurrent = false;
			unsubscribe();
		};
	});

	// The naming / rename write: the response IS the fresh label
	// state, so the header adopts it (a rename keeps the hidden
	// switch -- the response carries it).
	async function submitName(name: string): Promise<void> {
		if (writeBusy || detail === null) {
			return;
		}
		writeBusy = true;
		nameErrorKey = '';
		const outcome = await labelPerson(clusterId, name);
		writeBusy = false;
		if (!outcome.ok) {
			nameErrorKey = outcome.errorKey;
			return;
		}
		detail = {
			...detail,
			label: { name: outcome.label.name, isHidden: outcome.label.isHidden }
		};
		nameDialogOpen = false;
	}

	// The hidden toggle: the response is the label state after the
	// flip (badge and notice follow it). An unlabeled cluster answers
	// the message key the line renders honestly -- the naming CTA
	// beside it is the way out.
	async function toggleHidden(): Promise<void> {
		if (writeBusy || detail === null) {
			return;
		}
		writeBusy = true;
		hideErrorKey = '';
		const outcome = await setPersonHidden(clusterId, !hidden);
		writeBusy = false;
		if (!outcome.ok) {
			hideErrorKey = outcome.errorKey;
			return;
		}
		detail = {
			...detail,
			label: { name: outcome.label.name, isHidden: outcome.label.isHidden }
		};
	}

	// The merge-into fold: this cluster is the single source, so it
	// dissolves (with its name -- the honest cost); the target's
	// label persists. Success leaves for the named region.
	async function runMergeInto(targetId: string): Promise<void> {
		if (writeBusy) {
			return;
		}
		writeBusy = true;
		mergeIntoErrorKey = '';
		const outcome = await mergePeople([clusterId], targetId);
		writeBusy = false;
		if (!outcome.ok) {
			mergeIntoErrorKey = outcome.errorKey;
			return;
		}
		mergeIntoOpen = false;
		void goto('/people');
	}

	// The single-face move: the face joins the picked cluster; the
	// face list converges through the refetch (the moved face left,
	// the counts are the server's).
	async function runMove(targetId: string): Promise<void> {
		if (writeBusy || moveFaceId === null) {
			return;
		}
		writeBusy = true;
		moveErrorKey = '';
		const faceId = moveFaceId;
		const outcome = await moveFace(faceId, targetId);
		writeBusy = false;
		if (!outcome.ok) {
			moveErrorKey = outcome.errorKey;
			return;
		}
		moveFaceId = null;
		void convergeDetail();
	}

	// Fetch the merge-into target list on open: the caller's named
	// people, this cluster excluded (a target among the sources is
	// no such merge).
	$effect(() => {
		if (!mergeIntoOpen) {
			return;
		}
		mergeIntoPeople = [];
		mergeIntoErrorKey = '';
		mergeIntoLoading = true;
		void (async () => {
			const outcome = await listPeople();
			if (!mergeIntoOpen) {
				return;
			}
			mergeIntoLoading = false;
			if (outcome.ok) {
				mergeIntoPeople = outcome.people;
			}
		})();
	});

	// Fetch the move target list on open: the named people and the
	// unnamed region's first page, this cluster excluded (moving onto
	// the face's own cluster is an accepted no-op -- never offered).
	$effect(() => {
		if (moveFaceId === null) {
			return;
		}
		movePeople = [];
		moveUnnamed = [];
		moveErrorKey = '';
		moveLoading = true;
		void (async () => {
			const [peopleOutcome, unnamedOutcome] = await Promise.all([listPeople(), listUnnamedPeople(null)]);
			if (moveFaceId === null) {
				return;
			}
			moveLoading = false;
			if (peopleOutcome.ok) {
				movePeople = peopleOutcome.people;
			}
			if (unnamedOutcome.ok) {
				moveUnnamed = unnamedOutcome.page.items;
			}
		})();
	});

	/** Locale-aware capture date of one face's asset. */
	function dateLabel(takenAt: string): string {
		return new Intl.DateTimeFormat($locale ?? 'en', { dateStyle: 'medium' }).format(new Date(takenAt));
	}

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	const mergeIntoGroups = $derived([
		{
			label: $_('people_merge_group_people'),
			options: mergeIntoPeople
				.filter((entry) => entry.clusterId !== clusterId)
				.map((entry) => ({ id: entry.clusterId, label: entry.name }))
		}
	]);
	const moveGroups = $derived([
		{
			label: $_('people_merge_group_people'),
			options: movePeople
				.filter((entry) => entry.clusterId !== clusterId)
				.map((entry) => ({ id: entry.clusterId, label: entry.name }))
		},
		{
			label: $_('people_move_group_unnamed'),
			options: moveUnnamed
				.filter((cluster) => cluster.clusterId !== clusterId)
				.map((cluster) => ({
					id: cluster.clusterId,
					label: $_('people_merge_cluster_option', { values: { count: cluster.faceCount } })
				}))
		}
	]);

	const titleText = $derived(detail?.label?.name ?? $_('people_detail_unnamed'));
</script>

<svelte:head>
	<title>{titleText} · lumemor</title>
</svelte:head>

<section class="person-detail max-w-[1200px] mx-auto p-6" aria-busy={status === 'loading'}>
	<div class="person-detail__top flex items-center gap-2 mb-1">
		<a
			class="person-detail__back inline-flex items-center px-2.5 py-1.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft no-underline rounded-control transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/people"
		>
			<svg viewBox="0 0 24 24" width="16" height="16" fill="none" aria-hidden="true">
				<path
					d="M14.5 6.5L8 12l6.5 5.5"
					stroke="currentColor"
					stroke-width="1.75"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
			{$_('nav_people')}</a
		>
	</div>

	{#if firstLoad}
		<p class="person-detail__status my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('people_loading')}
		</p>
	{:else if status === 'error'}
		<div
			class="person-detail__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void convergeDetail()}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{:else if detail !== null}
		<header class="person-detail__header flex flex-wrap items-center gap-3 mb-3">
			<h1 class="person-detail__title m-0 font-display font-bold text-[24px] leading-[1.3] text-text">
				{titleText}
			</h1>
			{#if hidden}
				<span
					class="person-detail__badge inline-flex items-center px-1.5 py-0.5 font-body font-semibold text-[12px] leading-[1.4] text-text-faint border border-border rounded-control whitespace-nowrap"
					>{$_('people_hidden_badge')}</span
				>
			{/if}
			<!-- The actions: naming (or renaming), the hidden toggle and
			     the merge-into fold. -->
			<div class="person-detail__actions ml-auto flex flex-wrap items-center gap-2">
				{#if detail.label === null}
					<Button
						onclick={() => {
							nameErrorKey = '';
							nameDialogOpen = true;
						}}><PenLine size={20} aria-hidden="true" />{$_('people_detail_name_cta')}</Button
					>
				{:else}
					<Button
						variant="ghost"
						onclick={() => {
							nameErrorKey = '';
							nameDialogOpen = true;
						}}><PenLine size={20} aria-hidden="true" />{$_('people_rename_action')}</Button
					>
				{/if}
				<Button variant="ghost" disabled={writeBusy} onclick={() => void toggleHidden()}>
					{#if hidden}
						<Eye size={20} aria-hidden="true" />
					{:else}
						<EyeOff size={20} aria-hidden="true" />
					{/if}
					{hidden ? $_('action_unhide') : $_('action_hide')}
				</Button>
				<Button
					variant="ghost"
					onclick={() => {
						mergeIntoErrorKey = '';
						mergeIntoOpen = true;
					}}><GitMerge size={20} aria-hidden="true" />{$_('people_merge_into_action')}</Button
				>
			</div>
		</header>
		{#if hidden}
			<!-- Badge semantics, not a filter: the hidden person stays in
			     the caller's list, flagged. -->
			<p class="person-detail__notice my-0 mb-4 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
				{$_('people_hidden_notice')}
			</p>
		{/if}
		{#if hideErrorKey !== ''}
			<!-- The hidden toggle's honest failure: hiding rides on the
			     label row, so an unlabeled cluster answers the
			     requires-label message -- and the naming CTA above is the
			     way out. -->
			<p
				class="person-detail__hide-error my-0 mb-4 font-body font-medium text-[14px] leading-[1.5] text-danger-text"
				role="alert"
			>
				{errorText(hideErrorKey)}
			</p>
		{/if}

		<h2 class="person-detail__faces mt-4 mb-3 font-display font-bold text-[20px] leading-[1.3] text-text">
			{$_('people_detail_faces_heading')}
		</h2>
		<div class="person-detail__grid grid grid-cols-[repeat(auto-fill,minmax(min(200px,100%),1fr))] gap-3">
			{#each detail.faces as face (face.id)}
				<!-- One face: the crop links into the asset viewer at the
				     face's asset (the IA's route-level viewer), the context
				     carries the file name and the capture date, and the
				     move button opens the single-face move dialog. -->
				<div
					class="face-card group relative flex flex-col overflow-hidden rounded-card bg-card border border-border"
				>
					<a
						class="face-card__link relative block aspect-square bg-accent-soft"
						href="/a/{encodeURIComponent(face.assetId)}"
						aria-label={$_('people_face_alt', { values: { file: face.asset.originalFilename } })}
					>
						<img
							class="absolute inset-0 w-full h-full object-cover"
							src={face.thumbnail}
							alt=""
							loading="lazy"
							decoding="async"
							draggable="false"
						/>
					</a>
					<div class="face-card__meta flex items-center justify-between gap-2 px-3 py-2">
						<div class="face-card__context min-w-0">
							<p
								class="face-card__file m-0 font-body font-semibold text-[14px] leading-[1.4] text-text truncate"
							>
								{face.asset.originalFilename}
							</p>
							<p
								class="face-card__date m-0 font-body font-normal text-[13px] leading-[1.4] text-text-soft"
							>
								{dateLabel(face.asset.takenAt)}
							</p>
						</div>
						<button
							class="face-card__move inline-flex items-center justify-center w-9 h-9 shrink-0 rounded-control bg-transparent text-text-soft cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
							type="button"
							aria-label={$_('people_face_move_action')}
							title={$_('people_face_move_action')}
							onclick={() => {
								moveErrorKey = '';
								moveFaceId = face.id;
							}}
						>
							<svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden="true">
								<path
									d="M4 12h13M12.5 6.5L18 12l-5.5 5.5"
									stroke="currentColor"
									stroke-width="1.75"
									stroke-linecap="round"
									stroke-linejoin="round"
								/>
								<path d="M20 4.5v15" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
							</svg>
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</section>

<PersonNameDialog
	bind:open={nameDialogOpen}
	titleKey="people_name_dialog_title"
	submitKey="people_name_submit"
	initialName={detail?.label?.name ?? ''}
	busy={writeBusy}
	errorKey={nameErrorKey}
	onsubmit={(name) => void submitName(name)}
/>

<ClusterPickerDialog
	bind:open={mergeIntoOpen}
	titleKey="people_merge_dialog_title"
	bodyKey="people_merge_into_body"
	targetLabelKey="people_merge_target_label"
	submitKey="people_merge_action"
	emptyKey="people_picker_empty"
	groups={mergeIntoGroups}
	loading={mergeIntoLoading}
	busy={writeBusy}
	errorKey={mergeIntoErrorKey}
	onsubmit={(targetId) => void runMergeInto(targetId)}
/>

<ClusterPickerDialog
	open={moveFaceId !== null}
	titleKey="people_move_dialog_title"
	bodyKey="people_move_body"
	targetLabelKey="people_move_target_label"
	submitKey="people_move_submit"
	emptyKey="people_picker_empty"
	groups={moveGroups}
	loading={moveLoading}
	busy={writeBusy}
	errorKey={moveErrorKey}
	onclose={() => (moveFaceId = null)}
	onsubmit={(targetId) => void runMove(targetId)}
/>
