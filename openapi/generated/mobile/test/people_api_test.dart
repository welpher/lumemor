import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for PeopleApi
void main() {
  final instance = LumemorApi().getPeopleApi();

  group(PeopleApi, () {
    // Serve one face's 256x256 crop thumbnail.
    //
    // The face crop pixels of the people surface: a 256x256 `image/avif` crop of one detected face, derived at query time and never stored on disk. The source pixels are the face's asset's `view` derivative -- the inline serving size of `GET /assets/{id}/thumbnail` -- never the original; the face row's bounding box is in per-mille units of the full image, and the view derivative preserves the original's framing exactly, so the same per-mille values address the derivative's own pixels.  The crop window is pinned: a square whose side is 1.25x the larger of the box's pixel width and height (a 25% margin around the face), centered on the box and clamped to the image bounds (a face larger than the image yields the image's smaller edge); the window then renders to exactly 256x256 pixels -- upscaled when the window is smaller, a zoom rather than a fit.  The permission gate is the face's asset: the caller must see a live, non-component asset -- their own, or one shared to them through an album. Every other state answers the SAME byte-identical 404 PERSON_NOT_FOUND -- an unknown face id, a face on another account's unshared (or a trashed, or hidden) asset, and an asset whose view derivative has not been derived yet (the thumbnail job has not run). Face ids cannot be probed for existence; the web renders its skeleton placeholder until the pixels exist, which is why there is deliberately no separate not-ready code here, unlike the asset thumbnail endpoint.  The response is a private immutable derived asset: `Cache-Control: private, max-age=31536000, immutable`, and a strong `ETag` of `\"face-{faceId}-{updatedAt}\"`, where updatedAt is the face row's change signal in unix seconds -- cluster moves bump it, and re-detection replaces the face row outright. A matching `If-None-Match` answers 304 with the same validators and no body. Conditional requests and byte ranges otherwise follow the Go standard library's `ServeContent` semantics. 
    //
    //Future<Uint8List> getFaceThumbnail(String faceId) async
    test('test getFaceThumbnail', () async {
      // TODO
    });

    // One face cluster's detail as the caller sees it.
    //
    // The detail view of one face cluster: the caller's own label state and the cluster's faces on assets visible to the caller, strongest score first with the face id as the tiebreak.  `label` is null when the caller has not labeled the cluster (the unnamed state). It is the CALLING user's label or no label at all: another account's label never shows here (clusters are global, labels are private per user).  404 has a double meaning by design: the cluster id is unknown, or the cluster exists but none of its faces sits on an asset visible to the caller (the cluster's assets all belong to other accounts and none is shared to them). Both answer the same `PERSON_NOT_FOUND`, byte for byte, so cluster ids cannot be probed for existence -- the same anti-enumeration shape as `ASSET_NOT_FOUND`.  Every `thumbnail` points at the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail`, which serves the face's bounding-box crop of the asset's view-size derivative. 
    //
    //Future<ClusterDetail> getPersonCluster(String clusterId) async
    test('test getPersonCluster', () async {
      // TODO
    });

    // Name one face cluster (the caller's private label).
    //
    // Upserts the calling user's label of one face cluster: the `user_face_labels` row keyed (userId, clusterId) -- the per-user write. The cluster must be visible to the caller (at least one of its faces sits on an asset visible to them -- their own, or one shared to them through an album, so a sharee names the people they see); an unknown or invisible cluster id answers the same byte-identical 404 `PERSON_NOT_FOUND` as every people read (anti-enumeration -- cluster ids cannot be probed).  The name is trimmed and must span 1-64 characters; anything else answers 400 `VALIDATION_FAILED` naming the `name` field.  Duplicate names are ALLOWED, server-side and on purpose: no `NAME_TAKEN` error exists. Families have real duplicates (two grandparents named Grace), so a de-duplication hint is a web-UI suggestion, never a server rule.  Re-labeling an already labeled cluster renames it in place: the hidden switch keeps its state (a rename is not an unhide). The label is the caller's private state -- another account labeling the same cluster writes its own row and neither sees the other's.  The write publishes `person.labeled` on the activity stream with the acting user's id in the payload. 
    //
    //Future<PersonLabelResponse> labelPerson(String clusterId, LabelPersonRequest labelPersonRequest) async
    test('test labelPerson', () async {
      // TODO
    });

    // The caller's named people.
    //
    // The named region of the people surface: every face cluster the caller sees that carries one of the caller's own labels. A cluster is listed here when both hold:  - it contains at least one face on an asset visible to the   caller (an asset the caller can see that is live -- not   trashed, not hidden, no Live Photo component row: their   own, or one shared to them through an album), and - the caller has labeled it with a `user_face_labels` row of   their own.  The per-user-state semantics of the people domain: face clusters are GLOBAL structures of the instance (the same person's faces land in one cluster whatever account uploaded the photos), while names and the hidden switch are PRIVATE per user -- two accounts label the same cluster independently and neither sees the other's label. `faceCount` is honest per caller: it counts only the cluster's faces on caller-visible assets, so the same cluster can show different counts to different accounts.  Hidden people STAY in this list carrying `isHidden` true: the switch is a badge for the web UI, not a filter -- the caller labeled the person and keeps seeing them, flagged.  The list is sorted by name, case-insensitively, with the cluster id as the tiebreak. 
    //
    //Future<BuiltList<PersonSummary>> listPeople() async
    test('test listPeople', () async {
      // TODO
    });

    // The caller's unnamed face clusters, one keyset page at a time.
    //
    // The unnamed region of the people surface: every face cluster that contains at least one face on an asset visible to the caller (an asset the caller can see that is live -- not trashed, not hidden, no Live Photo component row: their own, or one shared to them through an album) and carries NO `user_face_labels` row of the caller. The moment the caller labels a cluster it moves to `GET /people` and out of this list; another account's label never moves it for this caller (clusters are global, labels are private per user).  Pages are keyset-paginated with an opaque cursor over the ordering `faceCount` descending, `clusterId` descending as the tiebreak. The continuation key is the (faceCount, clusterId) pair of a page's last row -- the same row-value predicate pattern the timeline endpoint applies to its (takenAt, id) pair, which keeps pages stable while clusters gain or lose faces. The response carries `nextCursor` exactly when another page exists; the first page omits the `cursor` parameter.  `limit` bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 `VALIDATION_FAILED`. 
    //
    //Future<UnnamedClustersResponse> listUnnamedPeople({ String cursor, int limit }) async
    test('test listUnnamedPeople', () async {
      // TODO
    });

    // Fold 1-50 face clusters into one surviving cluster.
    //
    // One structural merge: every source cluster's faces re-home onto the target cluster, the target's representative face recomputes to the strongest-scoring member of the merged set, and every source cluster dissolves -- with its label rows cascading. A single source is the pairwise fold the cluster detail's \"merge into\" action drives (that person merges into the picked one); more sources are the batch organize path of the unnamed region.  THE HONEST COST, by design: a source cluster's user-given name DIES with the cluster, whatever account gave it. A person merged away cannot keep the name a caller gave the cluster they merged; the target cluster's label persists. The web UI names the target before merging when the caller wants the merged person called something.  The whole merge runs as ONE transaction through the cluster maintenance discipline (representative re-pointing plus dissolution of the vacated clusters -- the same primitives the asset deletion sweep applies), never as bare member moves that would strand a stale representative or an empty cluster.  Rules: every source and the target must be visible to the caller (at least one face on an asset the caller OWNS and that is live) -- merging is cluster curation, and curation is the owner's alone: a sharee who sees a cluster through a shared album reads it and labels it, but a merge from their hand answers the same 404 `PERSON_NOT_FOUND`, so the shared structure never re-shapes under another account's cursor. The target must not be among the sources -- any violation answers the same byte-identical 404 `PERSON_NOT_FOUND` (anti-enumeration). An empty source list or more than 50 source ids (duplicates collapse first) answer 400 `VALIDATION_FAILED`.  The write publishes one `person.merged` event with the surviving cluster id, the dissolved ids comma-joined as `mergedClusterIds`, and the acting user's id. 
    //
    //Future<MergePeopleResponse> mergePeople(MergePeopleRequest mergePeopleRequest) async
    test('test mergePeople', () async {
      // TODO
    });

    // Move one face onto an existing face cluster.
    //
    // One face's move onto an existing target cluster: the tidy-up primitive behind the viewer's face chips. The face must sit on an asset the caller OWNS and that is live, and the target must be caller-visible -- any other state (unknown face, another account's asset, an asset shared to the caller but not owned -- the move is curation, and curation is the owner's alone -- an invisible or unknown target) answers the same byte-identical 404 `PERSON_NOT_FOUND` (anti-enumeration). A face the clustering has not yet assigned to any cluster is not part of any person and answers the same 404.  The move runs as ONE transaction through the cluster maintenance discipline: the target's representative face recomputes to its strongest member, and the vacated source cluster re-homes its representative to the best surviving member -- or dissolves with its label rows cascading when the move empties it (a source cluster's user-given name dies with the cluster, the same honest cost as a merge).  Moving a face onto the cluster it already occupies is an accepted no-op. The write publishes `person.face_moved` with the face id, the cluster it now occupies and the acting user's id. 
    //
    //Future<MoveFaceResponse> moveFace(String faceId, MoveFaceRequest moveFaceRequest) async
    test('test moveFace', () async {
      // TODO
    });

    // Flip the caller's hidden switch of one face cluster.
    //
    // Sets the hidden switch on the calling user's label row of one cluster. The switch rides on the label row (`user_face_labels` carries `name NOT NULL`), so hiding requires the caller to have named the cluster first: hiding an UNLABELED cluster answers 400 `VALIDATION_FAILED` whose `details` carry the message key `person_hide_requires_label` -- the web UI offers the naming dialog instead.  Hidden people STAY in the caller's `GET /people` list carrying `isHidden` true (badge semantics, not a filter). The switch is the caller's private state: hiding a cluster never touches another account's label row.  The cluster must be visible to the caller -- an unknown or invisible cluster id answers the same byte-identical 404 `PERSON_NOT_FOUND` as every people read (anti-enumeration).  The write publishes `person.hidden` on the activity stream in both directions (the payload carries the new `isHidden` value beside the acting user's id). 
    //
    //Future<PersonLabelResponse> setPersonHidden(String clusterId, SetPersonHiddenRequest setPersonHiddenRequest) async
    test('test setPersonHidden', () async {
      // TODO
    });

    // Remove the caller's label of one face cluster.
    //
    // Deletes the calling user's `user_face_labels` row of one cluster: the person returns to the caller's unnamed region (`GET /people/unnamed`) and out of the caller's named list. The cluster itself and every other account's label survive (labels are private per user; only the caller's own row dies).  The write is idempotent once the cluster is visible: naming no label still answers 204. The cluster must be visible to the caller -- an unknown or invisible cluster id answers the same byte-identical 404 `PERSON_NOT_FOUND` as every people read (anti-enumeration).  The write publishes `person.unlabeled` on the activity stream with the acting user's id in the payload. 
    //
    //Future unlabelPerson(String clusterId) async
    test('test unlabelPerson', () async {
      // TODO
    });

  });
}
