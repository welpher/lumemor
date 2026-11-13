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

    // The face the cluster tile renders: the person's chosen avatar face when it is a caller-visible live member of the cluster, otherwise the cluster's strongest-scoring caller-visible face (score descending, the face id ascending as the tiebreak) -- the best face the CALLER can see, never a face on an asset the caller cannot view. 
    // String representativeFaceId
    test('to test the property `representativeFaceId`', () async {
      // TODO
    });

    // URL of the tile's face crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the caller-visible best face's id. 
    // String thumbnail
    test('to test the property `thumbnail`', () async {
      // TODO
    });

  });
}
