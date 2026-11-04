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

    // The cluster's faces on assets visible to the caller -- the honest per-caller count, not the global member count. 
    // int faceCount
    test('to test the property `faceCount`', () async {
      // TODO
    });

    // The cluster's representative face (its strongest-scoring member); the face the person card renders. 
    // String representativeFaceId
    test('to test the property `representativeFaceId`', () async {
      // TODO
    });

    // URL of the representative face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the representative face's id. 
    // String thumbnail
    test('to test the property `thumbnail`', () async {
      // TODO
    });

    // The caller's hidden switch for this person. Hidden people stay listed carrying this flag (badge semantics); they are not filtered out. 
    // bool isHidden
    test('to test the property `isHidden`', () async {
      // TODO
    });

  });
}
