import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for FaceBoxPatchResponse
void main() {
  final instance = FaceBoxPatchResponseBuilder();
  // TODO add properties to the builder and call build()

  group(FaceBoxPatchResponse, () {
    // Identifier of the rewritten face.
    // String faceId
    test('to test the property `faceId`', () async {
      // TODO
    });

    // URL of the face's crop, which now derives from the new box (the crop endpoint computes at query time; the change signal is the face row's updated_at). 
    // String thumbnail
    test('to test the property `thumbnail`', () async {
      // TODO
    });

    // FaceBbox bbox
    test('to test the property `bbox`', () async {
      // TODO
    });

  });
}
