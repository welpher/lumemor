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

    // The face the person card renders: the cluster's strongest-scoring face on a caller-visible live asset (score descending, the face id ascending as the tiebreak) -- the best face the CALLER can see, not the cluster's global representative, so a card never crops out of an asset the caller cannot view. The field is the thumbnail's face id. 
    // String representativeFaceId
    test('to test the property `representativeFaceId`', () async {
      // TODO
    });

    // URL of the card's face crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the caller-visible best face's id. 
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
