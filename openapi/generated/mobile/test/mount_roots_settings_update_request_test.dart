import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for MountRootsSettingsUpdateRequest
void main() {
  final instance = MountRootsSettingsUpdateRequestBuilder();
  // TODO add properties to the builder and call build()

  group(MountRootsSettingsUpdateRequest, () {
    // Every mount root of the new allowlist, at most 32 entries. Each must be an absolute path within the 4,096-byte bound; the server cleans each value and collapses duplicates after the clean. Existence is not required -- a root may be configured before the volume is mounted. 
    // BuiltList<String> roots
    test('to test the property `roots`', () async {
      // TODO
    });

  });
}
