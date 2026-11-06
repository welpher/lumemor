# lumemor_api.model.SearchResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**BuiltList&lt;TimelineItem&gt;**](TimelineItem.md) | The page's assets, ordered by the fused score descending with the asset id ascending as the tiebreak (a text query), or by taken-at descending with the id descending (the filter browse -- the timeline's own order).  | 
**nextCursor** | **String** | The continuation token of the page: present exactly when another page follows. Opaque; pass it back as `cursor`.  | [optional] 
**facets** | [**SearchFacets**](SearchFacets.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


