# The m3-albums story's source fixture

The E4 albums-as-sources story (`web/e2e/m3-albums.spec.ts`) creates a
directory album that references a REAL directory on the server's
filesystem, then scans it. This tree is that directory: the repo's
compose file binds it into the lumemor container read-only at
`/mnt/e2e-source`, and the spec's administrator opens that root in the
instance's mount-root allowlist before the create dialog's completion
can list `coastal-week` under it.

- `coastal-week/` holds the two photos the story's album gathers --
  the same known-good face-portrait JPEGs the upload stories seed
  (`server/internal/ml/testdata/`), arriving through the source path
  this time: referenced in place, never uploaded.
- `trip-2026/` holds the four photos the M3d acceptance story's
  album gathers (scripts/acceptance_m3d.sh) -- byte-identical copies
  of the four fixtures that story's member uploads first (the
  Beijing GPS photo, the Chinese OCR photo, and the same-person face
  pair), so the scan's owner+checksum dedup answers each file with
  the honest skip: the EXISTING rows gain the album's membership, no
  second asset is ever created. The shared surface is therefore
  exactly the four the member already holds.
- The bind is `:ro` on purpose: the reference-style ingest writes
  nothing into the source (zero copy, zero move -- the scan that
  tried to would fail loudly against the read-only mount).
- The mount rides only the repository's own compose file. The
  dogfood stack composes its own file elsewhere and never sees it;
  with the allowlist empty (the default) the mount is inert -- no
  code path reads it until an administrator opens a root under it.
