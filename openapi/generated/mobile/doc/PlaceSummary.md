# lumemor_api.model.PlaceSummary

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**placeId** | **int** | The GeoNames geonameid of the place -- the `places` row's primary key and the `assets.place_id` foreign key the ingestion pipeline's resolution writes.  | 
**name** | **String** | The display name in the request's locale: the `place_names` row of that locale when the place data carries one, the GeoNames canonical (English or romanized) name otherwise. A locale switch changes this join only, never an asset row.  | 
**country** | **String** | The place's ISO 3166-1 alpha-2 country code. The display country name is the client's business (its own locale table maps the code).  | 
**assetCount** | **int** | The place's caller-visible live assets -- the honest per-caller count, not the global member count.  | 
**latitude** | **double** | The place's latitude in degrees (the GeoNames record's own coordinate, never a photo's). It is the point the places map pins and clusters -- the map layer is a view over the same grouping, so no asset coordinate ever needs to leave the instance for it.  | 
**longitude** | **double** | The place's longitude in degrees (the GeoNames record's own coordinate, never a photo's).  | 
**takenAtFrom** | [**DateTime**](DateTime.md) | The earliest capture time among the place's caller-visible live members (RFC 3339 UTC). The city card's year span starts here; the listing's own ordering reads the other end.  | 
**takenAtTo** | [**DateTime**](DateTime.md) | The latest capture time among the place's caller-visible live members (RFC 3339 UTC). It is the ordering key of the list -- the most recently photographed city leads -- and the city card's year span ends here. Equal to `takenAtFrom` when the place holds one member.  | 
**representativeCoverAssetIds** | **BuiltList&lt;String&gt;** | The place's cover candidates: its (up to) four most recent caller-visible live members, taken_at descending with the asset id as the tiebreak -- the wall's derived-cover discipline applied to city groups, newest first.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


