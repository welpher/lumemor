# lumemor_api.api.AlbumsApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getAlbum**](AlbumsApi.md#getalbum) | **GET** /albums/{albumId} | One album&#39;s detail and one page of its items.
[**listAlbums**](AlbumsApi.md#listalbums) | **GET** /albums | The caller&#39;s album wall.


# **getAlbum**
> AlbumDetail getAlbum(albumId, cursor, limit)

One album's detail and one page of its items.

The detail view of one album the caller can see: the caller owns it, or it is shared to the caller through an `album_shares` row. `callerRole` says which -- `owner` for the owner's write affordances, `viewer` for the read-only share.  Sync albums are not album resources: the id of a sync album -- the caller's own or anyone else's -- answers the same 404 as an unknown id. The sync album is the timeline (`GET /assets/timeline`); its stream surfaces there, never through this face.  404 is single-shape by design: an unknown id, another account's unshared album, and a sync album all answer the same `ALBUM_NOT_FOUND` byte for byte -- the anti-enumeration discipline of `ASSET_NOT_FOUND` applied to the album face. An album whose share to the caller was revoked before the request simply sits in that same invisible region.  `itemCount` counts the album's live members; `items` is one keyset page of them, ordered `takenAt` descending with the asset id as the tiebreak -- the timeline's own ordering and cursor pair, so the album grid pages with the timeline's walking machinery. Every item carries the membership's `addedAt` and the minimal asset context the grid renders; `thumbnail` is the asset thumbnail endpoint's `tile`-size URL. Live members only: trashed, hidden and Live Photo component rows never appear (A5 -- a hidden asset is invisible to the whole family).  `coverAssetId` is the raw pinned column: the single asset the owner chose, or null when unset -- the wall derives its collage fallback from the members, the detail carries the pin itself.  `limit` bounds the page size: absent means the default of 60; a value below 1 or over 200 answers 400 `VALIDATION_FAILED`. Pages continue through the opaque `cursor`, the timeline's exact keyset semantics: `nextCursor` is present exactly when another page follows, and the last page carries none. 

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

# **listAlbums**
> AlbumWallResponse listAlbums()

The caller's album wall.

The album wall of the calling user: every album that deserves a wall card, in two groups concatenated into one array -- the web client renders its sections off the `callerRole` discriminator each item carries.  The own group comes first: the albums the caller owns of type `virtual` or `directory`, the most recently updated first (the album id descending as the tiebreak). The shared group follows: every album shared to the caller through an `album_shares` row, the caller's most recently granted share first (the album id descending as the tiebreak). Shared items carry `owner` -- the honest attribution of another account's album: the owner's id and email.  Sync albums NEVER appear on the wall, in either group. The sync album is the account's core backup stream and lives as the timeline (`GET /assets/timeline`), not as an album resource; a sync album another account shared is likewise no wall card. The caller's own sync album is invisible here and answers 404 through `GET /albums/{albumId}`.  `coverAssetIds` is the wall card's collage source: the pinned `coverAssetId` as a single-element list when the owner set one, otherwise the album's most recent live member assets -- up to four, `takenAt` descending with the asset id as the tiebreak, fewer when the album holds fewer members, empty for an album with none. `itemCount` counts live members under the same predicate: trashed, hidden and Live Photo component rows never count (A5 -- a hidden asset is invisible to the whole family).  `isShared` reports whether the album carries at least one `album_shares` row -- the owner's \"someone else sees this\" badge. It is true by construction on every shared-group item (the caller's share row is the reason the item exists). 

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

