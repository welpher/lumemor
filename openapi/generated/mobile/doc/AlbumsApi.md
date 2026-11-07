# lumemor_api.api.AlbumsApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**addAlbumItems**](AlbumsApi.md#addalbumitems) | **POST** /albums/{albumId}/items | Add assets to one virtual album.
[**createAlbum**](AlbumsApi.md#createalbum) | **POST** /albums | Create one virtual album.
[**deleteAlbum**](AlbumsApi.md#deletealbum) | **DELETE** /albums/{albumId} | Delete one virtual album.
[**getAlbum**](AlbumsApi.md#getalbum) | **GET** /albums/{albumId} | One album&#39;s detail and one page of its items.
[**grantAlbumShare**](AlbumsApi.md#grantalbumshare) | **POST** /albums/{albumId}/shares | Grant one account the view ACL of an album.
[**listAlbums**](AlbumsApi.md#listalbums) | **GET** /albums | The caller&#39;s album wall.
[**removeAlbumItems**](AlbumsApi.md#removealbumitems) | **DELETE** /albums/{albumId}/items | Remove assets from one virtual album.
[**resetAlbumTimelinePreference**](AlbumsApi.md#resetalbumtimelinepreference) | **DELETE** /albums/{albumId}/timeline-preference | Reset the caller&#39;s timeline preference of one album.
[**revokeAlbumShare**](AlbumsApi.md#revokealbumshare) | **DELETE** /albums/{albumId}/shares/{userId} | Revoke one account&#39;s view ACL of an album.
[**setAlbumTimelinePreference**](AlbumsApi.md#setalbumtimelinepreference) | **PUT** /albums/{albumId}/timeline-preference | Set the caller&#39;s timeline preference of one album.
[**updateAlbum**](AlbumsApi.md#updatealbum) | **PATCH** /albums/{albumId} | Patch one album&#39;s properties.


# **addAlbumItems**
> AddAlbumItemsResponse addAlbumItems(albumId, albumItemsRequest)

Add assets to one virtual album.

Adds assets to the caller's virtual album with the batch surface's set semantics: one request, one list of at most 200 asset ids, one honest report `{added, skipped}`.  VIRTUAL albums only: directory albums manage their own membership (the rescan flow, a later milestone) and sync albums are no album resources; both answer the single-shape 404 like an unknown id.  Skip semantics (the batch surface's missing-ids discipline): every id must belong to a live asset the CALLER owns -- the owner-based live predicate (not trashed, not hidden, not a Live Photo component row). Curating another account's shared photos into a virtual album is the re-share feature and arrives with its own milestone: the union deliberately does not carry curation yet. A foreign account's asset, a trashed, hidden or component row, and an unknown id are silently skipped, never errors. An id that already sits in the album is skipped too: membership is a set, the add is idempotent. An all-skipped request still answers 200 with `added` 0. Repeated ids collapse to one application; the counts cover the distinct ids. The inserted rows record the caller as `added_by`.  The locked gate applies: a locked album refuses the write with 409 `ALBUM_LOCKED`. Authorization is the writes' two-tier answer (invisible 404, shared-to-caller 403, the owner or an admin override writes).  The write publishes one `album.items-changed` event exactly when at least one row was added -- the idempotent no-op stays silent. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumItemsRequest albumItemsRequest = {"assetIds":["0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6072","0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6073","0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6074"]}; // AlbumItemsRequest | 

try {
    final response = api.addAlbumItems(albumId, albumItemsRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->addAlbumItems: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumItemsRequest** | [**AlbumItemsRequest**](AlbumItemsRequest.md)|  | 

### Return type

[**AddAlbumItemsResponse**](AddAlbumItemsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **createAlbum**
> AlbumSummary createAlbum(albumCreateRequest)

Create one virtual album.

Creates one virtual album owned by the caller -- the album the wall lists, the detail shows and the curation endpoints edit. The virtual album is the ONLY creatable type through this face, so the request carries no type field: directory albums arrive with their own milestone (they mirror a filesystem path the server scans), and every account's sync album is born with the account.  The title is trimmed and must span 1-64 Unicode characters (the count is over runes, so multi-byte titles stay within the bound at the same length); the description is optional bounded free text (10,000 characters, the asset description's own bound) and defaults to the empty string.  The response is the created album's wall card -- the summary exactly as `GET /albums` lists it: `callerRole` `owner`, `isShared` false, an empty `coverAssetIds` array (never null) and `itemCount` 0, nothing being a member yet. The write publishes one `album.created` event. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final AlbumCreateRequest albumCreateRequest = {"title":"Road trip 2026","description":"The coastal week in the van."}; // AlbumCreateRequest | 

try {
    final response = api.createAlbum(albumCreateRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->createAlbum: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumCreateRequest** | [**AlbumCreateRequest**](AlbumCreateRequest.md)|  | 

### Return type

[**AlbumSummary**](AlbumSummary.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteAlbum**
> deleteAlbum(albumId)

Delete one virtual album.

Deletes the caller's virtual album: the row goes, and every `album_items` membership row and every `album_shares` row follows through the schema's cascades. The answer is 204 with no body, and the write publishes one `album.deleted` event.  VIRTUAL albums only: a directory album answers the single-shape 404 (directory removal is the directory album's own flow, not a generic delete -- it arrives with a later milestone), and a sync album is no album resource at all. The same 404 answers an unknown id.  The locked gate applies to the delete like every other write: the permission matrix pins deletion to the UNLOCKED state, so a locked album answers 409 `ALBUM_LOCKED` (the owner unlocks first -- one step, the freeze is a state to clear, not a wall to climb).  Authorization is the writes' two-tier answer: an album the caller cannot see answers the single-shape 404; an album shared to the caller answers 403 `AUTH_FORBIDDEN` (the share is view-only); the owner -- or an administrator overriding the owner check -- deletes. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.

try {
    api.deleteAlbum(albumId);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->deleteAlbum: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getAlbum**
> AlbumDetail getAlbum(albumId, cursor, limit)

One album's detail and one page of its items.

The detail view of one album the caller can see: the caller owns it, or it is shared to the caller through an `album_shares` row. `callerRole` says which -- `owner` for the owner's write affordances, `viewer` for the read-only share -- with the one exception below.  The caller's OWN sync album answers 200: the default album's read-only detail. `callerRole` is `owner`, the live members page through the same keyset as any album's (the grid and the viewer work like any album's), but the album is system-managed: NO write affordance applies to it -- the property patch, the membership curation, the sharing face, the timeline preference and the delete all keep refusing a sync album with their unchanged 404/400 answers, whatever this read admits. `inTimeline` is absent (the core stream is never excludable) and the owner's `shares` list rides the response always empty (no share row can name a sync album). The client renders the localized display name (en \"My photos\", zh-Hans \"我的照片\") keyed by `type: 'sync'` -- the stored title is the internal system label and never renders.  A sync album of ANOTHER account answers the same 404 as an unknown id, whatever share rows exist: invisible by design.  404 is single-shape by design: an unknown id, another account's unshared album, and another account's sync album all answer the same `ALBUM_NOT_FOUND` byte for byte -- the anti-enumeration discipline of `ASSET_NOT_FOUND` applied to the album face. An album whose share to the caller was revoked before the request simply sits in that same invisible region.  `itemCount` counts the album's live members; `items` is one keyset page of them, ordered `takenAt` descending with the asset id as the tiebreak -- the timeline's own ordering and cursor pair, so the album grid pages with the timeline's walking machinery. Every item carries the membership's `addedAt` and the minimal asset context the grid renders; `thumbnail` is the asset thumbnail endpoint's `tile`-size URL. Live members only: trashed, hidden and Live Photo component rows never appear (A5 -- a hidden asset is invisible to the whole family).  `coverAssetId` is the raw pinned column: the single asset the owner chose, or null when unset -- the wall derives its collage fallback from the members, the detail carries the pin itself.  `shares` rides only the owner's view: the album's grantee list, one entry per `album_shares` row in the grant order (`grantedAt` ascending), each carrying the grantee's id, email and grant time -- the share management dialog's list. The owner's response always carries the field, an empty array when the album holds no grants; viewer responses never carry it at all (a sharee sees the album, never the ACL).  `inTimeline` is the caller's timeline preference of this album (the exclusion toggle's source of truth): present on the excludable types -- the caller's own `directory` albums and the albums shared to them -- and absent on the caller's own `virtual` albums. The preference scopes the aggregate: it never filters this album's own items, so browsing a switched-off album still shows everything.  `limit` bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 `VALIDATION_FAILED`. Pages continue through the opaque `cursor`, the timeline's exact keyset semantics: `nextCursor` is present exactly when another page follows, and the last page carries none. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final String cursor = GN4WpRPw0AABkvShW2x9jp8KKzxNXm9w; // String | The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
final int limit = 60; // int | Page size: the number of album items one response carries at most. Defaults to 60; values below 1 or over 200 answer 400. 

try {
    final response = api.getAlbum(albumId, cursor, limit);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->getAlbum: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **cursor** | **String**| The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients).  | [optional] 
 **limit** | **int**| Page size: the number of album items one response carries at most. Defaults to 60; values below 1 or over 200 answer 400.  | [optional] [default to 60]

### Return type

[**AlbumDetail**](AlbumDetail.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **grantAlbumShare**
> AlbumShareResponse grantAlbumShare(albumId, albumShareGrantRequest)

Grant one account the view ACL of an album.

Grants an album's view ACL to another account: one `album_shares` row, one more viewer. The grantee's read access is immediate -- their wall lists the album in the shared group, the detail answers with `callerRole` `viewer` -- and the share is view-only forever: every write face of the album stays the owner's.  Authorization is the writes' two-tier answer: an album the caller cannot see at all -- an unknown id, another account's unshared album, every sync album -- answers the single-shape 404 `ALBUM_NOT_FOUND`, byte-identical to the read face, so album ids cannot be probed; an album the caller CAN see but does not own (an album shared to the caller) answers the honest 403 `AUTH_FORBIDDEN` -- a sharee cannot share onward. The owner grants, and an administrator may override the owner check.  VIRTUAL and DIRECTORY albums are shareable; sync albums are never -- a sync album id folds into the same single-shape 404 (the timeline is the sync album's face). `locked` does NOT gate the grant: the freeze is a curation gate, and a viewer only ever reads.  The grant is idempotent: re-granting to a current grantee answers 200 with the existing row -- the same `grantedAt`, the original grantor untouched -- and stays silent on the activity stream.  Target rules (both answer 400 `VALIDATION_FAILED` with a message key in `details.key`): the album's own owner is never a grantee (`album_share_to_owner` -- the owner sees the album by construction, a share row would be noise); an unknown or disabled account is not a share target (`album_share_user_invalid` -- the two cases answer the same shape, so user ids cannot be probed through this face).  The write publishes one `album.shared` event exactly when the grant landed a new row: the payload carries the album's id and the GRANTEE's user id, nothing else. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumShareGrantRequest albumShareGrantRequest = {"userId":"0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70"}; // AlbumShareGrantRequest | 

try {
    final response = api.grantAlbumShare(albumId, albumShareGrantRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->grantAlbumShare: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumShareGrantRequest** | [**AlbumShareGrantRequest**](AlbumShareGrantRequest.md)|  | 

### Return type

[**AlbumShareResponse**](AlbumShareResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listAlbums**
> AlbumWallResponse listAlbums()

The caller's album wall.

The album wall of the calling user: every album that deserves a wall card, in two groups concatenated into one array -- the web client renders its sections off the `callerRole` discriminator each item carries.  The caller's sync album is pinned FIRST, before the own group: the one card of type `sync`, the caller's own core upload stream surfaced as the default \"My photos\" album -- every upload lands in it, so a fresh account's wall is never empty. The own group follows: the albums the caller owns of type `virtual` or `directory`, the most recently updated first (the album id descending as the tiebreak). The shared group follows last: every album shared to the caller through an `album_shares` row, the caller's most recently granted share first (the album id descending as the tiebreak). Shared items carry `owner` -- the honest attribution of another account's album: the owner's id and email.  A sync album of ANOTHER account never appears, in either group: the shared arm carries `virtual` and `directory` albums only, whatever share rows exist. The pinned card's `title` is the system's internal fixed label ('Sync') -- the display name is the CLIENT's: `type: 'sync'` keys the localized default-album name (en \"My photos\", zh-Hans \"我的照片\"), and the stored title never renders.  `coverAssetIds` is the wall card's collage source: the pinned `coverAssetId` as a single-element list when the owner set one, otherwise the album's most recent live member assets -- up to four, `takenAt` descending with the asset id as the tiebreak, fewer when the album holds fewer members, empty for an album with none. `itemCount` counts live members under the same predicate: trashed, hidden and Live Photo component rows never count (A5 -- a hidden asset is invisible to the whole family).  `isShared` reports whether the album carries at least one `album_shares` row -- the owner's \"someone else sees this\" badge. It is true by construction on every shared-group item (the caller's share row is the reason the item exists).  `inTimeline` rides the excludable cards -- the own `directory` items and every shared item: the caller's timeline preference of that album, true when no preference row exists (the default of feeding the timeline). The own `virtual` items omit the field: they never feed the caller's own timeline. The pinned `sync` card omits it too: the core stream is never excludable (the preference face answers 400 for it). The field is the toggle's source of truth for the wall's \"excluded\" badge, kept current by the `PUT/DELETE /albums/{albumId}/timeline-preference` endpoints. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();

try {
    final response = api.listAlbums();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->listAlbums: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**AlbumWallResponse**](AlbumWallResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **removeAlbumItems**
> RemoveAlbumItemsResponse removeAlbumItems(albumId, albumItemsRequest)

Remove assets from one virtual album.

Removes membership rows from the caller's virtual album: `{removed, skipped}`, the same batch semantics as the add. An id that is no member of the album -- never was, or already removed -- is skipped; an all-skipped request still answers 200 with `removed` 0. Repeated ids collapse; the counts cover the distinct ids.  The removal carries no live predicate: a trashed member's row still leaves, the honest cleanup of the set (the live views filter trashed members out of the renders; removing the row tidies the set itself).  VIRTUAL albums only (directory and sync albums answer the single-shape 404); the locked gate applies (409 `ALBUM_LOCKED`); authorization is the writes' two-tier answer (invisible 404, shared-to-caller 403, the owner or an admin override writes).  The write publishes one `album.items-changed` event exactly when at least one row was removed -- the idempotent no-op stays silent. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumItemsRequest albumItemsRequest = {"assetIds":["0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6072","0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6073"]}; // AlbumItemsRequest | 

try {
    final response = api.removeAlbumItems(albumId, albumItemsRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->removeAlbumItems: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumItemsRequest** | [**AlbumItemsRequest**](AlbumItemsRequest.md)|  | 

### Return type

[**RemoveAlbumItemsResponse**](RemoveAlbumItemsResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **resetAlbumTimelinePreference**
> resetAlbumTimelinePreference(albumId)

Reset the caller's timeline preference of one album.

Removes the calling user's preference row of one album: the album returns to the default state of feeding the caller's timeline (a missing row means kept in). The eligibility scope is the PUT's own -- the caller's own directory albums and the albums shared to the caller; the same 400 `album_pref_not_excludable` answers the caller's own sync and own virtual albums, and the same single-shape 404 answers an album the caller cannot see.  The delete is idempotent: an album with no preference row (never set, or already reset) still answers 204 -- the row is gone either way, which is the state the caller asked for. Like the PUT it publishes no event: the preference is the caller's own state, and no other session has anything to refetch. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.

try {
    api.resetAlbumTimelinePreference(albumId);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->resetAlbumTimelinePreference: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **revokeAlbumShare**
> revokeAlbumShare(albumId, userId)

Revoke one account's view ACL of an album.

Removes one `album_shares` row: the grantee's read access ends immediately -- the album leaves their wall, the detail answers the single-shape 404.  The owner (or an administrator) revokes any grantee. A grantee may revoke their OWN row -- leaving a shared album is the sharee's own right, the family semantics -- through this same endpoint: the caller's own user id in the path is the departure, whoever granted the row. A sharee cannot revoke ANOTHER account's grant: 403 `AUTH_FORBIDDEN`, the honest refusal of a caller who can see the album but holds no power over its ACL.  Everything else mirrors the grant: the single-shape 404 for the invisible ids (an unknown album, another account's unshared album, every sync album), and `locked` never gates -- revocation is not curation.  The 204 is idempotent: revoking a share that is already gone -- never granted, already revoked, or an id that names no account -- is the state the caller asked for, so it answers 204 again and stays silent on the activity stream.  The write publishes one `album.unshared` event exactly when a row left: the payload carries the album's id and the revoked grantee's user id, nothing else. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final String userId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the account whose share row leaves: any grantee for the owner or an administrator, the caller's own id for a sharee's departure. 

try {
    api.revokeAlbumShare(albumId, userId);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->revokeAlbumShare: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **userId** | **String**| Identifier of the account whose share row leaves: any grantee for the owner or an administrator, the caller's own id for a sharee's departure.  | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **setAlbumTimelinePreference**
> AlbumTimelinePreferenceResponse setAlbumTimelinePreference(albumId, albumTimelinePreferenceRequest)

Set the caller's timeline preference of one album.

Upserts the calling user's per-user timeline preference of one album: the `user_album_prefs` row keyed (userId, albumId) whose `inTimeline` switch says whether the album feeds the caller's aggregated timeline. The preference is the CALLER's own state over an album they can see -- self-affecting only, which is why these writes publish no event: the caller's own response is the only view that changed, and no other session has anything to refetch. The scope is the albums whose membership a caller could want out of their timeline feed: the caller's own `directory` albums, and every album shared to the caller (whatever the owner's type). Two kinds of album answer 400 instead, with the `album_pref_not_excludable` message key in the details:  - the caller's own `sync` album: it IS the core stream, and   the core stream is never excludable (the aggregation   formula exempts the sync arm from the preference row --   a stray row there excludes nothing); - the caller's own `virtual` albums: they never feed the   caller's own timeline at all (in-stream curation would be   pure duplication), so there is nothing to exclude.  An album the caller cannot see -- unknown, another account's unshared album, a revoked share -- answers the same byte-identical 404 `ALBUM_NOT_FOUND` as every album face, so album ids cannot be probed for existence through this endpoint. There is no administrator override: the preference is always the caller's own row over their own view, never a write onto another account's state.  `inTimeline` is required and must be a JSON boolean: a body that is not valid JSON, or one without the field, answers 400 `VALIDATION_FAILED`. The write is an upsert: repeating it with the same value converges to the same row, and repeating it with the other value flips the switch -- the response always carries the stored state. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumTimelinePreferenceRequest albumTimelinePreferenceRequest = {"inTimeline":false}; // AlbumTimelinePreferenceRequest | 

try {
    final response = api.setAlbumTimelinePreference(albumId, albumTimelinePreferenceRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->setAlbumTimelinePreference: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumTimelinePreferenceRequest** | [**AlbumTimelinePreferenceRequest**](AlbumTimelinePreferenceRequest.md)|  | 

### Return type

[**AlbumTimelinePreferenceResponse**](AlbumTimelinePreferenceResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateAlbum**
> AlbumSummary updateAlbum(albumId, albumPatch)

Patch one album's properties.

The owner's partial edit of the album's properties: `title`, `description`, `coverAssetId` and `locked`, any subset, at least one field (a body that carries none -- JSON null counts as absent, the patch discipline of the asset surface -- is a client bug and answers 400). The response is the album's wall card after the edit.  Authorization is the two-tier answer every album write pins: an album the caller cannot see at all -- an unknown id, another account's unshared album, every sync album -- answers the single-shape 404 `ALBUM_NOT_FOUND`, byte-identical to the read face, so album ids cannot be probed; an album the caller CAN see but does not own (an album shared to the caller) is an honest authorization failure -- 403 `AUTH_FORBIDDEN`, the share is view-only. The owner writes, and an administrator may override the owner check (the admin role bypasses it, the permanent delete's own discipline).  The locked gate: a locked album refuses the property and curation fields (`title`, `description`, `coverAssetId`) with 409 `ALBUM_LOCKED` -- unlock first. The `locked` field itself is never gated: a patch of only `locked: false` IS the unlock.  Cover validation (the spec's application-layer pin): a present `coverAssetId` must be a LIVE member of THIS album -- the membership row exists and the asset passes the live predicate (not trashed, not hidden, not a Live Photo component row). Any other id -- an asset that is no member, a trashed, hidden or component row, another account's asset -- answers 400 `VALIDATION_FAILED` carrying the message key `album_cover_not_member`. A `coverAssetId` that is absent (or JSON null, which decodes as absent) leaves the pin unchanged.  The write publishes one `album.updated` event on every successful patch, whatever fields changed -- the convergent refetch makes the unconditional fire harmless. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAlbumsApi();
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the album.
final AlbumPatch albumPatch = {"title":"Road trip 2026 -- rerun"}; // AlbumPatch | 

try {
    final response = api.updateAlbum(albumId, albumPatch);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AlbumsApi->updateAlbum: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **albumId** | **String**| Identifier of the album. | 
 **albumPatch** | [**AlbumPatch**](AlbumPatch.md)|  | 

### Return type

[**AlbumSummary**](AlbumSummary.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

