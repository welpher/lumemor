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

    // The face the cluster tile renders: the cluster's strongest-scoring face on a caller-visible live asset (score descending, the face id ascending as the tiebreak) -- the best face the CALLER can see, not the cluster's global representative, so a tile never crops out of an asset the caller cannot view. 
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
