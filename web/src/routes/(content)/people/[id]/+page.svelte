<script lang="ts">
	// The person detail: ONE page
	// for both states, because the people domain has exactly two --
	// named (this cluster carries the caller's label) and unnamed (it
	// does not). The page's body is THIS PERSON'S PHOTOS over the
	// shared wall; the person's identity lives in the profile row,
	// never in the context bar.
	//
	// THE CONTEXT BAR (the same skeleton as the album and
	// memory details): sticky, 53px tall, back + the icon action
	// cluster on the right, and the shell's own topbar and drawer do
	// not render while this route is active (the content shell's
	// route rule). The three detail pages now agree.
	//
	// THE PROFILE ROW: the 72px round avatar, the name at
	// 22px display with the rename pencil beside it, and the meta
	// line -- the photo count (icon + number) and the birth date
	// ("Born {date}", the empty state being the add affordance). The
	// unnamed variant renders the localized unnamed slot (the
	// people_unnamed_title copy) with the naming entry beside it; the
	// profile row still offers the birth date (the entity's attribute
	// does not depend on the name).
	//
	// THE ACTIONS: avatar picker / unname / merge-into for a named
	// person; avatar picker / merge-into / name for an unnamed one
	// (no unname -- there is no name to remove). Hiding retired with
	// No hide action exists anywhere on this page.
	//
	// THE PINNED CHOICES:
	// - The label write adopts its response (the PUT body IS the
	//   fresh label state): no refetch on rename or name.
	// - The unname is immediate and unceremonious (naming is
	//   reversible): the toast echo carries the feedback and the
	//   label adoption lands the UNNAMED variant in place -- the
	//   cluster is now in the caller's unnamed region.
	// - The birth date and the avatar adopt their responses; the
	//   profile row's own change (the date line, the crop) is the
	//   feedback.
	// - The merge-into fold posts THIS cluster as the single source
	//   and lands on the SURVIVING person's page: the landing is the
	//   feedback.
	// - SSE: the person events that touch THIS cluster (a label or
	//   entity write on it, a merge that dissolves or survives as it,
	//   a face write that joins or leaves it) refetch the detail; the
	//   asset events that change the photo set (trash, delete,
	//   restore) converge the profile row's count. The refetched
	//   answer owns the truth.
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import {
		fetchPersonCluster,
		labelPerson,
		listPeople,
		mergePeople,
		setPersonAvatar,
		setPersonBirthdate,
		unlabelPerson,
		type ClusterDetail,
		type PersonSummary
	} from '$lib/api/people';
	import { subscribeAssetEvents, subscribePersonEvents, type PersonEvent } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { toast } from '$lib/components/toast.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import UnreachableCard from '$lib/components/UnreachableCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import PersonNameDialog from '$lib/people/PersonNameDialog.svelte';
	import PersonAvatarDialog from '$lib/people/PersonAvatarDialog.svelte';
	import PersonBirthdateDialog from '$lib/people/PersonBirthdateDialog.svelte';
	import PersonPickerDialog from '$lib/people/PersonPickerDialog.svelte';
	import PersonWall from '$lib/people/PersonWall.svelte';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import PenLine from '@lucide/svelte/icons/pen-line';
	import GitMerge from '@lucide/svelte/icons/git-merge';
	import UserX from '@lucide/svelte/icons/user-x';
	import ImageIcon from '@lucide/svelte/icons/image';

	const clusterId = $derived(page.params.id ?? '');

	type DetailStatus = 'idle' | 'loading' | 'ready' | 'error';

	let detail = $state<ClusterDetail | null>(null);
	let status = $state<DetailStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');
	/**
	 * The dead end: the cluster read answered the byte-identical
	 * 404 (a merged-away cluster, another account's person) or a
	 * 403. The page renders the one unreachable face then -- the bar
	 * keeps the way back, and neither the name nor the count exists
	 * anywhere on the page.
	 */
	let unreachable = $state(false);

	// One write at a time across every action (the dialogs and the
	// bar toggles share the flag).
	let writeBusy = $state(false);

	// The naming / rename dialog.
	let nameDialogOpen = $state(false);
	let nameErrorKey = $state('');

	// The avatar picker dialog.
	let avatarDialogOpen = $state(false);
	let avatarErrorKey = $state('');

	// The birth date dialog.
	let birthdateDialogOpen = $state(false);
	let birthdateErrorKey = $state('');

	// The merge-into dialog: the existing people (minus this cluster)
	// as the target list, fetched on open.
	let mergeIntoOpen = $state(false);
	let mergeIntoErrorKey = $state('');
	let mergeIntoPeople = $state<PersonSummary[]>([]);
	let mergeIntoLoading = $state(false);

	// The unname write's honest failure (a dead session, a vanished
	// cluster) -- the bar owns the line.
	let unnameErrorKey = $state('');

	const firstLoad = $derived(detail === null && status === 'loading');
	const unnamed = $derived(detail?.label === null);

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
		unreachable = outcome.unreachable;
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
	// entity writes on it, a merge that dissolves it (this cluster
	// was a source) or survives it (it absorbed the sources), a face
	// write that joins it or leaves it.
	function touchesThisCluster(event: PersonEvent): boolean {
		const id = untrack(() => clusterId);
		switch (event.name) {
			case 'person.labeled':
			case 'person.unlabeled':
			case 'person.birthdate_updated':
			case 'person.avatar_updated':
				return event.clusterId === id;
			case 'person.merged':
				return event.survivingClusterId === id || event.mergedClusterIds.includes(id);
			case 'person.face_moved':
			case 'person.face_clustered':
			case 'person.face_detached':
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
		unreachable = false;
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
		// The photo set changed under the profile row (a batch hide or
		// unhide -- both publish asset.updated --, a trash, a purge, a
		// restore): the count is the server's, so the detail converges.
		// A mere metadata edit also rides asset.updated, and its cost is
		// one cheap read per event (the wall converges the tile anyway).
		const unsubscribeAssets = subscribeAssetEvents((event) => {
			switch (event.name) {
				case 'asset.updated':
				case 'asset.trashed':
				case 'asset.deleted':
				case 'asset.restored':
					void convergeDetail();
					break;
				default:
					break;
			}
		});
		return () => {
			fetchCurrent = false;
			unsubscribe();
			unsubscribeAssets();
		};
	});

	// The naming / rename write: the response IS the fresh label
	// state, so the page adopts it (naming an unnamed cluster lands
	// the named variant in place).
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
		detail = { ...detail, label: { name: outcome.label.name } };
		nameDialogOpen = false;
	}

	// The unname write: immediate, no confirm (naming is reversible).
	// The toast echo carries the feedback and the label adoption lands
	// the unnamed variant; the SSE self-event converges the same
	// answer.
	async function runUnname(): Promise<void> {
		if (writeBusy || detail === null || detail.label === null) {
			return;
		}
		writeBusy = true;
		unnameErrorKey = '';
		const outcome = await unlabelPerson(clusterId);
		writeBusy = false;
		if (!outcome.ok) {
			unnameErrorKey = outcome.errorKey;
			return;
		}
		detail = { ...detail, label: null };
		toast($_('person_unname_done'), 'success');
	}

	// The avatar pick: the response names the stored face, which the
	// profile row renders immediately.
	async function submitAvatar(faceId: string): Promise<void> {
		if (writeBusy || detail === null) {
			return;
		}
		writeBusy = true;
		avatarErrorKey = '';
		const outcome = await setPersonAvatar(clusterId, faceId);
		writeBusy = false;
		if (!outcome.ok) {
			avatarErrorKey = outcome.errorKey;
			return;
		}
		detail = { ...detail, avatarFaceId: outcome.avatar.avatarFaceId };
		avatarDialogOpen = false;
	}

	// The birth date write: the response carries the stored value
	// (null after a clear), and the profile row renders it.
	async function submitBirthdate(birthDate: string | null): Promise<void> {
		if (writeBusy || detail === null) {
			return;
		}
		writeBusy = true;
		birthdateErrorKey = '';
		const outcome = await setPersonBirthdate(clusterId, birthDate);
		writeBusy = false;
		if (!outcome.ok) {
			birthdateErrorKey = outcome.errorKey;
			return;
		}
		detail = { ...detail, birthDate: outcome.birthdate.birthDate };
		birthdateDialogOpen = false;
	}

	// The merge-into fold: this cluster is the single source, so it
	// dissolves (with its name -- the honest cost); the target's
	// label persists. Success lands on the SURVIVING person's page.
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
		void goto(`/people/${encodeURIComponent(outcome.merge.survivingClusterId)}`);
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

	/** The person's ceil of the date parts, in LOCAL time (a UTC parse
	 * would shift the day for half the world; the wire is a calendar
	 * date, not an instant). */
	function birthDateValue(date: string): Date {
		const [year, month, day] = date.split('-').map(Number);
		return new Date(year, month - 1, day);
	}

	// The locale-aware date: numeric year/month/day through
	// Intl.DateTimeFormat -- NEVER hand-assembled (zh-Hans reads
	// 1992/3/15, en-US 3/15/1992) -- with the wire value riding the
	// <time datetime> beside it.
	const birthDateLabel = $derived(
		detail?.birthDate === null || detail?.birthDate === undefined
			? ''
			: new Intl.DateTimeFormat($locale ?? 'en', {
					year: 'numeric',
					month: 'numeric',
					day: 'numeric'
				}).format(birthDateValue(detail.birthDate))
	);

	// The avatar face: the stored pick when it is one of the caller's
	// visible faces, otherwise the best visible member (the detail's
	// faces arrive strongest first, so the first row IS the fallback).
	const avatarFace = $derived.by(() => {
		if (detail === null) {
			return null;
		}
		const picked = detail.faces.find((face) => face.id === detail?.avatarFaceId);
		return picked ?? detail.faces[0] ?? null;
	});

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	const mergeIntoTargets = $derived(mergeIntoPeople.filter((entry) => entry.clusterId !== clusterId));
	const titleText = $derived(detail?.label?.name ?? $_('people_unnamed_title'));
