import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for MountRootsSettingsResponse
void main() {
  final instance = MountRootsSettingsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(MountRootsSettingsResponse, () {
    // The configured mount roots: absolute directory paths, at most 32, stored as written. A root's live usability (exists, is a directory, outside the managed tree) is the completion and creation domains' concern, not this list's -- the list is the honest record of the configuration. 
    // BuiltList<String> roots
    test('to test the property `roots`', () async {
      // TODO
    });

  });
}
