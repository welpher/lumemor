import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for RemoveTagAssetsResponse
void main() {
  final instance = RemoveTagAssetsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(RemoveTagAssetsResponse, () {
    // The assignment rows the request deleted. Zero for an all-skipped request, which is still a 200. 
    // int removed
    test('to test the property `removed`', () async {
      // TODO
    });

    // The distinct ids that carried no assignment of this tag -- never tagged, or already untagged. 
    // int skipped
    test('to test the property `skipped`', () async {
      // TODO
    });

  });
}
