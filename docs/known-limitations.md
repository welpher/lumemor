# Known limitations

The boundaries this release ships with on purpose. Each one is a decision,
not an oversight; an item leaves this list when the behavior changes.

## The instance serves one family

Every deployment is a single household: one shared library, one set of
users, no tenant isolation. Two families in one instance would see each
other's photos, so multi-tenancy is out of scope by design.

## The mobile app is separate

This repository holds the server and the web app. The mobile app is not
part of it and has not been released; the OpenAPI contract in
[`openapi/lumemor.yaml`](../openapi/lumemor.yaml) is the integration point.

## AI features are optional and need a download

Faces, semantic search and OCR run locally, but the model weights are not
bundled. An instance that never downloads them simply runs without AI:
search keeps its text and filters, and no face, place or OCR intelligence
runs.

## The map needs a tile source you choose

No tile service ships as a default — a public provider's usage policy
cannot carry an application's whole install base. Until an administrator
configures one, the map draws its pins over a stated missing base layer,
never a blank canvas.

## Directory albums are references, not copies

A directory album points at files where they already live: the application
never writes into the folder. A photo inside a read-only source cannot be
deleted from the library (delete it where it lives and the next scan
notices); a writable source's photo can move to the trash.

## Archived photos leave the family flows

An archived photo disappears from the timeline, search, favorites, people,
places, maps, memories and every count, while staying inside the albums it
belongs to. Unarchiving brings it back everywhere.

## Two translations are maintained

`en` and `zh-Hans` are official. Other languages are welcome as pull
requests; a translation platform is planned for when the community
outgrows that.

## A person's avatar can stay stale across sessions

If another session changes a face box, an already-open session may show the
old crop until it reloads. Crop URLs are immutable by design; a reload
picks up the new crop.