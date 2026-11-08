# lumemor_api.model.AlbumAudienceResponse

## Load the model package
```dart
import 'package:lumemor_api/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**albumId** | **String** | Identifier of the album the audience belongs to. | 
**audience** | [**BuiltList&lt;AlbumShareEntry&gt;**](AlbumShareEntry.md) | The grantee list after the write, one entry per `album_shares` row in the grant order (`grantedAt` ascending) -- the same list the owner's album detail carries as `shares`, the edit dialog's chips.  | 
**granted** | **BuiltList&lt;String&gt;** | The ids whose rows THIS write landed -- empty for the idempotent repeat of the stored set. One `album.shared` event fired per entry.  | 
**revoked** | **BuiltList&lt;String&gt;** | The ids whose rows THIS write removed -- empty for the idempotent repeat. One `album.unshared` event fired per entry.  | 
**skipped** | [**BuiltList&lt;AlbumAudienceSkip&gt;**](AlbumAudienceSkip.md) | The requested ids that stayed ungranted, each with its reason. Empty when the whole request was actionable.  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


