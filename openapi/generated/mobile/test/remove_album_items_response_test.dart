import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for RemoveAlbumItemsResponse
void main() {
  final instance = RemoveAlbumItemsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(RemoveAlbumItemsResponse, () {
    // The membership rows the request deleted. Zero for an all-skipped request, which is still a 200. 
    // int removed
    test('to test the property `removed`', () async {
      // TODO
    });

    // The distinct ids that were no members of the album -- never added, or already removed. 
    // int skipped
    test('to test the property `skipped`', () async {
      // TODO
    });

  });
}
