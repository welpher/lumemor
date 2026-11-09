# The E6 writable source fixture

The album autonomy story (`web/e2e/m3-albums.spec.ts` scene set and the
acceptance chains) pins the deletability chain's TRUE arm end to end:
a writable source creation answers the write probe's `sourceWritable:
true`, the edit dialog renders the deletability row, and the trash
move physically relocates a file out of the directory (the restore
moves it back). A read-only bind cannot carry that story, so this
tree is bound into the lumemor container WITHOUT `:ro`, at
`/mnt/e2e-source-rw` -- the writable sibling of the read-only
`m3-source` mount.

- The bind is the E6 boundary itself: no allowlist layer remains, the
  compose mounts are what the container sees. An administrator's
  create dialog completes paths against this tree like any other the
  container holds.
- The photos are NEVER committed: they are seeded per run -- the spec
  bootstrap and the chains copy genuinely distinct bytes
  (`server/internal/ml/testdata/face-portrait-other*.jpg`, distinct
  from the read-only fixture's pair, so the rescan's owner+checksum
  dedup never links the two stories' rows) in as `IMG_E6_*.jpg`. The
  trash scene moves one of them into the managed trash area and the
  restore returns it, so the tree self-heals for the next run. Only
  this README is tracked; `web/e2e/fixtures/m3-source-rw/*` is
  git-ignored with this file excepted.
- The directory -- not the files -- is what the container's uid 10001
  must be able to write: the write probe creates and removes its own
  `probe-<uuid>` entry, and the trash move is a rename out of the
  directory. On Linux the acceptance chains chown the bind source to
  10001:10001 through the image (the `prepare_bind_mount` pattern,
  extended to this mount); macOS desktop runtimes map bind mounts
  permissively and need no preparation. A standalone spec run on a
  bare Linux host needs the same chown from its caller.
- The mount rides only the repository's own compose file. The dogfood
  stack composes its own file elsewhere and never sees it.
