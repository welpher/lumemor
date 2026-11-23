# lumemor_api.model.SearchFacets

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**people** | [**BuiltList&lt;PersonFacetCount&gt;**](PersonFacetCount.md) | The caller's named people: one entry per labeled face cluster with a face on the current set.  | 
**places** | [**BuiltList&lt;PlaceFacetCount&gt;**](PlaceFacetCount.md) | The resolved places of the current set with the locale display name (`locale` parameter, the `GET /places` join).  | 
**years** | [**BuiltList&lt;YearFacetCount&gt;**](YearFacetCount.md) | The taken-at years of the current set, newest first: the time-range facet (the `from`/`to` filters are the dimension the year counts exclude).  | 
**cameras** | [**BuiltList&lt;CameraFacetCount&gt;**](CameraFacetCount.md) | The EXIF camera makes/models of the current set. The `id` is the round-trip filter value (`make|model`); `make` and `model` are the display halves, null exactly when the EXIF row carries no value there.  | 
**kinds** | [**BuiltList&lt;KindFacetCount&gt;**](KindFacetCount.md) | The media families of the current set. | 
**albums** | [**BuiltList&lt;AlbumFacetCount&gt;**](AlbumFacetCount.md) | The caller-visible albums (own and shared, never the sync albums -- the timeline is their face) holding members of the current set.  | 
**tags** | [**BuiltList&lt;TagFacetCount&gt;**](TagFacetCount.md) | The CALLER's tags with members in the current set (another account's tags never surface).  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


