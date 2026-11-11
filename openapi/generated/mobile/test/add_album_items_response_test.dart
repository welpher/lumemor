import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AddAlbumItemsResponse
void main() {
  final instance = AddAlbumItemsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(AddAlbumItemsResponse, () {
    // The membership rows the request landed -- the photos the creator currently sees and that were not members yet. Zero for an all-skipped request, which is still a 200. 
    // int added
    test('to test the property `added`', () async {
      // TODO
    });

    // The distinct ids that stayed out of the set: not currently visible to the caller (the re-share ground failed), not a live row, or already a member (the idempotent no-op). 
    // int skipped
    test('to test the property `skipped`', () async {
      // TODO
    });

  });
}
