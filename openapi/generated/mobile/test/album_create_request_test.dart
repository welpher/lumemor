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

    // The album's free-text description; empty when absent. At most 10,000 Unicode characters, the asset description's own bound. 
    // String description
    test('to test the property `description`', () async {
      // TODO
    });

  });
}
