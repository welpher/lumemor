# lumemor_api.api.SearchApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**searchAssets**](SearchApi.md#searchassets) | **GET** /search | The unified search, one text box, two merged paths, filter facets.


# **searchAssets**
> SearchResponse searchAssets(query, personIds, placeIds, from, to, cameras, kinds, albumIds, tagIds, isMotion, locale, cursor, limit)

The unified search, one text box, two merged paths, filter facets.

The unified search over the caller-visible asset domain (the album union of the caller's own sync and directory albums plus every album shared to them, and the timeline's live-row filters: not trashed, not archived, no Live Photo component row). One optional text box drives TWO backend paths that merge deterministically; the optional filter parameters narrow both paths and the pure-filter mode (no text) is the filter browse the search page opens with.  The TEXT path is unaccented case-insensitive SUBSTRING matching over the asset's own text sources -- the original filename, the description, and the extracted OCR text -- plus the caller's relational vocabularies joined at query time: the caller's own tags, the titles of albums the caller can see (own albums and albums shared to them; the sync album is the timeline's face, not a filter), and the caller's own non-hidden person labels. The query is a LITERAL substring, never a pattern: the engine escapes its own wildcard characters, so `50%` matches the literal text `50%` and never globs. Word semantics, stemming and segmentation are out of scope by design (the v1 boundary: the match is containment, which is language-agnostic by construction; a tokenizing engine is the v1.x upgrade path). The path's weak rank signal is the trigram similarity of the best matched source.  The SEMANTIC path embeds the query through the active CLIP text tower (the model registry's clip-text slot) and ranks the caller-visible assets by cosine similarity over the image embeddings the active clip-embed model produced -- the same joint space both towers share. The semantic path needs the model machinery: when no text tower is active (or the tower's dimension disagrees with the embeddings column) the semantic path honestly answers EMPTY and the text path still serves -- the degradation the model admin page resolves by downloading and activating the text tower.  The MERGE: both paths answer at most their top 500 ranked hits; an asset's fused score is the sum over the paths it appears in of (500 minus its zero-based position in that path), so an asset hit by both paths outranks the single-path hits around it; the page orders by the fused score descending with the asset id ascending as the total-order tiebreak, and one asset hit through many sources or both paths appears exactly once. The rule is deterministic: the same query over unchanged data answers the same page, and cursors page through that stable order.  A query of fewer than two characters (after trimming), or only whitespace, takes NO text path and NO semantic path: the request is the pure filter browse. An empty `from`/`to` pair with no other filter is the whole visible timeline.  The FACETS count the current filtered set minus their own dimension: `people` lists the caller's named, non-hidden people with the number of assets of the current set (query and every filter applied EXCEPT personIds) that carry a face of that cluster; `places`, `years`, `cameras`, `kinds`, `albums` and `tags` follow the same rule for their own dimension. Facet ids the caller cannot resolve (an album they cannot see, another account's tag) DROP OUT of the filter parameters rather than emptying the result -- filters narrow, they never probe.  `personIds`, `albumIds` and `tagIds` are comma-separated id lists; `placeIds` is a comma-separated integer list; `cameras` is a comma-separated list of the camera facet ids (`make|model`). Every id list is capped at 200 entries; a longer list answers 400 `VALIDATION_FAILED`. `from` is inclusive, `to` is EXCLUSIVE; an inverted range answers 400. `locale` picks the places facet's display names exactly the way `GET /places` does. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getSearchApi();
final String query = 海边; // String | The free-text query: a literal substring matched (unaccented, case-insensitively) against the filename, the description, the OCR text and the caller's tag, album-title and person-name vocabularies, and embedded through the CLIP text tower for the semantic path. A value of fewer than two characters (after trimming) takes no text path: the request is the pure filter browse. The length is capped at 256 characters. 
final BuiltList<String> personIds = ; // BuiltList<String> | Restrict the results to assets carrying a face of one of these face clusters (the caller's people surface's cluster ids). 
final BuiltList<int> placeIds = ; // BuiltList<int> | Restrict the results to assets resolved to one of these places (the places facet's GeoNames ids). 
final DateTime from = 2024-01-01T00:00:00Z; // DateTime | The inclusive lower bound of the taken-at range. 
final DateTime to = 2025-01-01T00:00:00Z; // DateTime | The EXCLUSIVE upper bound of the taken-at range. A `to` at or before `from` answers 400 `VALIDATION_FAILED`. 
final BuiltList<String> cameras = ; // BuiltList<String> | Restrict the results to assets whose EXIF camera matches one of these camera facet ids (`make|model`). 
final BuiltList<MediaKind> kinds = ; // BuiltList<MediaKind> | Restrict the results to these media families. A value outside `image`/`video` answers 400. 
final BuiltList<String> albumIds = ; // BuiltList<String> | Restrict the results to members of these albums. Albums the caller cannot see drop out of the filter: they contribute nothing, they never empty the result. 
final BuiltList<String> tagIds = ; // BuiltList<String> | Restrict the results to assets carrying one of the CALLER's tags. Another account's tag ids drop out of the filter (tags are per-user state). 
final bool isMotion = true; // bool | Restrict the results to Live Photos (`true`) or to everything else (`false`): an asset counts as a Live Photo exactly when it carries a motion. THE INTERFACE LEAVES THIS SLOT UNEXPOSED for now -- the search surface's filter bar does not render it and the web client never sends it; the parameter exists so the backend can answer the question when a surface asks, without a second contract revision. It is not a facet: the facet counts ignore it. 
final String locale = zh-Hans; // String | The display locale of the places facet's names, exactly the `GET /places` locale join. Absent means `en`. A value outside the set answers 400 `VALIDATION_FAILED`. 
final String cursor = GN4WpRPw0AABkvShW2x9jp8KKzxNXm9w; // String | The `nextCursor` of the previous page; absent on the first page. The cursor carries the page's ordering mode (the fused rank of a text query, or the timeline order of the filter browse): a cursor of the other mode answers 400 `VALIDATION_FAILED`, and so does any value this endpoint did not produce. 
final int limit = 100; // int | Page size: the number of items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400. 

try {
    final response = api.searchAssets(query, personIds, placeIds, from, to, cameras, kinds, albumIds, tagIds, isMotion, locale, cursor, limit);
    print(response);
} on DioException catch (e) {
    print('Exception when calling SearchApi->searchAssets: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **query** | **String**| The free-text query: a literal substring matched (unaccented, case-insensitively) against the filename, the description, the OCR text and the caller's tag, album-title and person-name vocabularies, and embedded through the CLIP text tower for the semantic path. A value of fewer than two characters (after trimming) takes no text path: the request is the pure filter browse. The length is capped at 256 characters.  | [optional] 
 **personIds** | [**BuiltList&lt;String&gt;**](String.md)| Restrict the results to assets carrying a face of one of these face clusters (the caller's people surface's cluster ids).  | [optional] 
 **placeIds** | [**BuiltList&lt;int&gt;**](int.md)| Restrict the results to assets resolved to one of these places (the places facet's GeoNames ids).  | [optional] 
 **from** | **DateTime**| The inclusive lower bound of the taken-at range.  | [optional] 
 **to** | **DateTime**| The EXCLUSIVE upper bound of the taken-at range. A `to` at or before `from` answers 400 `VALIDATION_FAILED`.  | [optional] 
 **cameras** | [**BuiltList&lt;String&gt;**](String.md)| Restrict the results to assets whose EXIF camera matches one of these camera facet ids (`make|model`).  | [optional] 
 **kinds** | [**BuiltList&lt;MediaKind&gt;**](MediaKind.md)| Restrict the results to these media families. A value outside `image`/`video` answers 400.  | [optional] 
 **albumIds** | [**BuiltList&lt;String&gt;**](String.md)| Restrict the results to members of these albums. Albums the caller cannot see drop out of the filter: they contribute nothing, they never empty the result.  | [optional] 
 **tagIds** | [**BuiltList&lt;String&gt;**](String.md)| Restrict the results to assets carrying one of the CALLER's tags. Another account's tag ids drop out of the filter (tags are per-user state).  | [optional] 
 **isMotion** | **bool**| Restrict the results to Live Photos (`true`) or to everything else (`false`): an asset counts as a Live Photo exactly when it carries a motion. THE INTERFACE LEAVES THIS SLOT UNEXPOSED for now -- the search surface's filter bar does not render it and the web client never sends it; the parameter exists so the backend can answer the question when a surface asks, without a second contract revision. It is not a facet: the facet counts ignore it.  | [optional] 
 **locale** | **String**| The display locale of the places facet's names, exactly the `GET /places` locale join. Absent means `en`. A value outside the set answers 400 `VALIDATION_FAILED`.  | [optional] [default to 'en']
 **cursor** | **String**| The `nextCursor` of the previous page; absent on the first page. The cursor carries the page's ordering mode (the fused rank of a text query, or the timeline order of the filter browse): a cursor of the other mode answers 400 `VALIDATION_FAILED`, and so does any value this endpoint did not produce.  | [optional] 
 **limit** | **int**| Page size: the number of items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400.  | [optional] [default to 100]

### Return type

[**SearchResponse**](SearchResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

