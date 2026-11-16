import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TimelineItem
void main() {
  final instance = TimelineItemBuilder();
  // TODO add properties to the builder and call build()

  group(TimelineItem, () {
    // UUIDv7 identifier of the asset.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // UUIDv7 identifier of the owning account.
    // String ownerId
    test('to test the property `ownerId`', () async {
      // TODO
    });

    // Media family of the asset. A Live Photo is an `image` whose linked component video row never appears in the timeline. 
    // String kind
    test('to test the property `kind`', () async {
      // TODO
    });

    // The file name the asset was uploaded with.
    // String originalFilename
    test('to test the property `originalFilename`', () async {
      // TODO
    });

    // Size of the stored file in bytes.
    // int fileSize
    test('to test the property `fileSize`', () async {
      // TODO
    });

    // Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`. 
    // String mimeType
    test('to test the property `mimeType`', () async {
      // TODO
    });

    // SHA-256 digest of the file bytes, lowercase hex. The prefix of the entity tags the media endpoints serve. 
    // String checksum
    test('to test the property `checksum`', () async {
      // TODO
    });

    // When the asset record was created (RFC 3339 UTC).
    // DateTime createdAt
    test('to test the property `createdAt`', () async {
      // TODO
    });

    // Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; the timeline orders by it. 
    // DateTime takenAt
    test('to test the property `takenAt`', () async {
      // TODO
    });

    // Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer). 
    // String takenAtSource
    test('to test the property `takenAtSource`', () async {
      // TODO
    });

    // Always false in the timeline: trashed assets never enter it. The field exists to keep the shape identical to the asset detail's. 
    // bool isTrashed
    test('to test the property `isTrashed`', () async {
      // TODO
    });

    // The row's archive state: `false` on every family-flow wall (the flows exclude archived rows), `true` on the archived view's items and on an album's archived members (the container keeps them visible and unarchivable). 
    // bool isArchived
    test('to test the property `isArchived`', () async {
      // TODO
    });

    // Whether the REQUESTING user favorites this asset: the caller's own row of the per-user favorites state, written by the batch endpoint's favorite action. False whenever the caller has no favorites row for the asset -- the same asset can be a favorite for one viewer of a shared timeline and not for another. 
    // bool isFavorite
    test('to test the property `isFavorite`', () async {
      // TODO
    });

    // True once the whole pipeline of the asset (metadata and serving copies) completed; until then the pipeline-dependent fields are null and the client renders placeholders. 
    // bool isReady
    test('to test the property `isReady`', () async {
      // TODO
    });

    // Pixel width in sensor order, straight from the assets column: the upload writes the 0 placeholder and the metadata job refines the real value, so 0 means \"not refined yet\" (the wall layout assumes a fallback aspect until then). Always present -- the justified photo wall reads it on every tile, never null. 
    // int width
    test('to test the property `width`', () async {
      // TODO
    });

    // Pixel height in sensor order, straight from the assets column: the upload writes the 0 placeholder and the metadata job refines the real value, so 0 means \"not refined yet\" (the wall layout assumes a fallback aspect until then). Always present -- the justified photo wall reads it on every tile, never null. 
    // int height
    test('to test the property `height`', () async {
      // TODO
    });

    // Playback duration of a video in milliseconds; null for images and until the metadata job refined it. 
    // int durationMs
    test('to test the property `durationMs`', () async {
      // TODO
    });

    // The blurhash of the tile (image assets) or poster frame (video assets) for placeholder rendering; null until the thumbnail job stored it. 
    // String blurhash
    test('to test the property `blurhash`', () async {
      // TODO
    });

    // For a Live Photo image, the id of its linked component video row (its original is downloadable by that id, the transcoded play stream is not); null otherwise. 
    // String livePhotoVideoId
    test('to test the property `livePhotoVideoId`', () async {
      // TODO
    });

  });
}
