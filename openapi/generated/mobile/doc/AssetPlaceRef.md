# lumemor_api.model.AssetPlaceRef

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**placeId** | **int** | The GeoNames geonameid of the place -- the `places` row's primary key and the `assets.place_id` foreign key, the same identifier the place-filtered browse filters on.  | 
**name** | **String** | The display name at the request's locale: the `place_names` row of that locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


