# lumemor_api.api.PeopleApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getPersonCluster**](PeopleApi.md#getpersoncluster) | **GET** /people/{clusterId} | One face cluster&#39;s detail as the caller sees it.
[**listPeople**](PeopleApi.md#listpeople) | **GET** /people | The caller&#39;s named people.
[**listUnnamedPeople**](PeopleApi.md#listunnamedpeople) | **GET** /people/unnamed | The caller&#39;s unnamed face clusters, one keyset page at a time.


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

