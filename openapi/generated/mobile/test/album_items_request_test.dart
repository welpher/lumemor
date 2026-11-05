import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumItemsRequest
void main() {
  final instance = AlbumItemsRequestBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumItemsRequest, () {
    // The asset ids of the membership write, between 1 and 200 entries. The add skips -- never errors on -- every id that is not a live caller-owned asset or that already sits in the album; the remove skips the ids that are no members. 
    // BuiltList<String> assetIds
    test('to test the property `assetIds`', () async {
      // TODO
    });

  });
}
