# The m3-albums story's source fixture

The albums-as-sources story (`web/e2e/m3-albums.spec.ts`) creates a
directory album that references a REAL directory on the server's
filesystem, then scans it. This tree is that directory: the repo's
compose file binds it into the lumemor container read-only at
`/mnt/e2e-source`, and the boundary is the compose mount itself --
the administrator's create dialog completes paths against whatever the
container sees, with no allowlist layer left to open.

- `coastal-week/` holds the two photos the story's album gathers --
  the same known-good face-portrait JPEGs the upload stories seed
  (`server/internal/ml/testdata/`), arriving through the source path
  this time: referenced in place, never uploaded.
- `trip-2026/` holds the four photos the search acceptance
  story's album gathers (the m3-search spec) -- byte-identical copies
  of the four fixtures that story's administrator uploads first (the
  Beijing GPS photo, the Chinese OCR photo, and the same-person face
  pair), so the scan's owner+checksum dedup answers each file with
  the honest skip: the EXISTING rows gain the album's membership, no
  second asset is ever created. The shared surface is therefore
  exactly the four the owner already holds.
- The bind is `:ro` on purpose: the source ingest writes
  nothing into the source (zero copy, zero move -- the scan that
  tried to would fail loudly against the read-only mount), and the
  creation-time write probe answers the honest `sourceWritable:
  false` against it -- a read-only directory is a legitimate album
  whose photos cannot be deleted through the application. The
  writable arm of that story lives in the `m3-source-rw` sibling.
- The mount rides only the repository's own compose file. The dogfood
  stack composes its own file elsewhere and never sees it.
