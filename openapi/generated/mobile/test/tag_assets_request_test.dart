import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for TagAssetsRequest
void main() {
  final instance = TagAssetsRequestBuilder();
  // TODO add properties to the builder and call build()

  group(TagAssetsRequest, () {
    // The asset ids of the assignment write, between 1 and 200 entries. The add skips -- never errors on -- every id that is not caller-visible or that already carries the tag; the remove skips the ids that carry no assignment. 
    // BuiltList<String> assetIds
    test('to test the property `assetIds`', () async {
      // TODO
    });

  });
}
