# lumemor_api.api.TagsApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**addTagAssets**](TagsApi.md#addtagassets) | **POST** /tags/{tagId}/assets | Tag assets in one batch.
[**createTag**](TagsApi.md#createtag) | **POST** /tags | Create one tag, idempotently.
[**deleteTag**](TagsApi.md#deletetag) | **DELETE** /tags/{tagId} | Delete one tag.
[**listTags**](TagsApi.md#listtags) | **GET** /tags | The caller&#39;s tags, the search vocabulary.
[**removeTagAssets**](TagsApi.md#removetagassets) | **DELETE** /tags/{tagId}/assets | Untag assets in one batch.
[**renameTag**](TagsApi.md#renametag) | **PATCH** /tags/{tagId} | Rename one tag.


# **addTagAssets**
> AddTagAssetsResponse addTagAssets(tagId, tagAssetsRequest)

Tag assets in one batch.

Adds the caller's tag to a list of at most 200 asset ids with the batch surface's set semantics: one request, one honest report `{added, skipped}`.  Skip semantics (the missing-ids discipline, the album-items precedent widened to the union): every id must address an asset the CALLER CAN SEE -- the album union (the caller's own sync and directory albums plus every album shared to them), the same resolution the favorites arms of the batch endpoint apply. The union deliberately carries tags: a sharee tags what they see with THEIR OWN tag, and the owner's tag on the same asset stays untouched (B4 -- the assignment rows are per-user). An unknown id, another account's unshared asset, and any id the caller cannot see are silently skipped, never errors. An id that already carries this tag is skipped too: assignment is a set, the add is idempotent. NO live-state filter applies to the resolution: the assignment row is per-user state written whatever the asset's live state (a trashed or hidden asset carries the tag underneath the view's filters and the count returns with the restore or unhide, exactly like a favorite row). An all-skipped request still answers 200 with `added` 0. Repeated ids collapse to one application; the counts cover the distinct ids.  The tag must be the caller's own: an unknown id and another account's tag answer the same 404 `TAG_NOT_FOUND`. The write publishes nothing: the only view that changes is the caller's own tags, which the response itself reports. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getTagsApi();
final String tagId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the tag.
final TagAssetsRequest tagAssetsRequest = {"assetIds":["0192f6a1-7b8c-9d0e-a1b2-3c4d5e6f7091","0192f6a1-7b8c-9d0e-a1b2-3c4d5e6f7092","0192f6a1-7b8c-9d0e-a1b2-3c4d5e6f7093"]}; // TagAssetsRequest | 

try {
    final response = api.addTagAssets(tagId, tagAssetsRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling TagsApi->addTagAssets: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tagId** | **String**| Identifier of the tag. | 
 **tagAssetsRequest** | [**TagAssetsRequest**](TagAssetsRequest.md)|  | 

### Return type

[**AddTagAssetsResponse**](AddTagAssetsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **createTag**
> TagSummary createTag(tagNameRequest)

Create one tag, idempotently.

Creates one tag owned by the caller. The name is trimmed server-side and must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte names stay within the bound at the same length); anything else answers 400 `VALIDATION_FAILED`.  Creation is IDEMPOTENT, the assignment face's set semantics applied to the vocabulary: a name the caller already owns converges on the tag that already exists. A fresh name answers 201 with the new row; a name that already exists answers 200 with the EXISTING tag -- the same id, nothing written, `updatedAt` untouched -- never a 409, because the vocabulary is a set and the duplicate ask is the same ask. Two concurrent creators converge on one row the same way (the unique constraint is the race guard). Names are distinct when their characters differ; the duplicate check is the constraint's own equality.  The response is the tag's summary exactly as `GET /tags` lists it. The write publishes nothing: tags are per-user state, the only view that changes is the caller's own, and the response itself carries it. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getTagsApi();
final TagNameRequest tagNameRequest = {"name":"Beach"}; // TagNameRequest | 

try {
    final response = api.createTag(tagNameRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling TagsApi->createTag: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tagNameRequest** | [**TagNameRequest**](TagNameRequest.md)|  | 

### Return type

[**TagSummary**](TagSummary.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteTag**
> deleteTag(tagId)

Delete one tag.

Deletes the caller's tag and every assignment row with it: the schema's `ON DELETE CASCADE` removes the asset_tags rows in the same statement, so a deleted tag leaves no orphans behind -- on any account's view, because the rows were the caller's own to begin with. The answer is 204 with no body.  Repeating the request answers 404 `TAG_NOT_FOUND`: the tag is gone exactly like an unknown id (the album delete's discipline -- a resource delete, not a set removal). The tag must be the caller's own; another account's tag folds into the same single-shape 404. The write publishes nothing (per-user state, the create's reasoning). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getTagsApi();
final String tagId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the tag.

try {
    api.deleteTag(tagId);
} on DioException catch (e) {
    print('Exception when calling TagsApi->deleteTag: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tagId** | **String**| Identifier of the tag. | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listTags**
> BuiltList<TagSummary> listTags()

The caller's tags, the search vocabulary.

Every tag the calling user owns, one array sorted by name case-insensitively with the tag id as the deterministic tiebreak (the named-people listing's own ordering discipline): the vocabulary the search's tag facet, the batch bar's \"add tag\" and the info panel's tag section consume. Unpaginated like the places listing -- a family instance's vocabulary is a bounded set, one document.  Tags are per-user state (B4, the favorites' privacy shape): the tags' owner is always the caller, and no tag, name or count of another account ever surfaces here. The listing carries the vocabulary in full -- a tag whose every member is trashed or hidden still lists, with its honest count of zero (the assignment rows survive, exactly like a favorite row, and the count returns with the restore or unhide).  `assetCount` counts the tag's assignment rows joined to CALLER-VISIBLE LIVE assets: the album union (the caller's own sync and directory albums plus every album shared to them) and the timeline's live filters -- not trashed, not hidden, no Live Photo component -- the honest-counts discipline of the people and places surfaces. A sharee's count covers the shared members they can see; a revoked share retracts them while the assignment rows survive underneath. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getTagsApi();

try {
    final response = api.listTags();
    print(response);
} on DioException catch (e) {
    print('Exception when calling TagsApi->listTags: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**BuiltList&lt;TagSummary&gt;**](TagSummary.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **removeTagAssets**
> RemoveTagAssetsResponse removeTagAssets(tagId, tagAssetsRequest)

Untag assets in one batch.

Removes the caller's assignment rows of one tag: `{removed, skipped}`, the same batch semantics as the add. An id that carries no assignment of this tag -- never was, or already removed -- is skipped; an all-skipped request still answers 200 with `removed` 0. Repeated ids collapse; the counts cover the distinct ids.  The removal carries no visibility predicate: it tidies the caller's own set, so the rows leave whatever the assets' live state (a trashed member's assignment row still leaves, the honest cleanup of the set itself). The tag must be the caller's own: an unknown id and another account's tag answer the same 404 `TAG_NOT_FOUND`. The write publishes nothing (per-user state, the add's reasoning). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getTagsApi();
final String tagId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the tag.
final TagAssetsRequest tagAssetsRequest = {"assetIds":["0192f6a1-7b8c-9d0e-a1b2-3c4d5e6f7091","0192f6a1-7b8c-9d0e-a1b2-3c4d5e6f7092"]}; // TagAssetsRequest | 

try {
    final response = api.removeTagAssets(tagId, tagAssetsRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling TagsApi->removeTagAssets: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tagId** | **String**| Identifier of the tag. | 
 **tagAssetsRequest** | [**TagAssetsRequest**](TagAssetsRequest.md)|  | 

### Return type

[**RemoveTagAssetsResponse**](RemoveTagAssetsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **renameTag**
> TagSummary renameTag(tagId, tagNameRequest)

Rename one tag.

Renames the caller's tag. The name validation is the create's own: trimmed server-side, 1-64 Unicode characters after the trim, else 400 `VALIDATION_FAILED`.  Renaming to the name the row already carries is the idempotent no-op: 200 with the row's current state, nothing written (not even `updatedAt`). Renaming to a name ANOTHER of the caller's tags carries answers 409 `TAG_NAME_TAKEN`: the vocabulary keeps one row per name, and the honest answer for a collision is the conflict, not a silent merge into the other tag. (The create's idempotence does not extend here: the URL already names one tag, and converging on the other row would substitute the resource instead of answering.)  The response is the tag's summary after the edit, exactly as `GET /tags` lists it. The write publishes nothing (per-user state, the create's reasoning). The tag must be the caller's own: an unknown id and another account's tag answer the same 404 `TAG_NOT_FOUND`, byte for byte, so tag ids cannot be probed for existence. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getTagsApi();
final String tagId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the tag.
final TagNameRequest tagNameRequest = {"name":"Roadtrip"}; // TagNameRequest | 

try {
    final response = api.renameTag(tagId, tagNameRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling TagsApi->renameTag: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tagId** | **String**| Identifier of the tag. | 
 **tagNameRequest** | [**TagNameRequest**](TagNameRequest.md)|  | 

### Return type

[**TagSummary**](TagSummary.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

