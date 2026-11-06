# lumemor_api.api.PlacesApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**listPlaces**](PlacesApi.md#listplaces) | **GET** /places | The caller&#39;s city grouping.


# **listPlaces**
> BuiltList<PlaceSummary> listPlaces(locale)

The caller's city grouping.

The places surface: every place of the GeoNames reference data that holds at least one asset the caller can see -- the city grouping the places page renders as its grid. A place joins the list when it carries a caller-visible live asset (the album union of the caller's own sync and directory albums plus every album shared to them, and the timeline's live-row filters: not trashed, not hidden, no Live Photo component row); `assetCount` counts exactly those assets, so the same place can show different counts to different accounts -- the honest per-caller counting discipline the people surface pins, applied to cities.  `name` is the locale-joined display name: the `place_names` row of the requested locale when the reference data carries one, the GeoNames canonical (English or romanized) name otherwise. Locale is a REQUEST parameter (the web app owns the active locale and passes it along); `en` is the default. A locale switch changes the join only -- the assets rows never carry locale strings, so switching languages never rewrites asset data.  `representativeCoverAssetIds` is the wall's derived-cover discipline applied to city groups: the place's (up to) four most recent caller-visible live members, taken_at descending with the asset id as the tiebreak -- the cover tiles the places page renders from. A listed place always carries at least one candidate (the listing itself proved a visible member exists).  The list is ordered by `assetCount` descending, the place id ascending as the deterministic tiebreak. It is not paginated: the same single-document shape the named-people listing uses, sized for a family instance's few thousand places.  Place resolution itself (which asset belongs to which city) is the ingestion pipeline's business -- the metadata job's GPS step and the admin re-resolution run -- and follows the reference data baked into the server image; this endpoint only reads what those wrote. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPlacesApi();
final String locale = zh-Hans; // String | The display locale of the place names: `en` or `zh-Hans`, the instance's UI language set. Absent means `en`. A value outside the set answers 400 `VALIDATION_FAILED`. 

try {
    final response = api.listPlaces(locale);
    print(response);
} on DioException catch (e) {
    print('Exception when calling PlacesApi->listPlaces: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **locale** | **String**| The display locale of the place names: `en` or `zh-Hans`, the instance's UI language set. Absent means `en`. A value outside the set answers 400 `VALIDATION_FAILED`.  | [optional] [default to 'en']

### Return type

[**BuiltList&lt;PlaceSummary&gt;**](PlaceSummary.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

