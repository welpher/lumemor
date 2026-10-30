# lumemor_api.api.AssetsApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getAsset**](AssetsApi.md#getasset) | **GET** /assets/{id} | Fetch the detail of one owned asset.
[**getAssetByDeviceId**](AssetsApi.md#getassetbydeviceid) | **GET** /assets/by-device-id/{deviceAssetId} | Look up an uploaded asset by its device identifier.
[**getAssetOriginal**](AssetsApi.md#getassetoriginal) | **GET** /assets/{id}/original | Download the original file of one owned asset.
[**getAssetThumbnail**](AssetsApi.md#getassetthumbnail) | **GET** /assets/{id}/thumbnail | Serve a derived thumbnail, or a video&#39;s poster frame.
[**getTimeline**](AssetsApi.md#gettimeline) | **GET** /assets/timeline | The aggregated timeline, one keyset page at a time.
[**uploadAsset**](AssetsApi.md#uploadasset) | **POST** /assets | Upload one photo, optionally with its Live Photo video.


# **getAsset**
> AssetDetail getAsset(id)

Fetch the detail of one owned asset.

The asset detail: the identity and storage fields of the upload response plus the refinements the pipeline produced afterwards -- dimensions, duration, blurhash, the EXIF hot fields, and readiness -- which the asset detail view renders.  Access is owner-only: an id that belongs to another account answers the same 404 `ASSET_NOT_FOUND` as an id that does not exist, byte for byte, so asset ids cannot be enumerated. Album sharing and ACLs widen access for shared content in a later milestone (M2c, with the album domain); until then only the owner's own assets answer.  A trashed asset answers 200 with `isTrashed: true` -- the trash view reads the detail the same way. The component video row of a Live Photo is readable by its id (the future playback endpoint resolves it through `livePhotoVideoId`) but never appears in the timeline. 

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

The static serving copy of the asset:  - image assets: the AVIF thumbnail of the requested size --   `tile` (fits the 250px timeline box), `view` (long edge   capped at 1080px), `original` (the full-resolution   re-encode) -- served as `image/avif`; - video assets: the poster frame, whatever `size` says -- the   poster (the first frame, scaled into the tile box) is the   video's only serving copy until transcodes exist -- served   as `image/jpeg`.  The response is a private immutable static asset: `Cache-Control: private, max-age=31536000, immutable`, and a strong `ETag` built from the first 16 hex characters of the asset's checksum (quoted). A matching `If-None-Match` answers 304 with the same validators and no body. Conditional requests and byte ranges otherwise follow the Go standard library's `ServeContent` semantics.  The asset exists but its derived file is absent -- the thumbnail job has not run yet, or failed and is retrying -- answers 404 `ASSET_NOT_READY`: the client renders the blurhash or a placeholder and waits for the `asset.ready` event instead of treating the asset as gone. `ASSET_NOT_FOUND`, by contrast, means the id is unknown or not the caller's; the owner-only anti-enumeration of the detail endpoint applies here too, with byte-identical 404 bodies for foreign and unknown ids.  `size` must be one of the three values below; anything else (including a missing `size`) answers 400 `VALIDATION_FAILED`. 

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

# **getTimeline**
> TimelineResponse getTimeline(cursor, limit, albumId)

The aggregated timeline, one keyset page at a time.

The aggregated timeline of the caller: the union of the albums that feed it -- the caller's sync album (the core backup stream, never excludable), the caller's own directory albums, and every album shared to the caller regardless of type (virtual albums included) -- minus the albums the caller turned off with the per-user \"keep out of the timeline\" preference (a missing preference row means kept in). The caller's own virtual albums never feed their own timeline: in-stream curation would be pure duplication. One asset visible through several albums appears once.  Every item is live by construction: hidden assets, assets in the trash, and the component video rows of Live Photos never appear.  Pages are keyset-paginated with an opaque cursor: the response carries `nextCursor` exactly when another page exists, and the client passes it back as `cursor` to continue after the last item of the page. Cursors are stable for the ordering key (`takenAt` descending, `id` as the tiebreak); an asset uploaded with an older `takenAt` than the cursor appears only in a fresh pagination run, never in the current one. The first page omits the `cursor` parameter.  `limit` bounds the page size: absent means the default of 100, a value over 200 is answered with the maximum of 200, and a value below 1 answers 400 `VALIDATION_FAILED`.  `albumId` narrows the response to that album's own timeline instead of the aggregate: the album's items in the same ordering and filtering. The caller must be able to see the album (own album of any type, or an album shared to the caller); an album that does not exist or is not visible answers the empty list, not an error, so album ids cannot be probed for existence through this endpoint. 

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

