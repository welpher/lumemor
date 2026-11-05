import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AlbumItemAsset
void main() {
  final instance = AlbumItemAssetBuilder();
  // TODO add properties to the builder and call build()

  group(AlbumItemAsset, () {
    // When the asset was taken (RFC 3339 UTC).
    // DateTime takenAt
    test('to test the property `takenAt`', () async {
      // TODO
    });

    // The file name the asset was uploaded under.
    // String originalFilename
    test('to test the property `originalFilename`', () async {
      // TODO
    });

    // The asset's media kind.
    // String kind
    test('to test the property `kind`', () async {
      // TODO
    });

    // URL of the asset's tile-size serving copy: the asset thumbnail endpoint `GET /api/v1/assets/{assetId}/thumbnail?size=tile`. 
    // String thumbnail
    test('to test the property `thumbnail`', () async {
      // TODO
    });

  });
}
