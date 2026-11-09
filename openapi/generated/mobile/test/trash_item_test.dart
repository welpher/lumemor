import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TrashItem
void main() {
  final instance = TrashItemBuilder();
  // TODO add properties to the builder and call build()

  group(TrashItem, () {
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

    // Media family of the asset.
    // String kind
    test('to test the property `kind`', () async {
      // TODO
    });

    // Where the asset's original file lives. `blob` rows were uploaded into the server's originals tree and were soft-deleted in place (the file stays until expiry). `external` rows reference a directory album's source file and were moved INTO the server's trash area when the owner trashed them -- the listing is honest for both kinds, and a restore moves the external file back to its recorded original location. External rows the rescan's orphan arm marked deleted never appear here at all: the file was never the server's and the trash has no meaning for it. 
    // String storageKind
    test('to test the property `storageKind`', () async {
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

    // Capture time (RFC 3339), the value the timeline would order the asset by. 
    // DateTime takenAt
    test('to test the property `takenAt`', () async {
      // TODO
    });

    // Where `takenAt` comes from.
    // String takenAtSource
    test('to test the property `takenAtSource`', () async {
      // TODO
    });

    // True once the whole pipeline of the asset (metadata and serving copies) completed. 
    // bool isReady
    test('to test the property `isReady`', () async {
      // TODO
    });

    // Pixel width in sensor order; null until the metadata job refined the dimensions. 
    // int width
    test('to test the property `width`', () async {
      // TODO
    });

    // Pixel height in sensor order; null until the metadata job refined the dimensions. 
    // int height
    test('to test the property `height`', () async {
      // TODO
    });

    // Playback duration of a video in milliseconds; null for images and until the metadata job refined it. 
    // int durationMs
    test('to test the property `durationMs`', () async {
      // TODO
    });

    // The blurhash of the tile for placeholder rendering; null until the thumbnail job stored it. 
    // String blurhash
    test('to test the property `blurhash`', () async {
      // TODO
    });

    // For a Live Photo image, the id of its linked component video row (restored and expired together with the photo); null otherwise. 
    // String livePhotoVideoId
    test('to test the property `livePhotoVideoId`', () async {
      // TODO
    });

    // When the asset was soft deleted (RFC 3339 UTC). The retention window ends at `deletedAt` plus the instance's `trash_retention_days`. 
    // DateTime deletedAt
    test('to test the property `deletedAt`', () async {
      // TODO
    });

    // Whole days until the expiry job removes the asset, rounded up (an asset trashed moments ago with the default retention of 30 days shows 30). 0 once the deadline has passed -- the row survives only until the next hourly expiry run. 
    // int remainingDays
    test('to test the property `remainingDays`', () async {
      // TODO
    });

  });
}
