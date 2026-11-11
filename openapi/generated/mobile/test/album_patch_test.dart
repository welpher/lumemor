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

    // The owner's deletability switch of a DIRECTORY album: whether the album's photos may be deleted through the application (the trash move of the source file). On for a writable source by default; the owner who manages file-system deletions themselves turns it off, and the wall's delete affordances disappear with it (the derived `deletable` face of the detail read). A patch carrying the field on a memory album answers 400 with the message key `album_deletable_not_directory` -- the switch is the directory album's own, a memory album's deletions never move files. Idempotent: repeating the stored value converges. 
    // bool allowAppDeletion
    test('to test the property `allowAppDeletion`', () async {
      // TODO
    });

  });
}
