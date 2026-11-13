import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ClusterDetail
void main() {
  final instance = ClusterDetailBuilder();
  // TODO add properties to the builder and call build()

  group(ClusterDetail, () {
    // Identifier of the face cluster.
    // String clusterId
    test('to test the property `clusterId`', () async {
      // TODO
    });

    // The caller's label of the cluster, or null when the caller has not labeled it (the unnamed state). It is the calling user's label or no label at all: another account's label never shows here. 
    // PersonLabel label
    test('to test the property `label`', () async {
      // TODO
    });

    // The person's birth date (the entity's own attribute, one value for every account), or null when unset. Rendered locale-aware by the client (`Intl.DateTimeFormat`'s numeric year/month/day -- never hand-assembled). 
    // Date birthDate
    test('to test the property `birthDate`', () async {
      // TODO
    });

    // The face this person's avatar renders (the entity's own attribute), or null when none was picked -- the reads then fall back to the cluster's best caller-visible face. 
    // String avatarFaceId
    test('to test the property `avatarFaceId`', () async {
      // TODO
    });

    // The number of DISTINCT live assets the cluster has a face on that the caller can see (photos, not faces). 
    // int photoCount
    test('to test the property `photoCount`', () async {
      // TODO
    });

    // The cluster's faces on assets visible to the caller, strongest score first with the face id as the tiebreak. Faces on other accounts' assets never appear. 
    // BuiltList<FaceDetail> faces
    test('to test the property `faces`', () async {
      // TODO
    });

  });
}
