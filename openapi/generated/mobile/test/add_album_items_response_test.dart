import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AddAlbumItemsResponse
void main() {
  final instance = AddAlbumItemsResponseBuilder();
  // TODO add properties to the builder and call build()

  group(AddAlbumItemsResponse, () {
    // The membership rows the request inserted. Zero for an all-skipped request, which is still a 200 -- batch semantics, never an error. 
    // int added
    test('to test the property `added`', () async {
      // TODO
    });

    // The distinct ids that did not land: unknown ids, another account's assets, trashed, hidden or component rows, and the ids already sitting in the album (the idempotent add). 
    // int skipped
    test('to test the property `skipped`', () async {
      // TODO
    });

  });
}
