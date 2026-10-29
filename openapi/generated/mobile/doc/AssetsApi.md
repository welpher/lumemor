# lumemor_api.api.AssetsApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getAssetByDeviceId**](AssetsApi.md#getassetbydeviceid) | **GET** /assets/by-device-id/{deviceAssetId} | Look up an uploaded asset by its device identifier.
[**uploadAsset**](AssetsApi.md#uploadasset) | **POST** /assets | Upload one photo, optionally with its Live Photo video.


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

