// The picker's test state: a $state object the tests mutate from
// outside while the mounted dialog reads it through getter props
// (the documented external-reactivity pattern -- a plain props
// object cannot change under a mounted Svelte 5 component).
export const pickerState = $state({
	open: true,
	currentCoverIds: [] as string[]
});
