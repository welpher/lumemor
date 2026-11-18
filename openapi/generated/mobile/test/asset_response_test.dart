import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AssetResponse
void main() {
  final instance = AssetResponseBuilder();
  // TODO add properties to the builder and call build()

  group(AssetResponse, () {
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

    // The file name the asset was uploaded with: the client-side `fileName` of the photo (the component video keeps its own file part name). 
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

    // SHA-256 digest of the file bytes, lowercase hex. The owner-scoped idempotency key of the upload. 
    // String checksum
    test('to test the property `checksum`', () async {
      // TODO
    });

    // The upload outcome: `created` (201), `duplicate` (200, an identical live asset) or `trashed` (200, an identical asset in the trash). 
    // String state
    test('to test the property `state`', () async {
      // TODO
    });

    // When the asset record was created (RFC 3339 UTC).
    // DateTime createdAt
    test('to test the property `createdAt`', () async {
      // TODO
    });

    // Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; responses render the best value the server has at answer time. 
    // DateTime takenAt
    test('to test the property `takenAt`', () async {
      // TODO
    });

    // True when the asset sits in the trash; `state: trashed` responses carry it as true so clients can offer the restore or save-as-new decision. 
    // bool isTrashed
    test('to test the property `isTrashed`', () async {
      // TODO
    });

    // What the client must tell the user about this upload. Absent or empty when there is nothing to say; the array is informational, never a failure list. Present on a 201 `created` answer today (a Live Photo motion absorbed into an existing video); the client surfaces each entry to the user without blocking the upload. 
    // BuiltList<AssetNotice> notices
    test('to test the property `notices`', () async {
      // TODO
    });

  });
}
