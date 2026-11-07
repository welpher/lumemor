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
**type** | **String** | The album's type; `sync` is the caller's own default album -- a read-only, system-managed detail (no write affordance applies, the display name is the client's localized one keyed by the type). Another account's sync album never reaches this shape: it answers the single-shape 404.  | 
**locked** | **bool** | The owner's lock (metadata to a viewer). | 
**coverAssetId** | **String** | The pinned cover asset, or null when unset -- the raw column; the wall derives its collage fallback, the detail carries the pin itself.  | 
**callerRole** | **String** | The caller's relation to this album: the owner holds the write affordances, a viewer is read-only.  | 
**itemCount** | **int** | The album's live member count, the same predicate as the items below.  | 
**items** | [**BuiltList&lt;AlbumItem&gt;**](AlbumItem.md) | One page of the album's live members, `takenAt` descending with the asset id as the tiebreak.  | 
**nextCursor** | **String** | The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page.  | [optional] 
**shares** | [**BuiltList&lt;AlbumShareEntry&gt;**](AlbumShareEntry.md) | The album's grantee list, riding ONLY the owner's view: one entry per `album_shares` row in the grant order (`grantedAt` ascending) -- the share management dialog's list. The owner's response always carries the field, an empty array when the album holds no grants; viewer responses never carry it at all.  | [optional] 
**inTimeline** | **bool** | The caller's timeline preference of this album: whether it feeds their aggregated timeline. Present exactly when the album is excludable for the caller -- an own `directory` album or an album shared to them; an own `virtual` album omits the field (it never feeds the caller's own timeline), and the caller's own `sync` album omits it too (the core stream is never excludable). The value is the preference row's switch, true when no row exists (the default). The preference scopes the aggregated timeline only: it never filters this album's own items, whatever the switch says.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


