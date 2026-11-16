# lumemor_api.api.PlacesApi

## Load the API package
```dart
import 'package:lumemor_api/api.dart';
```

All URIs are relative to */api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getMapConfig**](PlacesApi.md#getmapconfig) | **GET** /map/config | The map&#39;s client-side tile configuration.
[**listPlaces**](PlacesApi.md#listplaces) | **GET** /places | The caller&#39;s city grouping.


# **getMapConfig**
> MapConfigResponse getMapConfig()

The map's client-side tile configuration.

The configuration the places map and the viewer's location card need before they can draw a base layer: the tile source the administrator configured, and nothing else.  THE POLYGON: the value is a URL the BROWSER loads directly from the tile host -- the server never fetches, proxies or caches a tile, so a pan or a zoom never makes the photo server a tile CDN. What the tile host sees is the viewport, the zoom level and the browsing device's address; the pins the map draws come from this instance's own API and never cross the tile line. A deployment whose browser cannot reach a public tile host points the setting at a self-hosted or mirrored source; there is deliberately no bundled default (a public tile service's usage policy forbids carrying the traffic of an application's whole install base).  `tilejsonUrl` is null when no source is configured. That is not an error: the map renders its honest degraded state -- the pins stay, the base layer is stated as absent -- and the admin settings face is where it gets fixed. A stored value is handed back verbatim; its reachability is the browser's business, and a source that fails to load lands on the same stated absence. 

### Example
```dart
import 'package:lumemor_api/api.dart';
// TODO Configure API key authorization: cookieAuth
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKey = 'YOUR_API_KEY';
// uncomment below to setup prefix (e.g. Bearer) for API key, if needed
//defaultApiClient.getAuthentication<ApiKeyAuth>('cookieAuth').apiKeyPrefix = 'Bearer';

final api = LumemorApi().getPlacesApi();

try {
    final response = api.getMapConfig();
    print(response);
} on DioException catch (e) {
    print('Exception when calling PlacesApi->getMapConfig: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**MapConfigResponse**](MapConfigResponse.md)

### Authorization

[cookieAuth](../README.md#cookieAuth), [bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listPlaces**
> BuiltList<PlaceSummary> listPlaces(locale)

The caller's city grouping.

The places surface: every place of the GeoNames reference data that holds at least one asset the caller can see -- the city grouping the places page renders as its grid. A place joins the list when it carries a caller-visible live asset (the album union of the caller's own sync and directory albums plus every album shared to them, and the timeline's live-row filters: not trashed, not archived, no Live Photo component row); `assetCount` counts exactly those assets, so the same place can show different counts to different accounts -- the honest per-caller counting discipline the people surface pins, applied to cities.  `name` is the locale-joined display name: the `place_names` row of the requested locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. Locale is a REQUEST parameter (the web app owns the active locale and passes it along); `en` is the default. A locale switch changes the join only -- the assets rows never carry locale strings, so switching languages never rewrites asset data.  `representativeCoverAssetIds` is the wall's derived-cover discipline applied to city groups: the place's (up to) four most recent caller-visible live members, taken_at descending with the asset id as the tiebreak -- the cover tiles the places page renders from. A listed place always carries at least one candidate (the listing itself proved a visible member exists).  `takenAtFrom` and `takenAtTo` are the place's caller-visible live members' earliest and latest capture times: the span the city card prints as its year range (\"2019-2026\", one year when the two fall in the same calendar year). A listed place always carries both.  The list is ordered by `takenAtTo` descending -- the most recently photographed city first -- with the place id ascending as the deterministic tiebreak. It is not paginated: the same single-document shape the named-people listing uses, sized for a family instance's few thousand places.  Place resolution itself (which asset belongs to which city) is the ingestion pipeline's business -- the metadata job's GPS step and the admin re-resolution run -- and follows the place data baked into the server image; this endpoint only reads what those wrote. 

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

