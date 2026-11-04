import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for UnnamedClusterSummary
void main() {
  final instance = UnnamedClusterSummaryBuilder();
  // TODO add properties to the builder and call build()

  group(UnnamedClusterSummary, () {
    // Identifier of the face cluster.
    // String clusterId
    test('to test the property `clusterId`', () async {
      // TODO
    });

    // The cluster's faces on assets visible to the caller -- the honest per-caller count, not the global member count. 
    // int faceCount
    test('to test the property `faceCount`', () async {
      // TODO
    });

    // The cluster's representative face (its strongest-scoring member); the face the cluster tile renders. 
    // String representativeFaceId
    test('to test the property `representativeFaceId`', () async {
      // TODO
    });

    // URL of the representative face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the representative face's id. 
    // String thumbnail
    test('to test the property `thumbnail`', () async {
      // TODO
    });

  });
}
