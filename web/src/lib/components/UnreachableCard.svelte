<script lang="ts">
	// The unreachable face: the ONE card every dead end renders -- an
	// unknown address, an unknown id, another account's resource, a
	// share that has been withdrawn. The contract answers them all
	// with the same 404 byte for byte, so the face keeps the same
	// silence: it takes NO props, renders no resource name, no cover,
	// no count and no skeleton, and its copy never says which of the
	// two causes it was. A card that could carry an identifier would
	// leak whether the thing exists.
	//
	// The way back is the timeline; the footnote names the two
	// non-answers (an incomplete address, a withdrawn share) without
	// distinguishing them either. The glyph is the face's own mark,
	// decorative.
	import { _ } from 'svelte-i18n';
	import FileQuestion from '@lucide/svelte/icons/file-question';
</script>

<div
	class="unreachable-card mx-auto my-16 max-w-[420px] box-content rounded-card border border-border bg-card p-8 text-center shadow-card"
>
	<span class="unreachable-card__mark inline-flex text-text-faint">
		<FileQuestion size={28} aria-hidden="true" />
	</span>
	<p class="unreachable-card__title m-0 mt-3 font-display font-bold text-[16px] leading-[1.3] text-text">
		{$_('unreachable_title')}
	</p>
	<p class="unreachable-card__body m-0 mt-2 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
		{$_('unreachable_body')}
	</p>
	<a
		class="btn mt-5 inline-flex items-center justify-center gap-2 px-4 py-2 font-body font-semibold text-[15px] leading-[1.2] rounded-control border border-transparent bg-accent text-accent-on no-underline hover:bg-accent-strong"
		href="/timeline">{$_('unreachable_action')}</a
	>
	<p class="unreachable-card__footnote m-0 mt-3 font-body font-normal text-[12px] leading-[1.4] text-text-faint">
		{$_('unreachable_footnote')}
	</p>
</div>
