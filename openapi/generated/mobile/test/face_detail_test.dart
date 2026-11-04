import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for FaceDetail
void main() {
  final instance = FaceDetailBuilder();
  // TODO add properties to the builder and call build()

  group(FaceDetail, () {
    // Identifier of the face row.
    // String id
    test('to test the property `id`', () async {
      // TODO
    });

    // Identifier of the asset the face sits on.
    // String assetId
    test('to test the property `assetId`', () async {
      // TODO
    });

    // The detector's confidence for this face.
    // double score
    test('to test the property `score`', () async {
      // TODO
    });

    // URL of the face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with this face's id, serving the bounding-box crop of the asset's view-size derivative. 
    // String thumbnail
    test('to test the property `thumbnail`', () async {
      // TODO
    });

    // FaceDetailAsset asset
    test('to test the property `asset`', () async {
      // TODO
    });

  });
}
