# lumemor_api.api.PeopleApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getFaceThumbnail**](PeopleApi.md#getfacethumbnail) | **GET** /faces/{faceId}/thumbnail | Serve one face&#39;s 256x256 crop thumbnail.
[**getPersonCluster**](PeopleApi.md#getpersoncluster) | **GET** /people/{clusterId} | One face cluster&#39;s detail as the caller sees it.
[**labelPerson**](PeopleApi.md#labelperson) | **PUT** /people/{clusterId}/label | Name one face cluster (the caller&#39;s private label).
[**listPeople**](PeopleApi.md#listpeople) | **GET** /people | The caller&#39;s named people.
[**listUnnamedPeople**](PeopleApi.md#listunnamedpeople) | **GET** /people/unnamed | The caller&#39;s unnamed face clusters, one keyset page at a time.
[**mergePeople**](PeopleApi.md#mergepeople) | **POST** /people/merge | Fold 2-50 face clusters into one surviving cluster.
[**moveFace**](PeopleApi.md#moveface) | **POST** /faces/{faceId}/move | Move one face onto an existing face cluster.
[**setPersonHidden**](PeopleApi.md#setpersonhidden) | **PUT** /people/{clusterId}/hidden | Flip the caller&#39;s hidden switch of one face cluster.
[**unlabelPerson**](PeopleApi.md#unlabelperson) | **DELETE** /people/{clusterId}/label | Remove the caller&#39;s label of one face cluster.


# **getFaceThumbnail**
> Uint8List getFaceThumbnail(faceId)

Serve one face's 256x256 crop thumbnail.

The face crop pixels of the people surface: a 256x256 `image/avif` crop of one detected face, derived at query time and never stored on disk. The source pixels are the face's asset's `view` derivative -- the inline serving size of `GET /assets/{id}/thumbnail` -- never the original; the face row's bounding box is in per-mille units of the full image, and the view derivative preserves the original's framing exactly, so the same per-mille values address the derivative's own pixels.  The crop window is pinned: a square whose side is 1.25x the larger of the box's pixel width and height (a 25% margin around the face), centered on the box and clamped to the image bounds (a face larger than the image yields the image's smaller edge); the window then renders to exactly 256x256 pixels -- upscaled when the window is smaller, a zoom rather than a fit.  The permission gate is the face's asset: the caller must own a live, visible, non-component asset. Every other state answers the SAME byte-identical 404 PERSON_NOT_FOUND -- an unknown face id, a face on another account's (or a trashed, or hidden) asset, and an asset whose view derivative has not been derived yet (the thumbnail job has not run). Face ids cannot be probed for existence; the web renders its skeleton placeholder until the pixels exist, which is why there is deliberately no separate not-ready code here, unlike the asset thumbnail endpoint.  The response is a private immutable derived asset: `Cache-Control: private, max-age=31536000, immutable`, and a strong `ETag` of `\"face-{faceId}-{updatedAt}\"`, where updatedAt is the face row's change signal in unix seconds -- cluster moves bump it, and re-detection replaces the face row outright. A matching `If-None-Match` answers 304 with the same validators and no body. Conditional requests and byte ranges otherwise follow the Go standard library's `ServeContent` semantics. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPeopleApi();
final String faceId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the face to crop.

try {
    final response = api.getFaceThumbnail(faceId);
    print(response);
} on DioException catch (e) {
    print('Exception when calling PeopleApi->getFaceThumbnail: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **faceId** | **String**| Identifier of the face to crop. | 

### Return type

[**Uint8List**](Uint8List.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: image/avif, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getPersonCluster**
> ClusterDetail getPersonCluster(clusterId)

One face cluster's detail as the caller sees it.

The detail view of one face cluster: the caller's own label state and the cluster's faces on assets visible to the caller, strongest score first with the face id as the tiebreak.  `label` is null when the caller has not labeled the cluster (the unnamed state). It is the CALLING user's label or no label at all: another account's label never shows here (B4 -- clusters are global, labels are private per user).  404 has a double meaning by design: the cluster id is unknown, or the cluster exists but none of its faces sits on an asset visible to the caller (before library sharing: the cluster's assets all belong to other accounts). Both answer the same `PERSON_NOT_FOUND`, byte for byte, so cluster ids cannot be probed for existence -- the same anti-enumeration shape as `ASSET_NOT_FOUND`.  Every `thumbnail` points at the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail`, which serves the face's bounding-box crop of the asset's view-size derivative. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPeopleApi();
final String clusterId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the face cluster.

try {
    final response = api.getPersonCluster(clusterId);
    print(response);
} on DioException catch (e) {
    print('Exception when calling PeopleApi->getPersonCluster: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **clusterId** | **String**| Identifier of the face cluster. | 

### Return type

[**ClusterDetail**](ClusterDetail.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **labelPerson**
> PersonLabelResponse labelPerson(clusterId, labelPersonRequest)

Name one face cluster (the caller's private label).

Upserts the calling user's label of one face cluster: the `user_face_labels` row keyed (userId, clusterId) -- the B4 write. The cluster must be visible to the caller (at least one of its faces sits on an asset visible to them); an unknown or invisible cluster id answers the same byte-identical 404 `PERSON_NOT_FOUND` as every people read (anti-enumeration -- cluster ids cannot be probed).  The name is trimmed and must span 1-64 characters; anything else answers 400 `VALIDATION_FAILED` naming the `name` field.  Duplicate names are ALLOWED, server-side and on purpose: no `NAME_TAKEN` error exists. Families have real duplicates (two grandparents named Grace), so a de-duplication hint is a web-UI suggestion, never a server rule.  Re-labeling an already labeled cluster renames it in place: the hidden switch keeps its state (a rename is not an unhide). The label is the caller's private state -- another account labeling the same cluster writes its own row and neither sees the other's (B4).  The write publishes `person.labeled` on the activity stream with the acting user's id in the payload. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPeopleApi();
final String clusterId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the face cluster.
final LabelPersonRequest labelPersonRequest = {"name":"Ada"}; // LabelPersonRequest | 

try {
    final response = api.labelPerson(clusterId, labelPersonRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling PeopleApi->labelPerson: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **clusterId** | **String**| Identifier of the face cluster. | 
 **labelPersonRequest** | [**LabelPersonRequest**](LabelPersonRequest.md)|  | 

### Return type

[**PersonLabelResponse**](PersonLabelResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listPeople**
> BuiltList<PersonSummary> listPeople()

The caller's named people.

The named region of the people surface: every face cluster the caller sees that carries one of the caller's own labels. A cluster is listed here when both hold:  - it contains at least one face on an asset visible to the   caller (before library sharing exists: an asset the caller   owns that is live -- not trashed, not hidden, no Live Photo   component row), and - the caller has labeled it with a `user_face_labels` row of   their own.  The B4 semantics of the people domain: face clusters are GLOBAL structures of the instance (the same person's faces land in one cluster whatever account uploaded the photos), while names and the hidden switch are PRIVATE per user -- two accounts label the same cluster independently and neither sees the other's label. `faceCount` is honest per caller: it counts only the cluster's faces on caller-visible assets, so the same cluster can show different counts to different accounts.  Hidden people STAY in this list carrying `isHidden` true: the switch is a badge for the web UI, not a filter -- the caller labeled the person and keeps seeing them, flagged.  The list is sorted by name, case-insensitively, with the cluster id as the tiebreak. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPeopleApi();

try {
    final response = api.listPeople();
    print(response);
} on DioException catch (e) {
    print('Exception when calling PeopleApi->listPeople: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**BuiltList&lt;PersonSummary&gt;**](PersonSummary.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listUnnamedPeople**
> UnnamedClustersResponse listUnnamedPeople(cursor, limit)

The caller's unnamed face clusters, one keyset page at a time.

The unnamed region of the people surface: every face cluster that contains at least one face on an asset visible to the caller (before library sharing exists: an asset the caller owns that is live -- not trashed, not hidden, no Live Photo component row) and carries NO `user_face_labels` row of the caller. The moment the caller labels a cluster it moves to `GET /people` and out of this list; another account's label never moves it for this caller (B4: clusters are global, labels are private per user).  Pages are keyset-paginated with an opaque cursor over the ordering `faceCount` descending, `clusterId` descending as the tiebreak. The continuation key is the (faceCount, clusterId) pair of a page's last row -- the same row-value predicate pattern the timeline endpoint applies to its (takenAt, id) pair, which keeps pages stable while clusters gain or lose faces. The response carries `nextCursor` exactly when another page exists; the first page omits the `cursor` parameter.  `limit` bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 `VALIDATION_FAILED`. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPeopleApi();
final String cursor = cursor_example; // String | The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
final int limit = 60; // int | Page size: the number of cluster summaries one response carries at most. Defaults to 60; values below 1 or over 200 answer 400. 

try {
    final response = api.listUnnamedPeople(cursor, limit);
    print(response);
} on DioException catch (e) {
    print('Exception when calling PeopleApi->listUnnamedPeople: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **cursor** | **String**| The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients).  | [optional] 
 **limit** | **int**| Page size: the number of cluster summaries one response carries at most. Defaults to 60; values below 1 or over 200 answer 400.  | [optional] [default to 60]

### Return type

[**UnnamedClustersResponse**](UnnamedClustersResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **mergePeople**
> MergePeopleResponse mergePeople(mergePeopleRequest)

Fold 2-50 face clusters into one surviving cluster.

One structural merge: every source cluster's faces re-home onto the target cluster, the target's representative face recomputes to the strongest-scoring member of the merged set, and every source cluster dissolves -- with its label rows cascading.  THE HONEST COST, by design: a source cluster's user-given name DIES with the cluster, whatever account gave it. A person merged away cannot keep the name a caller gave the cluster they merged; the target cluster's label persists. The web UI names the target before merging when the caller wants the merged person called something.  The whole merge runs as ONE transaction through the cluster maintenance discipline (representative re-pointing plus dissolution of the vacated clusters -- the same primitives the asset deletion sweep applies), never as bare member moves that would strand a stale representative or an empty cluster.  Rules: every source and the target must be visible to the caller (at least one face on an asset visible to them), and the target must not be among the sources -- any violation answers the same byte-identical 404 `PERSON_NOT_FOUND` (anti-enumeration). Fewer than 2 or more than 50 source ids (duplicates collapse first) answer 400 `VALIDATION_FAILED`.  The write publishes one `person.merged` event with the surviving cluster id, the dissolved ids comma-joined as `mergedClusterIds`, and the acting user's id. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPeopleApi();
final MergePeopleRequest mergePeopleRequest = {"sourceClusterIds":["0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f76","0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f78"],"targetClusterId":"0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70"}; // MergePeopleRequest | 

try {
    final response = api.mergePeople(mergePeopleRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling PeopleApi->mergePeople: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **mergePeopleRequest** | [**MergePeopleRequest**](MergePeopleRequest.md)|  | 

### Return type

[**MergePeopleResponse**](MergePeopleResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **moveFace**
> MoveFaceResponse moveFace(faceId, moveFaceRequest)

Move one face onto an existing face cluster.

One face's move onto an existing target cluster: the tidy-up primitive behind the viewer's face chips. The face must sit on an asset visible to the caller and the target must be visible too -- any other state (unknown face, foreign asset, invisible or unknown target) answers the same byte-identical 404 `PERSON_NOT_FOUND` (anti-enumeration). A face the clustering has not yet assigned to any cluster is not part of any person and answers the same 404.  The move runs as ONE transaction through the cluster maintenance discipline: the target's representative face recomputes to its strongest member, and the vacated source cluster re-homes its representative to the best surviving member -- or dissolves with its label rows cascading when the move empties it (a source cluster's user-given name dies with the cluster, the same honest cost as a merge).  Moving a face onto the cluster it already occupies is an accepted no-op. The write publishes `person.face_moved` with the face id, the cluster it now occupies and the acting user's id. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPeopleApi();
final String faceId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the face to move.
final MoveFaceRequest moveFaceRequest = {"targetClusterId":"0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70"}; // MoveFaceRequest | 

try {
    final response = api.moveFace(faceId, moveFaceRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling PeopleApi->moveFace: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **faceId** | **String**| Identifier of the face to move. | 
 **moveFaceRequest** | [**MoveFaceRequest**](MoveFaceRequest.md)|  | 

### Return type

[**MoveFaceResponse**](MoveFaceResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **setPersonHidden**
> PersonLabelResponse setPersonHidden(clusterId, setPersonHiddenRequest)

Flip the caller's hidden switch of one face cluster.

Sets the hidden switch on the calling user's label row of one cluster. The switch rides on the label row (`user_face_labels` carries `name NOT NULL`), so hiding requires the caller to have named the cluster first: hiding an UNLABELED cluster answers 400 `VALIDATION_FAILED` whose `details` carry the message key `person_hide_requires_label` -- the web UI offers the naming dialog instead.  Hidden people STAY in the caller's `GET /people` list carrying `isHidden` true (badge semantics, not a filter). The switch is the caller's private state: hiding a cluster never touches another account's label row (B4).  The cluster must be visible to the caller -- an unknown or invisible cluster id answers the same byte-identical 404 `PERSON_NOT_FOUND` as every people read (anti-enumeration).  The write publishes `person.hidden` on the activity stream in both directions (the payload carries the new `isHidden` value beside the acting user's id). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPeopleApi();
final String clusterId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the face cluster.
final SetPersonHiddenRequest setPersonHiddenRequest = {"isHidden":true}; // SetPersonHiddenRequest | 

try {
    final response = api.setPersonHidden(clusterId, setPersonHiddenRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling PeopleApi->setPersonHidden: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **clusterId** | **String**| Identifier of the face cluster. | 
 **setPersonHiddenRequest** | [**SetPersonHiddenRequest**](SetPersonHiddenRequest.md)|  | 

### Return type

[**PersonLabelResponse**](PersonLabelResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **unlabelPerson**
> unlabelPerson(clusterId)

Remove the caller's label of one face cluster.

Deletes the calling user's `user_face_labels` row of one cluster: the person returns to the caller's unnamed region (`GET /people/unnamed`) and out of the caller's named list. The cluster itself and every other account's label survive (B4 -- labels are private per user; only the caller's own row dies).  The write is idempotent once the cluster is visible: naming no label still answers 204. The cluster must be visible to the caller -- an unknown or invisible cluster id answers the same byte-identical 404 `PERSON_NOT_FOUND` as every people read (anti-enumeration).  The write publishes `person.unlabeled` on the activity stream with the acting user's id in the payload. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPeopleApi();
final String clusterId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the face cluster.

try {
    api.unlabelPerson(clusterId);
} on DioException catch (e) {
    print('Exception when calling PeopleApi->unlabelPerson: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **clusterId** | **String**| Identifier of the face cluster. | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

