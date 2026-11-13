// The path-id shape guard. Every detail route names its resource by
// the contract's UUID, and the server's binder rejects a path id of
// any other shape with 400 VALIDATION_FAILED before it looks anything
// up. That refusal is retryable by classification, which is the wrong
// voice for a mistyped address: nothing will change on a retry, and a
// non-UUID string names no resource at all, so nothing can be
// revealed by answering it as unreachable. The detail clients call
// this before the fetch and render the dead-end face instead.
const UUID_SHAPE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Whether a path id carries the contract's UUID shape. */
export function isUuidShaped(id: string): boolean {
	return UUID_SHAPE.test(id);
}
