import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for PersonSummary
void main() {
  final instance = PersonSummaryBuilder();
  // TODO add properties to the builder and call build()

  group(PersonSummary, () {
    // Identifier of the face cluster.
    // String clusterId
    test('to test the property `clusterId`', () async {
      // TODO
    });

    // The caller's own label of the cluster.
    // String name
    test('to test the property `name`', () async {
      // TODO
    });

    // The number of DISTINCT live assets the cluster has a face on that the caller can see -- photos, not faces, so two faces of one person in one photo count once. Honest per caller, never the global count. 
    // int photoCount
    test('to test the property `photoCount`', () async {
      // TODO
    });

    // The face the person card renders: the person's chosen avatar face when it is a caller-visible live member of the cluster, otherwise the cluster's strongest-scoring caller-visible face (score descending, the face id ascending as the tiebreak) -- the best face the CALLER can see, never a face on an asset the caller cannot view. The field is the thumbnail's face id. 
    // String representativeFaceId
    test('to test the property `representativeFaceId`', () async {
      // TODO
    });

    // URL of the card's face crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with `representativeFaceId`. 
    // String thumbnail
    test('to test the property `thumbnail`', () async {
      // TODO
    });

  });
}
