import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for PeopleApi
void main() {
  final instance = LumemorApi().getPeopleApi();

  group(PeopleApi, () {
    // Seed a new person from one face.
    //
    // The from-face person creation: the face leaves whatever cluster it occupies and becomes the sole member AND the representative face of a brand-new cluster. Naming acts on a whole cluster, so a face grouped into the wrong person becomes its own person here -- the caller labels the new cluster afterwards like any other.  The refusal discipline is the face move's own: the face must sit on an asset the caller OWNS and that is live -- any other state (unknown face, another account's asset, an asset shared to the caller but not owned, a trashed, archived or Live Photo component asset) answers the same byte-identical 404 `PERSON_NOT_FOUND` (anti-enumeration). A face the clustering has not yet assigned to any cluster is VALID here: the creation seeds the new cluster with it and there is simply no source to maintain.  A call on a face that already holds a cluster is a CREATE, never a no-op: the face re-homes onto a new cluster and the vacated source follows the cluster maintenance discipline -- its representative re-homes to the best surviving member, or the cluster dissolves with its label rows cascading when this creation emptied it (the same honest cost a merge pays; a source cluster's user-given name dies with the cluster).  The whole structural change lands as ONE transaction. The write publishes `person.face_clustered` with the face id, the new cluster id and the acting user's id. 
    //
    //Future<ClusterFaceResponse> clusterFace(String faceId) async
    test('test clusterFace', () async {
      // TODO
    });

    // Add one hand-drawn face to an asset.
    //
    // The manual face: the owner draws a box around a face the detector missed and the drawn box becomes a face row of that asset -- the same rows detection writes, minus a detector (the row's model is null, its score 1.0 -- the caller's own certainty) and minus an embedding (the recognition never saw it, so the face joins no cluster until the caller assigns one). The box rides in the FaceBbox per-mille units; the width and height halves must be at least 1 (a zero-extent box is nothing drawn).  The refusal discipline is the face writes' own: the face must sit on an asset the caller OWNS and that is live -- any other state (unknown asset, another account's asset, an asset shared to the caller but not owned, a trashed, archived or Live Photo component asset) answers the same byte-identical 404 `PERSON_NOT_FOUND` (anti-enumeration). A bbox value outside the 0..1000 bounds, a missing half, or a body that fails to decode answers 400 `VALIDATION_FAILED`.  The write inserts the row and answers it; it publishes NO event: a face with no cluster announces no person change, and the caller's own session converges through the response (the same silence a fresh detection's rows keep until recognition names them). The caller's next step is the assign flow -- `POST /faces/{faceId}/move` or the from-face creation -- which is where the drawn face becomes someone. 
    //
    //Future<FaceCreated> createFace(FaceCreateRequest faceCreateRequest) async
    test('test createFace', () async {
      // TODO
    });

    // Delete this person outright.
    //
    // The person's real delete, distinct from the two softer actions that also remove things today: removing the caller's own label (DELETE /people/{clusterId}/label -- the person returns to that caller's unnamed region) and releasing one face (DELETE /faces/{faceId} -- the face row survives, its membership does not). This write deletes the PERSON: the cluster row AND every one of its face rows go away in one transaction -- the photos lose the person's chips, the people lists lose the person, and the unnamed pool gains NOTHING (the faces are deleted, never unassigned). The asset rows themselves are untouched.  The domain is the instance administrator's: the ML records are shared instance state (clusters are global; a delete reshapes every account's view), so a caller whose role is not admin answers `403 AUTH_FORBIDDEN` with the domain in the message. A member's honest removal action is their own label delete.  The visibility gate is the people surface's own count (the same gate every people write fronts): a cluster the caller cannot see answers the same `PERSON_NOT_FOUND` as an unknown id -- anti-enumeration, byte for byte.  The write publishes `person.deleted` with the cluster id and the acting user's id, so every open list converges. 
    //
    //Future deletePerson(String clusterId) async
    test('test deletePerson', () async {
      // TODO
    });

    // Release one face from its person.
    //
    // The single-face detach: the face's cluster assignment goes NULL, returning it to the not-yet-grouped pool it came from (the state a freshly detected face holds before the clustering assigns it). The face row survives -- it still lists among its asset's faces, now carrying no person -- and any later assign write can re-home it.  The refusal discipline is the face move's own: the face must sit on an asset the caller OWNS and that is live -- any other state (unknown face, another account's asset, an asset shared to the caller but not owned, a trashed, archived or Live Photo component asset) answers the same byte-identical 404 `PERSON_NOT_FOUND` (anti-enumeration). A face the clustering has not yet assigned to any cluster is part of no person and answers the same 404: there is no membership to release.  The source cluster does NOT dissolve unless this detach emptied it; the vacated cluster re-homes its representative to the best surviving member, and an emptied cluster dissolves WITH its label rows cascading -- the same honest cost a merge pays. The whole structural change lands as ONE transaction. The write publishes `person.face_detached` with the face id, the cluster the face left and the acting user's id. 
    //
    //Future detachFace(String faceId) async
    test('test detachFace', () async {
      // TODO
    });

    // Serve one face's 256x256 crop thumbnail.
    //
    // The face crop pixels of the people surface: a 256x256 `image/avif` crop of one detected face, derived at query time and never stored on disk. The source pixels are the face's asset's `view` derivative -- the inline serving size of `GET /assets/{id}/thumbnail` -- never the original; the face row's bounding box is in per-mille units of the full image, and the view derivative preserves the original's framing exactly, so the same per-mille values address the derivative's own pixels.  The crop window is pinned: a square whose side is 1.25x the larger of the box's pixel width and height (a 25% margin around the face), centered on the box and clamped to the image bounds (a face larger than the image yields the image's smaller edge); the window then renders to exactly 256x256 pixels -- upscaled when the window is smaller, a zoom rather than a fit.  The permission gate is the face's asset: the caller must see a live, non-component asset -- their own, or one shared to them through an album. Every other state answers the SAME byte-identical 404 PERSON_NOT_FOUND -- an unknown face id, a face on another account's unshared (or a trashed, or archived) asset, and an asset whose view derivative has not been derived yet (the thumbnail job has not run). Face ids cannot be probed for existence; the web renders its skeleton placeholder until the pixels exist, which is why there is deliberately no separate not-ready code here, unlike the asset thumbnail endpoint.  The response is a private immutable derived asset: `Cache-Control: private, max-age=31536000, immutable`, and a strong `ETag` of `\"face-{faceId}-{updatedAt}\"`, where updatedAt is the face row's change signal in unix seconds -- cluster moves bump it, and re-detection replaces the face row outright. A matching `If-None-Match` answers 304 with the same validators and no body. Conditional requests and byte ranges otherwise follow the Go standard library's `ServeContent` semantics. 
    //
    //Future<Uint8List> getFaceThumbnail(String faceId) async
    test('test getFaceThumbnail', () async {
      // TODO
    });

    // One face cluster's detail as the caller sees it.
    //
    // The detail view of one face cluster: the caller's own label state, the person entity's global attributes and the cluster's faces on assets visible to the caller, strongest score first with the face id as the tiebreak.  `label` is null when the caller has not labeled the cluster (the unnamed state). It is the CALLING user's label or no label at all: another account's label never shows here (clusters are global, labels are private per user).  `birthDate` and `avatarFaceId` are the PERSON ENTITY's attributes, not the caller's: one birthday per person, every account reading the same value (a family agrees on the person's birthday), and one avatar face per cluster whatever account picked it. Both are independent of naming -- an unnamed cluster carries them too.  `photoCount` is the count the detail bar renders: the number of DISTINCT live assets the cluster has a face on that the caller can see (photos, not faces).  404 has a double meaning by design: the cluster id is unknown, or the cluster exists but none of its faces sits on an asset visible to the caller (the cluster's assets all belong to other accounts and none is shared to them). Both answer the same `PERSON_NOT_FOUND`, byte for byte, so cluster ids cannot be probed for existence -- the same anti-enumeration shape as `ASSET_NOT_FOUND`.  Every `thumbnail` points at the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail`, which serves the face's bounding-box crop of the asset's view-size derivative. 
    //
    //Future<ClusterDetail> getPersonCluster(String clusterId) async
    test('test getPersonCluster', () async {
      // TODO
    });

    // Name one face cluster (the caller's private label).
    //
    // Upserts the calling user's label of one face cluster: the `user_face_labels` row keyed (userId, clusterId) -- the per-user write. The cluster must be visible to the caller (at least one of its faces sits on an asset visible to them -- their own, or one shared to them through an album, so a sharee names the people they see); an unknown or invisible cluster id answers the same byte-identical 404 `PERSON_NOT_FOUND` as every people read (anti-enumeration -- cluster ids cannot be probed).  The name is trimmed and must span 1-64 characters; anything else answers 400 `VALIDATION_FAILED` naming the `name` field.  Duplicate names are ALLOWED, server-side and on purpose: no `NAME_TAKEN` error exists. Families have real duplicates (two grandparents named Grace), so a de-duplication hint is a web-UI suggestion, never a server rule.  Re-labeling an already labeled cluster renames it in place. The label is the caller's private state -- another account labeling the same cluster writes its own row and neither sees the other's -- and it is the ONLY private state of the people domain: the person entity's birthday and avatar ride the cluster's own columns, shared by every account.  The write publishes `person.labeled` on the activity stream with the acting user's id in the payload. 
    //
    //Future<PersonLabelResponse> labelPerson(String clusterId, LabelPersonRequest labelPersonRequest) async
    test('test labelPerson', () async {
      // TODO
    });

    // The caller's named people.
    //
    // The named region of the people surface: every face cluster the caller sees that carries one of the caller's own labels. A cluster is listed here when both hold:  - it contains at least one face on an asset visible to the   caller (an asset the caller can see that is live -- not   trashed, not archived, no Live Photo component row: their   own, or one shared to them through an album), and - the caller has labeled it with a `user_face_labels` row of   their own.  The per-user-state semantics of the people domain: face clusters are GLOBAL structures of the instance (the same person's faces land in one cluster whatever account uploaded the photos), while names are PRIVATE per user -- two accounts label the same cluster independently and neither sees the other's label. The people domain has exactly two states: named (this list) and unnamed (`GET /people/unnamed`). There is no third state: hiding a person retired whole (the per-user switch, its badge and its events are gone; removing the caller's name -- `DELETE /people/{clusterId}/label` -- is the way a person leaves this list).  `photoCount` is honest per caller: the number of DISTINCT live assets the cluster has a face on that the caller can see, so the same cluster can show different counts to different accounts. It counts photos, not faces -- two faces of one person in one photo are one photo.  The list is sorted by name, case-insensitively, with the cluster id as the tiebreak. 
    //
    //Future<BuiltList<PersonSummary>> listPeople() async
    test('test listPeople', () async {
      // TODO
    });

    // The caller's unnamed face clusters, one keyset page at a time.
    //
    // The unnamed region of the people surface: every face cluster that contains at least one face on an asset visible to the caller (an asset the caller can see that is live -- not trashed, not archived, no Live Photo component row: their own, or one shared to them through an album) and carries NO `user_face_labels` row of the caller. The moment the caller labels a cluster it moves to `GET /people` and out of this list; another account's label never moves it for this caller (clusters are global, labels are private per user).  Pages are keyset-paginated with an opaque cursor over the ordering `faceCount` descending, `clusterId` descending as the tiebreak. The continuation key is the (faceCount, clusterId) pair of a page's last row -- the same row-value predicate pattern the timeline endpoint applies to its (takenAt, id) pair, which keeps pages stable while clusters gain or lose faces. The response carries `nextCursor` exactly when another page exists; the first page omits the `cursor` parameter.  `totalCount` is the caller's whole unnamed region -- every cluster this listing would walk -- so a page never has to be walked to its end to count the region (the people list's unnamed entry badge and this page's own bar count read it).  `limit` bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 `VALIDATION_FAILED`. 
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
    // One face's move onto an existing target cluster: the tidy-up primitive behind the viewer's face chips. The face must sit on an asset the caller OWNS and that is live, and the target must be caller-visible -- any other state (unknown face, another account's asset, an asset shared to the caller but not owned -- the move is curation, and curation is the owner's alone -- an invisible or unknown target) answers the same byte-identical 404 `PERSON_NOT_FOUND` (anti-enumeration). A face the clustering has not yet assigned to any cluster is a valid source: it simply joins the target (there is no source cluster to maintain).  The move runs as ONE transaction through the cluster maintenance discipline: the target's representative face recomputes to its strongest member, and the vacated source cluster re-homes its representative to the best surviving member -- or dissolves with its label rows cascading when the move empties it (a source cluster's user-given name dies with the cluster, the same honest cost as a merge).  Moving a face onto the cluster it already occupies is an accepted no-op. The write publishes `person.face_moved` with the face id, the cluster it now occupies and the acting user's id. 
    //
    //Future<MoveFaceResponse> moveFace(String faceId, MoveFaceRequest moveFaceRequest) async
    test('test moveFace', () async {
      // TODO
    });

    // Rewrite one face's bounding box.
    //
    // The face-box editor's write: the caller adjusts a face's bounding box on the photo (drag to move, drag a corner to resize) and the PATCH stores the adjusted box. The body carries the box NORMALIZED to the image (0-1, four values required -- x/y the top-left corner, width/height the extents); the server stores it in the faces table's own per-mille units (the same numbers the crop endpoint and the on-photo face-box layer address).  The box must stay in frame: every coordinate and extent within 0-1 with x + width <= 1 and y + height <= 1, and neither extent below 0.01 (a box smaller than a hundredth of the image is nothing worth storing). Any other body answers `400 VALIDATION_FAILED`.  The guard is the face writes' own: the face must sit on an asset the caller OWNS and that is live -- an unknown face, another account's asset, or a trashed, archived or Live Photo component asset answers the same byte-identical 404 `PERSON_NOT_FOUND` (anti-enumeration).  On success the face's crop is recomputed from the new box -- the crop endpoint derives it at query time, so the change converges through the face row's own `updated_at` (the crop ETag's change signal). The write publishes `person.face_box_updated` with the face id and the acting user's id. 
    //
    //Future<FaceBoxPatchResponse> patchFaceBox(String faceId, FaceBoxPatchRequest faceBoxPatchRequest) async
    test('test patchFaceBox', () async {
      // TODO
    });

    // Pick the face one person's avatar renders.
    //
    // Writes the PERSON ENTITY's avatar face: the cluster's chosen face, read by every account (the person card on `GET /people` and the detail's profile row both render it). Like the birth date it is not the caller's private state, and it does not depend on naming -- an unnamed cluster carries an avatar too.  The face must belong to the cluster and sit on an asset visible to the caller (the avatar picker offers exactly the cluster's caller-visible faces); anything else -- a face of another cluster, an invisible face, an unknown cluster -- answers the same byte-identical 404 `PERSON_NOT_FOUND` as every people write (anti-enumeration).  The pick is advisory to the clustering, never a structural write: the representative face the clustering maintains for merges and face moves is untouched, and a face that later leaves the cluster (a move, a detach, a purged asset) simply stops counting -- the read falls back to the cluster's best-scoring caller-visible face.  The write publishes `person.avatar_updated` on the activity stream with the acting user's id in the payload. 
    //
    //Future<PersonAvatarResponse> setPersonAvatar(String clusterId, SetPersonAvatarRequest setPersonAvatarRequest) async
    test('test setPersonAvatar', () async {
      // TODO
    });

    // Set or clear one person's birth date.
    //
    // Writes the PERSON ENTITY's birth date: one value per cluster, read by every account (`GET /people/{clusterId}` serves it to all of them). It is not the caller's private state -- a family agrees on the person's birthday -- and it is independent of naming: an unnamed cluster carries a birth date too, and `person_birthdate` (the profile row) renders it as soon as it is set.  The use of the value is honest: the person's age at the time each photo was taken, and the birthday memories the product will build later. Nothing else reads it.  Any logged-in user who can see the cluster may write it (at least one of the cluster's faces sits on an asset visible to them); an unknown or invisible cluster id answers the same byte-identical 404 `PERSON_NOT_FOUND` as every people read (anti-enumeration -- cluster ids cannot be probed).  `birthDate` is the wire's `date` form (`YYYY-MM-DD`): a null clears the value. A malformed date answers 400 `VALIDATION_FAILED` naming the `birthDate` field.  The write publishes `person.birthdate_updated` on the activity stream with the acting user's id in the payload (the person entity's attributes are shared state, so every open view of the person converges). 
    //
    //Future<PersonBirthdateResponse> setPersonBirthdate(String clusterId, SetPersonBirthdateRequest setPersonBirthdateRequest) async
    test('test setPersonBirthdate', () async {
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
