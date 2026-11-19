# lumemor_api.model.PlacesDatabaseResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**version** | **String** | The data-version marker the startup import recorded -- the sha256 digest of the exact loaded bytes. An empty string means the instance carries no bake (nothing resolves). A 64-character value is a content digest, not a release date: the card displays its first eight characters.  | 
**languages** | **BuiltList&lt;String&gt;** | The display-name locales the bake carries, in the product's language-set order. The places read's `locale` parameter accepts exactly this set.  | 
**gpsAssets** | **int** | The full re-resolution scope: every asset whose exif row carries GPS coordinates, whatever its lifecycle state. A row with a manual location is excluded -- the walk leaves those rows to the owner's own decision, so the count and the walk cover the same rows.  | 
**missingPlaces** | **int** | The fill-in scope: the GPS-bearing assets whose place is absent (the subset a `missing` run walks), with the same manual-location exclusion as `gpsAssets`.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


