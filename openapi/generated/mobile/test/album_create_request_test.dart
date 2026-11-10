import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumCreateRequest
void main() {
  final instance = AlbumCreateRequestBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumCreateRequest, () {
    // The album's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
    // String title
    test('to test the property `title`', () async {
      // TODO
    });

    // The source directory: an existing absolute path the server validates against the path matrix (existence, directory-ness, readability, symlink resolution, the managed tree) before creating anything. The album row stores the RESOLVED absolute path -- symlink segments evaluated -- so a later symlink swap cannot repoint the album. After the matrix passes, the server probes the directory's writability once and records the fact; a read-only directory is a valid source, never a refusal. 
    // String rootPath
    test('to test the property `rootPath`', () async {
      // TODO
    });

    // The owner's deletability switch, carried from the creation dialog: whether the album's photos may be deleted through the application (the trash move of the source file). On by default; the dialog renders the switch only where the source is writable (the completion row's `writable` fact), and on a read-only source the stored value governs nothing -- the derived `deletable` face stays false whatever it says. Editable later through the property patch. 
    // bool allowAppDeletion
    test('to test the property `allowAppDeletion`', () async {
      // TODO
    });

    // The creator's timeline preference, carried from the creation dialog: `false` keeps the album out of the CREATOR's aggregated timeline from birth -- the same per-user switch `PUT /albums/{albumId}/timeline-preference` writes, landed in the creation itself (a per-user preference, not an album fact: it needs no photos to exist, so the creation is exactly the right moment). Absent or true feeds the timeline (the default -- no preference row is written). The exclusion is the creator's own: the audience members' feeds are governed by their own preference rows, never by this field. 
    // bool inTimeline
    test('to test the property `inTimeline`', () async {
      // TODO
    });

    // The accounts the album starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors. 
    // BuiltList<String> audience
    test('to test the property `audience`', () async {
      // TODO
    });

  });
}