</script>

<svelte:head>
	<title>{titleText} · lumemor</title>
</svelte:head>

<!-- The page root pins the wall's sticky month headers below the
     contextual bar: the bar is py-2 (16) around an h-9 row (36) plus
     its 1px border -- 53px. -->
<section class="person-detail w-full" style:--wall-sticky-offset="53px">
	<!-- THE CONTEXTUAL BAR: back + the person's icon actions.
	     The shell's own topbar and drawer hide while this route is
	     active (the content shell's route rule); the name, the photo
	     count and the birth date live in the profile row below. -->
	<header
		class="person-bar sticky top-0 z-(--z-sticky-bar) flex items-center gap-2 px-4 py-2 bg-card border-b border-border"
	>
		<a
			class="person-bar__back inline-flex items-center justify-center w-9 h-9 shrink-0 rounded-control bg-transparent text-text-soft no-underline cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/people"
			aria-label={$_('people_back')}
		>
			<ChevronLeft size={20} aria-hidden="true" />
		</a>
		<div class="person-bar__actions ml-auto flex items-center gap-1">
			{#if detail !== null}
				<IconButton
					disabled={writeBusy}
					aria-label={$_('person_avatar_action')}
					title={$_('person_avatar_action')}
					onclick={() => {
						avatarErrorKey = '';
						avatarDialogOpen = true;
					}}
				>
					<ImageIcon size={20} aria-hidden="true" />
				</IconButton>
				{#if unnamed}
					<!-- The unnamed variant's primary action is the naming
					     (the same dialog the profile row's pencil opens);
					     there is no name to remove. -->
					<IconButton
						disabled={writeBusy}
						aria-label={$_('people_detail_name_cta')}
						title={$_('people_detail_name_cta')}
						onclick={() => {
							nameErrorKey = '';
							nameDialogOpen = true;
						}}
					>
						<PenLine size={20} aria-hidden="true" />
					</IconButton>
				{:else}
					<!-- The unname: naming is reversible (a rename restores
					     it), so no confirm stands between the click and the
					     write; the toast is the echo. -->
					<IconButton
						disabled={writeBusy}
						aria-label={$_('person_unname_action')}
						title={$_('person_unname_action')}
						onclick={() => void runUnname()}
					>
						<UserX size={20} aria-hidden="true" />
					</IconButton>
				{/if}
				<IconButton
					disabled={writeBusy}
					aria-label={$_('people_merge_into_action')}
					title={$_('people_merge_into_action')}
					onclick={() => {
						mergeIntoErrorKey = '';
						mergeIntoOpen = true;
					}}
				>
					<GitMerge size={20} aria-hidden="true" />
				</IconButton>
			{/if}
		</div>
	</header>

	{#if firstLoad}
		<LoadingLine label={$_('people_loading')} />
	{:else if status === 'error' && unreachable}
		<!-- The dead end: one card, and none of the person's own face
		     -- no name, no count, no avatar -- because that face is
		     exactly what would tell a caller the cluster exists. The
		     bar keeps the way back alone. -->
		<UnreachableCard />
	{:else if status === 'error'}
		<StateCard
			tone="error"
			body={errorText(errorKey)}
			actionLabel={$_('action_retry')}
			onaction={() => void convergeDetail()}
		/>
	{:else if detail !== null}
		<!-- THE PROFILE ROW: the avatar, the name (with its pencil),
		     and the meta line -- the photo count and the birth date. -->
		<div class="person-profile flex items-center gap-4 px-4 py-4">
			<span
				class="person-profile__avatar relative block w-[72px] h-[72px] shrink-0 overflow-hidden rounded-full bg-accent-soft"
				aria-hidden="true"
			>
				{#if avatarFace !== null}
					<img
						class="absolute inset-0 w-full h-full object-cover"
						src={avatarFace.thumbnail}
						alt=""
						loading="lazy"
						decoding="async"
						draggable="false"
					/>
				{/if}
			</span>
			<div class="flex min-w-0 flex-col gap-1">
				<div class="flex items-center gap-2">
					<h1
						class="person-profile__name m-0 min-w-0 truncate font-display font-bold text-[22px] leading-[1.3] text-text"
						data-person-name
					>
						{titleText}
					</h1>
					<button
						class="person-profile__rename inline-flex items-center justify-center w-7 h-7 shrink-0 rounded-control bg-transparent text-text-faint cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
						type="button"
						aria-label={unnamed ? $_('people_detail_name_cta') : $_('people_rename_action')}
						title={unnamed ? $_('people_detail_name_cta') : $_('people_rename_action')}
						onclick={() => {
							nameErrorKey = '';
							nameDialogOpen = true;
						}}
					>
						<PenLine size={16} aria-hidden="true" />
					</button>
				</div>
				<p
					class="person-profile__meta m-0 inline-flex flex-wrap items-center gap-1.5 font-body font-normal text-[13px] leading-[1.4] text-text-soft"
				>
					<CountLabel
						class="person-profile__count font-body font-normal text-[13px] leading-[1.4] text-text-soft"
						count={detail.photoCount}
						label={$_('people_photo_count', { values: { count: detail.photoCount } })}
					/>
					<span class="text-text-faint" aria-hidden="true">·</span>
					<!-- The birth date: the stored value renders as the
					     locale-aware line with the machine value in <time>;
					     unset, the same slot is the add affordance. Both
					     open the dialog. -->
					<button
						class="person-profile__birthdate inline-flex items-center gap-1 rounded-control bg-transparent text-text-soft cursor-pointer transition-[color] duration-(--motion-fast) ease-standard hover:text-accent"
						type="button"
						onclick={() => {
							birthdateErrorKey = '';
							birthdateDialogOpen = true;
						}}
					>
						{#if detail.birthDate !== null}
							<time datetime={detail.birthDate}
								>{$_('person_birthdate', { values: { date: birthDateLabel } })}</time
							>
						{:else}
							{$_('person_birthdate_add')}
						{/if}
					</button>
				</p>
			</div>
		</div>

		{#if unnameErrorKey !== ''}
			<!-- The unname write's honest failure (a dead session, a
			     cluster another session dissolved mid-flight). -->
			<p
				class="person-detail__unname-error mx-4 my-0 mb-3 font-body font-medium text-[14px] leading-[1.5] text-danger-text"
				role="alert"
			>
				{errorText(unnameErrorKey)}
			</p>
		{/if}

		<!-- THE PHOTO WALL: this person's photos over the shared wall.
		     Keyed on the cluster id so another person's page always
		     walks its own photos. -->
		{#key clusterId}
			<PersonWall {clusterId} />
		{/key}
	{/if}
</section>

<PersonNameDialog
	bind:open={nameDialogOpen}
	titleKey={unnamed ? 'people_detail_name_cta' : 'people_rename_action'}
	submitKey="people_name_submit"
	initialName={detail?.label?.name ?? ''}
	busy={writeBusy}
	errorKey={nameErrorKey}
	onsubmit={(name) => void submitName(name)}
/>

<!-- The avatar picker: the person's faces are the candidates (no
     file names, no governance), the stored pick badged. -->
<PersonAvatarDialog
	bind:open={avatarDialogOpen}
	faces={detail?.faces ?? []}
	currentFaceId={detail?.avatarFaceId ?? null}
	busy={writeBusy}
	errorKey={avatarErrorKey}
	onsubmit={(faceId) => void submitAvatar(faceId)}
/>

<!-- The birth date dialog: the entity's own attribute, settable before
     the cluster is named. -->
<PersonBirthdateDialog
	bind:open={birthdateDialogOpen}
	initialDate={detail?.birthDate ?? null}
	busy={writeBusy}
	errorKey={birthdateErrorKey}
	onsubmit={(birthDate) => void submitBirthdate(birthDate)}
/>

<!-- The merge face: the filterable person target list (this cluster
     never its own target). The submit folds this cluster into the
     picked person; the navigation lands on the surviving person's
     page -- the landing is the feedback, no toast beside it. -->
<PersonPickerDialog
	bind:open={mergeIntoOpen}
	titleKey="people_merge_into_action"
	bodyKey="people_merge_into_body"
	submitKey="people_merge_into_action"
	groupKey="people_merge_group_people"
	people={mergeIntoTargets}
	loading={mergeIntoLoading}
	errorKey={mergeIntoErrorKey}
	onsubmit={(targetId) => void runMergeInto(targetId)}
/>
