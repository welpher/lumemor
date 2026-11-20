# Third-party components

lumemor is distributed under **AGPL-3.0** (see `LICENSE`). This file names the
third-party components a built image or this repository carries, with their
licenses and where the authoritative text lives. It is a pointer, not a
replacement: each component's own license governs.

The AGPL's source offer is satisfied by this repository at the image's release
tag (the published image records its VERSION build argument).

## The container image

Built from `docker/server.Dockerfile` on Debian trixie-slim. The Debian
packages' licenses ship inside the image under `/usr/share/doc/<package>/copyright`.

| Component | License | Notes |
|---|---|---|
| Debian trixie (base, runtime and builder) | mixed (each package) | `/usr/share/doc/*/copyright` |
| Node.js 22 (web build stage) | MIT | build-time only |
| Go 1.26 (server build stage) | BSD-3-Clause | build-time only |
| libvips (`libvips42t64`) | LGPL-2.1-or-later | the thumbnail/derivative pipeline |
| libheif (`libheif1` + the aomenc/x265/libde265 plugins) | LGPL-3.0-or-later (plugins per their own terms) | AVIF/HEIC encode and decode |
| FFmpeg (Debian build) | the Debian build's own configuration (GPL/LGPL) | video poster extraction |
| ExifTool | Perl Artistic / GPL | metadata subprocess |
| util-linux | GPL-2.0-or-later | `prlimit(1)` for the subprocess memory caps |
| ca-certificates | MPL-2.0 | outbound TLS trust store |
| tzdata | public domain | zoneinfo for `TZ` |
| ONNX Runtime 1.31.0 | MIT | `onnxruntime-linux-*.tgz`, sha256 pinned in the Dockerfile |

## Geodata baked into the image

The place database (GeoNames `cities500`, `admin1CodesASCII`,
`alternateNamesV2`) is baked at build time and shipped in the image.
**GeoNames data is licensed CC BY 4.0** (https://www.geonames.org/); the
baked subset is used unmodified apart from the locale filtering the build
performs. Attribution: © GeoNames, CC BY 4.0.

## Models

The built-in model manifest (`server/internal/ml/manifests/default.json`)
records each entry's license. The **weights are downloaded at runtime** from
the source the administrator configures; the image does not bundle them.
Entries whose upstream card declares no license are recorded honestly as
`unknown` there.

Committed test weights and fixtures:

| File | License | Attribution |
|---|---|---|
| `server/internal/ml/testdata/face_detection_yunet_2023mar.onnx` | MIT, Copyright (c) 2020 Shiqi Yu | OpenCV Zoo's model directory; text reproduced in `server/internal/ml/testdata/README.md` |
| `server/internal/ml/testdata/cn_clip_vocab.txt` | the Chinese-CLIP project's BERT-Chinese vocabulary | the table's normal use; see the same README |
| The remaining test fixtures | self-made renders or freely licensed sources | each fixture's row in the same README |

## Go and web dependencies

The Go module and npm dependency trees are pinned by `server/go.sum` and
`web/pnpm-lock.yaml`; each dependency's own license applies. Their texts ship
with the modules in the build stage and are not redistributed in the runtime
image.

## Assets in this repository

The UI fonts are bundled from their packages (SIL Open Font License for the
Nunito faces; see the `@fontsource` packages). Sample/test imagery is either
self-made or freely licensed, attributed where it lands.
