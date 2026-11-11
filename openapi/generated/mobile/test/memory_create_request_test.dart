import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for MemoryCreateRequest
void main() {
  final instance = MemoryCreateRequestBuilder();
  // TODO add properties to the builder and call build()

  group(MemoryCreateRequest, () {
    // The memory's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
    // String title
    test('to test the property `title`', () async {
      // TODO
    });

    // The accounts the memory starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors. 
    // BuiltList<String> audience
    test('to test the property `audience`', () async {
      // TODO
    });

  });
}
