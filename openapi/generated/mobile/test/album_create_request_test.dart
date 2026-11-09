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

    // The source directory: an existing absolute path the server validates against the path matrix (existence, directory-ness, readability, symlink resolution, the managed tree, root overlap, the mount-root allowlist) before creating anything. The album row stores the RESOLVED absolute path -- symlink segments evaluated -- so a later symlink swap cannot repoint the album. 
    // String rootPath
    test('to test the property `rootPath`', () async {
      // TODO
    });

    // The accounts the album starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors. 
    // BuiltList<String> audience
    test('to test the property `audience`', () async {
      // TODO
    });

  });
}
