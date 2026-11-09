import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumPatch
void main() {
  final instance = AlbumPatchBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumPatch, () {
    // The album's display title, trimmed server-side; when present it must span 1-64 Unicode characters after the trim. 
    // String title
    test('to test the property `title`', () async {
      // TODO
    });

    // The pinned cover asset. When present it must be a LIVE member of THIS album (the membership row exists and the asset is not trashed, not hidden, not a Live Photo component row) -- otherwise the answer is 400 with the message key `album_cover_not_member`. Absent (or JSON null, which decodes as absent) leaves the pin unchanged. 
    // String coverAssetId
    test('to test the property `coverAssetId`', () async {
      // TODO
    });

    // The owner's lock. `true` freezes the album's curation and property fields against later writes (this same endpoint refuses them with 409 until the owner patches `locked: false` -- the unlock, never gated itself). Locking does not gate the album's delete. 
    // bool locked
    test('to test the property `locked`', () async {
      // TODO
    });

  });
}
