import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for PeopleApi
void main() {
  final instance = LumemorApi().getPeopleApi();

  group(PeopleApi, () {
    // One face cluster's detail as the caller sees it.
    //
    // The detail view of one face cluster: the caller's own label state and the cluster's faces on assets visible to the caller, strongest score first with the face id as the tiebreak.  `label` is null when the caller has not labeled the cluster (the unnamed state). It is the CALLING user's label or no label at all: another account's label never shows here (B4 -- clusters are global, labels are private per user).  404 has a double meaning by design: the cluster id is unknown, or the cluster exists but none of its faces sits on an asset visible to the caller (before library sharing: the cluster's assets all belong to other accounts). Both answer the same `PERSON_NOT_FOUND`, byte for byte, so cluster ids cannot be probed for existence -- the same anti-enumeration shape as `ASSET_NOT_FOUND`.  Every `thumbnail` points at the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail`, which serves the face's bounding-box crop of the asset's view-size derivative. 
    //
    //Future<ClusterDetail> getPersonCluster(String clusterId) async
    test('test getPersonCluster', () async {
      // TODO
    });

    // The caller's named people.
    //
    // The named region of the people surface: every face cluster the caller sees that carries one of the caller's own labels. A cluster is listed here when both hold:  - it contains at least one face on an asset visible to the   caller (before library sharing exists: an asset the caller   owns that is live -- not trashed, not hidden, no Live Photo   component row), and - the caller has labeled it with a `user_face_labels` row of   their own.  The B4 semantics of the people domain: face clusters are GLOBAL structures of the instance (the same person's faces land in one cluster whatever account uploaded the photos), while names and the hidden switch are PRIVATE per user -- two accounts label the same cluster independently and neither sees the other's label. `faceCount` is honest per caller: it counts only the cluster's faces on caller-visible assets, so the same cluster can show different counts to different accounts.  Hidden people STAY in this list carrying `isHidden` true: the switch is a badge for the web UI, not a filter -- the caller labeled the person and keeps seeing them, flagged.  The list is sorted by name, case-insensitively, with the cluster id as the tiebreak. 
    //
    //Future<BuiltList<PersonSummary>> listPeople() async
    test('test listPeople', () async {
      // TODO
    });

    // The caller's unnamed face clusters, one keyset page at a time.
    //
    // The unnamed region of the people surface: every face cluster that contains at least one face on an asset visible to the caller (before library sharing exists: an asset the caller owns that is live -- not trashed, not hidden, no Live Photo component row) and carries NO `user_face_labels` row of the caller. The moment the caller labels a cluster it moves to `GET /people` and out of this list; another account's label never moves it for this caller (B4: clusters are global, labels are private per user).  Pages are keyset-paginated with an opaque cursor over the ordering `faceCount` descending, `clusterId` descending as the tiebreak. The continuation key is the (faceCount, clusterId) pair of a page's last row -- the same row-value predicate pattern the timeline endpoint applies to its (takenAt, id) pair, which keeps pages stable while clusters gain or lose faces. The response carries `nextCursor` exactly when another page exists; the first page omits the `cursor` parameter.  `limit` bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 `VALIDATION_FAILED`. 
    //
    //Future<UnnamedClustersResponse> listUnnamedPeople({ String cursor, int limit }) async
    test('test listUnnamedPeople', () async {
      // TODO
    });

  });
}
