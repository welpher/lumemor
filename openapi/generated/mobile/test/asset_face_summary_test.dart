import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for AssetFaceSummary
void main() {
  final instance = AssetFaceSummaryBuilder();
  // TODO add properties to the builder and call build()

  group(AssetFaceSummary, () {
    // Identifier of the face row.
    // String id
    test('to test the property `id`', () async {
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

    // Identifier of the face's cluster, or null before the clustering assigned one (a face the recognition run has not grouped yet). 
    // String clusterId
    test('to test the property `clusterId`', () async {
      // TODO
    });

    // The caller's own label of the face's cluster, or null when the caller has not labeled it (the unnamed state) -- and null for a clusterless face, which has no cluster to label. 
    // PersonLabel label
    test('to test the property `label`', () async {
      // TODO
    });

    // The person's birth date (the entity's own attribute), or null when unset or the face has no cluster. The client computes the capture-time age from it and the asset's `takenAt` -- the birth date's whole purpose beside the birthday memories. 
    // Date birthDate
    test('to test the property `birthDate`', () async {
      // TODO
    });

  });
}
