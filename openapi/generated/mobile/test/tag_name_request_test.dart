import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TagNameRequest
void main() {
  final instance = TagNameRequestBuilder();
  // TODO add properties to the builder and call build()

  group(TagNameRequest, () {
    // The tag's name, trimmed server-side. On create a name the caller already owns converges on the existing tag; on rename a name another tag carries answers 409 `TAG_NAME_TAKEN`, the row's own name the idempotent no-op. 
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

  });
}
