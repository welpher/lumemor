# lumemor_api.api.AssetsApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**batchAssets**](AssetsApi.md#batchassets) | **POST** /assets/batch | Apply one set action to a batch of assets.
[**deleteAsset**](AssetsApi.md#deleteasset) | **DELETE** /assets/{id} | Permanently delete one asset, files and all.
[**getAsset**](AssetsApi.md#getasset) | **GET** /assets/{id} | Fetch the detail of one owned asset.
[**getAssetByDeviceId**](AssetsApi.md#getassetbydeviceid) | **GET** /assets/by-device-id/{deviceAssetId} | Look up an uploaded asset by its device identifier.
[**getAssetOriginal**](AssetsApi.md#getassetoriginal) | **GET** /assets/{id}/original | Download the original file of one owned asset.
[**getAssetThumbnail**](AssetsApi.md#getassetthumbnail) | **GET** /assets/{id}/thumbnail | Serve a derived thumbnail, or a video&#39;s poster frame.
[**getFavorites**](AssetsApi.md#getfavorites) | **GET** /assets/favorites | The caller&#39;s favorites, one keyset page at a time.
[**getHidden**](AssetsApi.md#gethidden) | **GET** /assets/hidden | The caller&#39;s hidden assets, one keyset page at a time.
[**getTimeline**](AssetsApi.md#gettimeline) | **GET** /assets/timeline | The aggregated timeline, one keyset page at a time.
[**getTrash**](AssetsApi.md#gettrash) | **GET** /trash | List the caller&#39;s trashed assets with their retention countdown.
[**patchAsset**](AssetsApi.md#patchasset) | **PATCH** /assets/{id} | Edit the description or hidden switch of one asset.
[**playAsset**](AssetsApi.md#playasset) | **GET** /assets/{id}/play | Stream the transcoded playable copy of one owned video.
[**restoreAsset**](AssetsApi.md#restoreasset) | **POST** /trash/{id}/restore | Restore one trashed asset.
[**uploadAsset**](AssetsApi.md#uploadasset) | **POST** /assets | Upload one photo, optionally with its Live Photo video.


# **batchAssets**
> BatchResponse batchAssets(batchRequest)

Apply one set action to a batch of assets.

The batch surface of the curation actions: one request applies one action -- favorite, unfavorite, hide, unhide, trash or restore -- to a list of up to 1000 asset ids, in one transaction. The endpoint follows set semantics: it is a convergence operation, not a state-change counter. Every id that resolves to an asset the caller owns is applied (an id already in the target state is the idempotent no-op and still counts as affected), and every id that does not resolve -- unknown, or another account's asset, the two indistinguishable -- is ignored and reported in the response's `missing` list. The two lists partition the request's distinct ids: no individual id ever fails the request, and a repeat of a successful batch converges to the same answer.  This is the deliberate trade of the list surface: the byte-identical 404 anti-enumeration of the single-asset endpoints does not apply here. The caller handed the ids over themselves, so reporting the unresolvable ones back in `missing` hides nothing -- and hands honest clients their retry list.  `ids` must carry between 1 and 1000 entries: an empty list, or one past the cap, answers 400 `VALIDATION_FAILED`, as does an `action` outside the enum or a body that is not valid JSON (an id that is not a uuid fails the body decode). Repeated ids collapse to one application.  The whole batch commits as one transaction: the action's writes land together or not at all, so a partial batch never surfaces. Failures on resolvable ids are not expected -- the per-item guards turn foreign and already-in-state ids into no-ops, not errors -- so the all-or-nothing bound is the infrastructure safety, not a client-visible case.  The actions resolve to their single-row primitives: favorite / unfavorite write or remove the caller's per-user favorites rows; hide / unhide flip the owner-level visibility switch of the row the id names (a hidden asset leaves every live view -- the timeline, the favorites -- until unhidden); trash / restore run the trash lifecycle of the trash endpoints, Live Photo pair included (trashing or restoring the photo carries its component video row along, whichever row the id names).  Events publish AFTER the commit, one per asset whose state actually changed -- the idempotent no-op stays silent, the same discipline as the single-asset restore: `asset.trashed` and `asset.restored` for trash and restore, `asset.updated` for hide and unhide (visibility is owner-level state, and every viewer of the shared timeline sees it change). favorite / unfavorite publish nothing: favorites are per-user state with no cross-user visibility effect, so no other session has anything to refresh. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final BatchRequest batchRequest = {"action":"trash","ids":["0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70","0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f71"]}; // BatchRequest | 

try {
    final response = api.batchAssets(batchRequest);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->batchAssets: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **batchRequest** | [**BatchRequest**](BatchRequest.md)|  | 

### Return type

[**BatchResponse**](BatchResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteAsset**
> deleteAsset(id)

Permanently delete one asset, files and all.

The explicit permanent deletion -- the destructive surface beside the trash. It removes the asset row whatever its state (live or trashed: this endpoint does not route through the trash), and with it the asset's on-disk artifacts under the shared-blob rule: content addressing means another owner's row (or any other row) with the same checksum references the SAME physical file, so the original file is removed only when no other asset row references it; otherwise the row goes and the file stays. The derived serving copies (the per-asset thumbnail directory) belong to this asset id alone and are always removed. The component video row of a Live Photo is deleted with its photo (it is never an asset of its own).  The application-layer permission gate of the two-layer delete discipline: the owner of the asset, or an administrator. An id of another user's asset answers the same 404 `ASSET_NOT_FOUND` as an id that does not exist, byte for byte, so asset ids cannot be enumerated. (The file-system layer of the gate matters for directory albums, whose files move on delete; it arrives with the album domain in a later milestone -- blob originals live inside the server's own data directory.)  The ordinary client \"delete\" is the trash (soft delete through the batch endpoint); this endpoint is the explicit bypass: the client that already knows it wants the asset gone forever, and the empty-trash surface. Repeating the request after a successful deletion answers 404 -- the asset no longer exists, exactly like an unknown id.  The deletion is committed before the files are removed: a failure between the two (a full disk, a failing mount) leaves the row gone and the file orphaned, which is a storage leak, never data loss; it is logged server-side and the response still reports the deletion. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the asset.

try {
    api.deleteAsset(id);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->deleteAsset: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Identifier of the asset. | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getAsset**
> AssetDetail getAsset(id)

Fetch the detail of one owned asset.

The asset detail: the identity and storage fields of the upload response plus the refinements the pipeline produced afterwards -- dimensions, duration, blurhash, the EXIF hot fields, and readiness -- and the two owner-level editable attributes (`description`, `isHidden`), which the asset detail view renders and the `PATCH` endpoint edits.  Access is owner-only: an id that belongs to another account answers the same 404 `ASSET_NOT_FOUND` as an id that does not exist, byte for byte, so asset ids cannot be enumerated. Album sharing and ACLs widen access for shared content in a later milestone (M2c, with the album domain); until then only the owner's own assets answer.  A trashed asset answers 200 with `isTrashed: true` -- the trash view reads the detail the same way. The component video row of a Live Photo is readable by its id (the detail and the original download resolve it through `livePhotoVideoId`; the play endpoint answers 404 for it -- the component counts no transcode of its own) but never appears in the timeline. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the asset.

try {
    final response = api.getAsset(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->getAsset: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Identifier of the asset. | 

### Return type

[**AssetDetail**](AssetDetail.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getAssetByDeviceId**
> AssetResponse getAssetByDeviceId(deviceAssetId)

Look up an uploaded asset by its device identifier.

The backup idempotency probe: the app asks, before uploading a local file, whether its `deviceAssetId` already exists for this account, and skips the upload when it does.  The lookup is owner-scoped -- a `deviceAssetId` of another user's asset is invisible and answers 404. The `state` field mirrors what a re-upload of the same content would produce: `duplicate` when the asset is live, `trashed` when it sits in the trash (the client can then offer restore or save-as-new before spending bandwidth). 404 means the device has not uploaded this identifier (or only under another account). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String deviceAssetId = deviceAssetId_example; // String | The device-side identity the asset was uploaded with.

try {
    final response = api.getAssetByDeviceId(deviceAssetId);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->getAssetByDeviceId: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **deviceAssetId** | **String**| The device-side identity the asset was uploaded with. | 

### Return type

[**AssetResponse**](AssetResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getAssetOriginal**
> Uint8List getAssetOriginal(id)

Download the original file of one owned asset.

Streams the stored original bytes as a download. It is available immediately after the upload: the original is the uploaded content-addressed file itself, not a pipeline product.  The download is always an attachment: `Content-Disposition: attachment; filename*=UTF-8''...` with the original file name escaped per RFC 5987 -- every byte outside the attribute character set is percent-encoded, which rules out header injection and in-browser rendering alike (SVG is not in the upload whitelist, so the download-only rule covers every stored format). `Content-Type` is the asset's stored MIME type. The response carries the same `ETag` and `Cache-Control` validators as the thumbnail (the original bytes are content-addressed, hence immutable).  Byte ranges follow the Go standard library's `ServeContent` semantics: a satisfiable single range answers 206 with `Content-Range`; multiple ranges answer a `multipart/byteranges` body; an unsatisfiable range answers 416 with the total size in `Content-Range`; a malformed Range header is ignored and the full body answers 200.  The same owner-only download gate as every asset endpoint: foreign and unknown ids answer byte-identical 404 `ASSET_NOT_FOUND`. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the asset.

try {
    final response = api.getAssetOriginal(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->getAssetOriginal: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Identifier of the asset. | 

### Return type

[**Uint8List**](Uint8List.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/octet-stream, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getAssetThumbnail**
> Uint8List getAssetThumbnail(id, size)

Serve a derived thumbnail, or a video's poster frame.

The static serving copy of the asset:  - image assets: the AVIF thumbnail of the requested size --   `tile` (fits the 250px timeline box), `view` (long edge   capped at 1080px), `original` (the full-resolution   re-encode) -- served as `image/avif`; - video assets: the poster frame, whatever `size` says -- the   poster (the first frame, scaled into the tile box) is the   video's thumbnail, while the playable stream is the   transcoded copy of `GET /assets/{id}/play` -- served as   `image/jpeg`.  The response is a private immutable static asset: `Cache-Control: private, max-age=31536000, immutable`, and a strong `ETag` built from the first 16 hex characters of the asset's checksum (quoted). A matching `If-None-Match` answers 304 with the same validators and no body. Conditional requests and byte ranges otherwise follow the Go standard library's `ServeContent` semantics.  The asset exists but its derived file is absent -- the thumbnail job has not run yet, or failed and is retrying -- answers 404 `ASSET_NOT_READY`: the client renders the blurhash or a placeholder and waits for the `asset.ready` event instead of treating the asset as gone. `ASSET_NOT_FOUND`, by contrast, means the id is unknown or not the caller's; the owner-only anti-enumeration of the detail endpoint applies here too, with byte-identical 404 bodies for foreign and unknown ids.  `size` must be one of the three values below; anything else (including a missing `size`) answers 400 `VALIDATION_FAILED`. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the asset.
final String size = tile; // String | The serving size: `tile` for the timeline, `view` for the inline display, `original` for the full-resolution serving copy. Video assets answer the poster frame for every value. 

try {
    final response = api.getAssetThumbnail(id, size);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->getAssetThumbnail: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Identifier of the asset. | 
 **size** | **String**| The serving size: `tile` for the timeline, `view` for the inline display, `original` for the full-resolution serving copy. Video assets answer the poster frame for every value.  | 

### Return type

[**Uint8List**](Uint8List.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: image/avif, image/jpeg, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getFavorites**
> TimelineResponse getFavorites(cursor, limit)

The caller's favorites, one keyset page at a time.

The favorites view: the assets the calling user favorites, filtered to the live view -- hidden assets, assets in the trash and the component video rows of Live Photos never appear -- with every item carrying `isFavorite: true` by construction (the listing joins the caller's favorites rows).  The view shares the timeline's shape, ordering and pagination: `takenAt` descending with the id as the tiebreak, `limit` absent meaning 100 and capped at 200, and the opaque `cursor` handed back as `nextCursor` precisely when another page follows, so the same wall UI that walks the timeline walks the favorites.  Favorites are per-user state written by the batch endpoint's favorite action. In this milestone the batch actions are owner-scoped, so favorites apply to the caller's own assets; the listing itself joins the caller's favorites rows with the live-asset filters, nothing more. Hiding (or trashing) a favorited asset removes it from this view; it returns when unhidden (or restored), the favorite row untouched -- an idempotent per-user mark, not a copy of the asset. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String cursor = AAAAAAAAAADvr_8PaGVsbw==; // String | The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
final int limit = 100; // int | Page size: the number of timeline items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400. 

try {
    final response = api.getFavorites(cursor, limit);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->getFavorites: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **cursor** | **String**| The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients).  | [optional] 
 **limit** | **int**| Page size: the number of timeline items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400.  | [optional] [default to 100]

### Return type

[**TimelineResponse**](TimelineResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getHidden**
> TimelineResponse getHidden(cursor, limit)

The caller's hidden assets, one keyset page at a time.

The hidden view: the caller's own assets whose owner-level visibility switch is off. Live rows only -- a trashed asset stays in the trash view until restored or expired, and a restored one returns here still hidden -- and no Live Photo component rows (the pair is represented by its photo). The same ordering and keyset pagination as the timeline: `takenAt` descending with the id as the tiebreak, `limit` absent meaning 100 and capped at 200, `nextCursor` exactly when another page follows.  Unhiding through the batch endpoint returns an asset to every live view (the timeline, the favorites if favorited). `isFavorite` still reports the caller's favorites state of a hidden asset -- the favorite row survives the hiding, exactly like it survives the trash. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String cursor = AAAAAAAAAADvr_8PaGVsbw==; // String | The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
final int limit = 100; // int | Page size: the number of timeline items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400. 

try {
    final response = api.getHidden(cursor, limit);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->getHidden: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **cursor** | **String**| The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients).  | [optional] 
 **limit** | **int**| Page size: the number of timeline items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400.  | [optional] [default to 100]

### Return type

[**TimelineResponse**](TimelineResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getTimeline**
> TimelineResponse getTimeline(cursor, limit, albumId)

The aggregated timeline, one keyset page at a time.

The aggregated timeline of the caller: the union of the albums that feed it -- the caller's sync album (the core backup stream, never excludable), the caller's own directory albums, and every album shared to the caller regardless of type (virtual albums included) -- minus the albums the caller turned off with the per-user \"keep out of the timeline\" preference (a missing preference row means kept in). The caller's own virtual albums never feed their own timeline: in-stream curation would be pure duplication. One asset visible through several albums appears once.  Every item is live by construction: hidden assets, assets in the trash, and the component video rows of Live Photos never appear.  `isFavorite` is per-user state: it reports whether the CALLING user favorites the asset (the favorites rows the batch endpoint writes), whatever the asset's owner -- the same item is a favorite for one viewer and a plain tile for another.  Pages are keyset-paginated with an opaque cursor: the response carries `nextCursor` exactly when another page exists, and the client passes it back as `cursor` to continue after the last item of the page. Cursors are stable for the ordering key (`takenAt` descending, `id` as the tiebreak); an asset uploaded with an older `takenAt` than the cursor appears only in a fresh pagination run, never in the current one. The first page omits the `cursor` parameter.  `limit` bounds the page size: absent means the default of 100, a value over 200 is answered with the maximum of 200, and a value below 1 answers 400 `VALIDATION_FAILED`.  `albumId` narrows the response to that album's own timeline instead of the aggregate: the album's items in the same ordering and filtering. The caller must be able to see the album (own album of any type, or an album shared to the caller); an album that does not exist or is not visible answers the empty list, not an error, so album ids cannot be probed for existence through this endpoint. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String cursor = AAAAAAAAAADvr_8PaGVsbw==; // String | The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients). 
final int limit = 100; // int | Page size: the number of timeline items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400. 
final String albumId = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Restrict the response to the timeline of this one album (an album the caller owns or that is shared to the caller) instead of the aggregate. An unknown or invisible album answers the empty list. 

try {
    final response = api.getTimeline(cursor, limit, albumId);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->getTimeline: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **cursor** | **String**| The `nextCursor` of the previous page; absent on the first page. An unparseable value answers 400 `VALIDATION_FAILED` (cursors are opaque strings produced by this endpoint, never assembled by clients).  | [optional] 
 **limit** | **int**| Page size: the number of timeline items one response carries at most. Defaults to 100; values over 200 are answered with 200, values below 1 answer 400.  | [optional] [default to 100]
 **albumId** | **String**| Restrict the response to the timeline of this one album (an album the caller owns or that is shared to the caller) instead of the aggregate. An unknown or invisible album answers the empty list.  | [optional] 

### Return type

[**TimelineResponse**](TimelineResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getTrash**
> TrashResponse getTrash()

List the caller's trashed assets with their retention countdown.

The trash view of the caller: every asset the caller soft deleted, newest deletion first, each with the countdown of whole days until the expiry job removes it. Retention is the instance setting `trash_retention_days` (default 30): the expiry job runs hourly and permanently deletes rows whose `deletedAt` fell past the retention window, so an item shows `remainingDays` 0 on its way out (the row survives only until the next job run) and disappears afterwards -- restoring it then answers 404, it does not resurrect.  `remainingDays` counts the days left rounded up: an asset trashed moments ago with the default retention shows 30. The list is owner-scoped and bounded: it answers at most the 200 most recent deletions (the trash is a recovery view for recent deletes, and older items age out through expiry long before the bound could bite). The component video rows of Live Photos never appear: the pair is represented by its photo. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();

try {
    final response = api.getTrash();
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->getTrash: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**TrashResponse**](TrashResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **patchAsset**
> AssetDetail patchAsset(id, assetPatch)

Edit the description or hidden switch of one asset.

The partial edit of the two owner-level attributes of an asset: `description`, the free text the detail panel renders, and `isHidden`, the owner-level visibility switch that keeps the asset out of every live view until it returns. Both body fields are optional but at least one must be present (a field set to JSON null counts as absent, so a body of nulls is the empty patch); a request that carries neither answers 400 `VALIDATION_FAILED`. Only the fields the request carries update -- the rest keep their values -- and the response is the full asset detail after the edit, the same shape `GET /assets/{id}` serves.  The edit follows set semantics: patching a field with the value the row already carries is the idempotent no-op -- the answer stays the 200 success with the row's current state, nothing is written (not even `updated_at`), and NO `asset.updated` event fires. A patch that actually changes either field publishes exactly ONE `asset.updated` event (the asset id and kind, after the commit): both fields are owner-level asset state, so the same announcement refreshes every viewer of the shared timeline and the detail panel. The description alone has no timeline effect -- it is detail-panel content -- but it is asset state the event covers all the same.  Access is owner-only, the attribute-editing row of the permission matrix: there is deliberately no administrator override here (unlike the permanent delete), so an id that is unknown or foreign answers the same 404 `ASSET_NOT_FOUND`, byte for byte, as every asset endpoint -- asset ids cannot be enumerated. A trashed asset is editable like it is readable (the detail answers 200 with `isTrashed: true`): the edit lands on the row and resurfaces with the restore. The hidden switch this endpoint flips is the same one the batch `hide` / `unhide` actions flip, so the two surfaces agree on the state of an asset.  `description` carries at most 10,000 Unicode characters (runes); above that the request answers 400 `VALIDATION_FAILED`. The value replaces the stored text wholesale -- an empty string is a valid patch and clears it. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the asset.
final AssetPatch assetPatch = {"description":"Sunset over the lake, our first evening there."}; // AssetPatch | 

try {
    final response = api.patchAsset(id, assetPatch);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->patchAsset: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Identifier of the asset. | 
 **assetPatch** | [**AssetPatch**](AssetPatch.md)|  | 

### Return type

[**AssetDetail**](AssetDetail.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **playAsset**
> Uint8List playAsset(id)

Stream the transcoded playable copy of one owned video.

Streams the web-safe transcoded MP4 of a video asset: the pipeline's transcode job re-encoded the original into H.264 video (yuv420p) plus AAC audio -- the pair every browser plays natively -- with the movie index relocated to the front of the container (`+faststart`), so playback begins while the file is still downloading. The stream is served inline as `video/mp4` with no `Content-Disposition`: it feeds the client's video element, while downloading the original bytes stays on `GET /assets/{id}/original` (the transcoded copy is a derivative, never the archival original).  The response is a private immutable static asset like the thumbnail: `Cache-Control: private, max-age=31536000, immutable`, and a strong `ETag` built from the first 16 hex characters of the asset's checksum (quoted) -- the transcode is a deterministic function of the immutable original bytes, so the tag is stable across re-runs of the job. A matching `If-None-Match` answers 304 with the same validators and no body.  Byte ranges follow the Go standard library's `ServeContent` semantics (video seeking needs them): a satisfiable single range answers 206 with `Content-Range`; multiple ranges answer a `multipart/byteranges` body; an unsatisfiable range answers 416 with the total size in `Content-Range`; a malformed Range header is ignored and the full body answers 200.  Only client-visible video assets have a playable stream: an image asset, and a Live Photo's component video row (linked through `livePhotoVideoId` -- it counts no transcode pipeline of its own, its pair's visible asset being the image), answer 404 `ASSET_NOT_FOUND` -- there is nothing to play by that id, and the answer is the asset's own not-found envelope, the same anti-enumeration shape (byte for byte) as an unknown or a foreign id. The component's original stays downloadable by its id through `GET /assets/{id}/original`. A video whose transcode has not been produced yet -- the pipeline is still running, or the transcode job failed and is retrying -- answers 404 `ASSET_NOT_READY`: the client renders the poster frame (the thumbnail endpoint serves it) and waits for the `asset.ready` event instead of treating the asset as gone. The video pipeline counts three jobs (metadata, thumbnail, transcode), so readiness follows all three. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the asset.

try {
    final response = api.playAsset(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->playAsset: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Identifier of the asset. | 

### Return type

[**Uint8List**](Uint8List.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: video/mp4, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **restoreAsset**
> restoreAsset(id)

Restore one trashed asset.

Clears the trash mark of one asset: the row becomes live again and re-enters the timeline (its album memberships were never removed -- a trashed asset is filtered out of the live views, not evicted from its albums). The component video row of a Live Photo is restored with its photo.  The operation follows set semantics: restoring an asset that is already live succeeds (204, nothing to do) -- a client that races another restore, or asks twice, converges instead of erroring; the row is already in the desired state. Only an id that is unknown, or that belongs to another account while the caller is not its owner, answers 404 -- byte-identical for both cases, so asset ids cannot be enumerated. An asset whose row the expiry job already removed answers 404 as well: a purge is not reversible.  The retention countdown of the trash list is the only signal that a restore is still possible; after the row is gone, the content itself can only return through a re-upload (which the upload idempotency answers as a fresh asset). 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String id = 38400000-8cf0-11bd-b23e-10b96e4ef00d; // String | Identifier of the trashed asset.

try {
    api.restoreAsset(id);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->restoreAsset: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Identifier of the trashed asset. | 

### Return type

void (empty response body)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **uploadAsset**
> AssetResponse uploadAsset(deviceAssetId, fileName, photo, deviceCreatedAt, deviceTimeZone, liveVideo)

Upload one photo, optionally with its Live Photo video.

The backup upload endpoint: one photo per request, streamed into content-addressed storage while it is read, with the Live Photo pairing video riding the same request as a second file part.  The request body is `multipart/form-data` with these parts:  - `deviceAssetId` (text, required) -- the stable identity the   uploading device assigned to the file; together with the   account it scopes the device idempotency of backups. - `fileName` (text, required) -- the original client-side file   name of the photo, recorded as the asset's   `originalFilename`. - `deviceCreatedAt` (text, optional) -- the capture time the   device reports, RFC 3339 with offset. It seeds the asset's   `takenAt` until the metadata pipeline refines it from EXIF. - `deviceTimeZone` (text, optional) -- the IANA time zone name   of the capture location; it feeds the metadata pipeline's   timezone fallback chain and is not validated at upload. - `photo` (file, required) -- the photo bytes. - `liveVideo` (file, optional) -- the pairing video of a Live   Photo; present only together with `photo`.  Part order is part of the contract: all text fields precede every file part, and the `liveVideo` part follows the `photo` part. The server streams file parts to disk without buffering them, so every field must be known and validated before the first file byte is accepted -- a file part arriving before the required fields, a text field after a file part, or a `liveVideo` without a preceding `photo` answers 400 `VALIDATION_FAILED`. Each text field value is capped at 1024 bytes; `deviceAssetId`, `fileName` and `deviceTimeZone` carry the length bounds below. The storage extension and the media kind derive from the *file part's own* file name, and the payload must be consistent with it: a renamed file of another type answers 400 `VALIDATION_FAILED`.  Uploads are idempotent through the owner-scoped checksum: re-uploading identical bytes never creates a second asset. The outcome is one of three states, discriminated by the `state` field of the response body:  - 201 `created`: no asset with this checksum existed; a new   asset was created (a Live Photo created two rows, the   response describes the photo). - 200 `duplicate`: an identical asset of the same owner is   live; that asset is returned. - 200 `trashed`: an identical asset of the same owner sits in   the trash; that asset is returned (with `isTrashed: true`)   so the client can offer the user restore or save-as-new.  Each file is capped at the instance's per-file upload limit (2 GiB by default); a file over the limit aborts mid-stream with 413 `ASSET_TOO_LARGE`. This endpoint takes multipart bodies, not JSON, and is deliberately exempt from the API-wide JSON body cap; the per-file upload limit governs instead. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getAssetsApi();
final String deviceAssetId = deviceAssetId_example; // String | Stable device-side identity of the uploaded file.
final String fileName = fileName_example; // String | Original client-side file name of the photo.
final MultipartFile photo = BINARY_DATA_HERE; // MultipartFile | The photo bytes; their file part name carries the extension.
final DateTime deviceCreatedAt = 2013-10-20T19:20:30+01:00; // DateTime | Capture time reported by the device (RFC 3339 with offset). Optional; when absent the server records the upload time until the metadata pipeline refines it. 
final String deviceTimeZone = deviceTimeZone_example; // String | IANA time zone name of the capture location, for the metadata pipeline's timezone fallback chain. Not validated at upload time. 
final MultipartFile liveVideo = BINARY_DATA_HERE; // MultipartFile | The Live Photo pairing video; present only together with `photo`. The server stores it as a linked component row that never appears in the timeline. 

try {
    final response = api.uploadAsset(deviceAssetId, fileName, photo, deviceCreatedAt, deviceTimeZone, liveVideo);
    print(response);
} on DioException catch (e) {
    print('Exception when calling AssetsApi->uploadAsset: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **deviceAssetId** | **String**| Stable device-side identity of the uploaded file. | 
 **fileName** | **String**| Original client-side file name of the photo. | 
 **photo** | **MultipartFile**| The photo bytes; their file part name carries the extension. | 
 **deviceCreatedAt** | **DateTime**| Capture time reported by the device (RFC 3339 with offset). Optional; when absent the server records the upload time until the metadata pipeline refines it.  | [optional] 
 **deviceTimeZone** | **String**| IANA time zone name of the capture location, for the metadata pipeline's timezone fallback chain. Not validated at upload time.  | [optional] 
 **liveVideo** | **MultipartFile**| The Live Photo pairing video; present only together with `photo`. The server stores it as a linked component row that never appears in the timeline.  | [optional] 

### Return type

[**AssetResponse**](AssetResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: multipart/form-data
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

