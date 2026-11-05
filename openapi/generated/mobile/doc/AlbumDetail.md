# lumemor_api.model.AlbumDetail

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **String** | Identifier of the album. | 
**title** | **String** | The album's display title. | 
**description** | **String** | The album's free-text description (empty when unset). | 
**type** | **String** | The album's type; sync albums never reach this shape. | 
**locked** | **bool** | The owner's lock (metadata to a viewer). | 
**coverAssetId** | **String** | The pinned cover asset, or null when unset -- the raw column; the wall derives its collage fallback, the detail carries the pin itself.  | 
**callerRole** | **String** | The caller's relation to this album: the owner holds the write affordances, a viewer is read-only.  | 
**itemCount** | **int** | The album's live member count, the same predicate as the items below.  | 
**items** | [**BuiltList&lt;AlbumItem&gt;**](AlbumItem.md) | One page of the album's live members, `takenAt` descending with the asset id as the tiebreak.  | 
**nextCursor** | **String** | The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


