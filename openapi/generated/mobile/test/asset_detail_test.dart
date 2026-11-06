import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AssetDetail
void main() {
  final instance = AssetDetailBuilder();
  // TODO add properties to the builder and call build()

  group(AssetDetail, () {
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

    // Media family of the asset. A Live Photo is an `image` whose video component is stored as a linked `video` row that never appears in the timeline. 
    // String kind
    test('to test the property `kind`', () async {
      // TODO
    });

    // The file name the asset was uploaded with; the original download re-uses it (RFC 5987 escaped) for `Content-Disposition`. 
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

    // Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; the response renders the best value the server has at answer time. 
    // DateTime takenAt
    test('to test the property `takenAt`', () async {
      // TODO
    });

    // Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer). 
    // String takenAtSource
    test('to test the property `takenAtSource`', () async {
      // TODO
    });

    // True when the asset sits in the trash.
    // bool isTrashed
    test('to test the property `isTrashed`', () async {
      // TODO
    });

    // True while the owner-level visibility switch is off: the asset is out of every live view (the timeline, the favorites) and manageable only through the hidden view, until it is unhidden. The same switch `PATCH /assets/{id}` and the batch `hide` / `unhide` actions flip. 
    // bool isHidden
    test('to test the property `isHidden`', () async {
      // TODO
    });

    // The owner's free-text description of the asset, the text the detail panel renders; the empty string until the owner wrote one. Editable only by the owner through `PATCH /assets/{id}`, which replaces it wholesale. 
    // String description
    test('to test the property `description`', () async {
      // TODO
    });

    // True once the whole pipeline of the asset completed -- the metadata and thumbnail jobs for an image, plus the transcode job for a video (for a Live Photo component, the transcode job alone); the `asset.ready` event fires at that moment. Until then the client shows placeholders and the media endpoints answer `ASSET_NOT_READY` for missing derived files. 
    // bool isReady
    test('to test the property `isReady`', () async {
      // TODO
    });

    // Pixel width in sensor order (not orientation-corrected); null until the metadata job refined the dimensions. 
    // int width
    test('to test the property `width`', () async {
      // TODO
    });

    // Pixel height in sensor order (not orientation-corrected); null until the metadata job refined the dimensions. 
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

    // For a Live Photo image, the id of its linked component video row (never in the timeline; its original is downloadable by that id, and its transcoded play stream is served there too); null otherwise. 
    // String livePhotoVideoId
    test('to test the property `livePhotoVideoId`', () async {
      // TODO
    });

    // The EXIF hot fields. Always present as an object; every field inside it is null until the metadata job wrote the extraction. 
    // ExifSummary exif
    test('to test the property `exif`', () async {
      // TODO
    });

    // The asset's faces as the calling user sees them, strongest score first with the face id as the tiebreak. The array is caller-scoped (B4): the asset being visible to the caller is the gate -- the faces ride on the detail's own permission -- and each entry's `label` is the caller's own label of the face's cluster, never another account's. Always present, empty when the asset carries no faces (none detected, or detection has not run yet). The PATCH response serves the same array, exactly as the GET serves it. 
    // BuiltList<AssetFaceSummary> faces
    test('to test the property `faces`', () async {
      // TODO
    });

    // The CALLER's tags on this asset (B4): the caller's own assignment rows joined to their own tag rows, sorted by name case-insensitively with the tag id as the tiebreak (the vocabulary listing's own ordering). Another account's tags on the same asset never surface, so the same shared asset answers different arrays for the owner and the sharee (each sees their own, or none). Always present, empty when the caller tagged the asset with nothing. The PATCH response serves the same array, exactly as the GET serves it. 
    // BuiltList<AssetTagSummary> tags
    test('to test the property `tags`', () async {
      // TODO
    });

  });
}
